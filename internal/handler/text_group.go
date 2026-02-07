package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/set-night/mindapp/internal/service"
	tg "github.com/set-night/mindapp/internal/telegram"
	"github.com/shopspring/decimal"
)

// HandleTextGroup processes supergroup text messages for AI requests.
func (h *Handler) HandleTextGroup(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	chatType := msg.Chat.Type

	if chatType != "supergroup" && chatType != "group" {
		return
	}

	// Skip commands
	if strings.HasPrefix(msg.Text, "/") {
		return
	}

	group := middleware.GetGroup(ctx)
	user := middleware.GetUser(ctx)
	if group == nil || user == nil {
		return
	}

	chatID := msg.Chat.ID

	// Check thread ID binding
	if group.ThreadID != nil {
		if msg.MessageThreadID != *group.ThreadID {
			return
		}
	}

	// 1. Check active request
	_, err := h.queries.TrySetActiveRequest(ctx, chatID)
	if err != nil {
		return // Silently skip in groups
	}
	defer h.queries.RemoveActiveRequest(ctx, chatID)

	// 2. Get model info
	model, err := h.openRouter.GetModel(ctx, group.SelectedModel)
	if err != nil {
		slog.Error("get group model", "error", err, "model", group.SelectedModel)
		return
	}

	// 3. Check group balance for paid models
	if !model.IsFree() {
		if group.Balance.LessThan(decimal.Zero) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Баланс группы исчерпан. Админ может пополнить: /pay <сумма>",
			})
			return
		}
	}

	// 4. Check cooldown
	cooldown := config.CooldownRegular
	if group.IsPremium() {
		cooldown = config.CooldownPremium
	} else if model.IsFree() {
		cooldown = config.CooldownFree
	}

	timeSinceLast := time.Since(group.LastInteraction)
	if timeSinceLast < cooldown {
		return // Silently skip in groups
	}

	// 5. Update last interaction
	h.queries.UpdateGroupLastInteraction(ctx, group.ID)

	// 6. Build messages from context
	var chatMessages []service.ChatMessage

	if group.ContextEnabled {
		contextMsgs, err := h.groupService.GetContextMessages(ctx, group.ID)
		if err != nil {
			slog.Error("get group context", "error", err)
		} else {
			for _, m := range contextMsgs {
				chatMessages = append(chatMessages, service.ChatMessage{
					Role:    m.Role,
					Content: m.Text,
				})
			}
		}
	}

	// Add current message
	userText := msg.Text
	if msg.Caption != "" {
		userText = msg.Caption
	}
	if userText == "" {
		return
	}

	// Add user info prefix if enabled
	senderName := ""
	if msg.From != nil {
		senderName = msg.From.FirstName
		if msg.From.Username != "" {
			senderName += fmt.Sprintf(" (@%s)", msg.From.Username)
		}
	}
	if senderName != "" {
		userText = fmt.Sprintf("[%s]: %s", senderName, userText)
	}

	chatMessages = append(chatMessages, service.ChatMessage{
		Role:    "user",
		Content: userText,
	})

	// 7. Send typing indicator
	b.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: chatID,
		Action: models.ChatActionTyping,
	})

	// 8. Call OpenRouter
	reqCtx, cancel := context.WithTimeout(ctx, config.RequestTimeout)
	defer cancel()

	aiResp, err := h.openRouter.Chat(reqCtx, chatMessages, group.SelectedModel, nil)
	if err != nil {
		slog.Error("openrouter group chat", "error", err)
		return
	}

	if len(aiResp.Choices) == 0 {
		return
	}

	responseText := aiResp.Choices[0].Message.Content

	// 9. Calculate cost and process transaction
	markupPercent := h.cfg.MarkupPercentNormal
	if group.IsPremium() {
		markupPercent = h.cfg.MarkupPercentPremium
	}

	totalCost := decimal.Zero
	var newBalance decimal.Decimal

	if !model.IsFree() {
		totalCost = service.CalculateCost(
			aiResp.Usage.PromptTokens,
			aiResp.Usage.CompletionTokens,
			model.PromptPrice,
			model.CompletionPrice,
			markupPercent,
		)

		if aiResp.Usage.TotalCost > 0 {
			baseCost := decimal.NewFromFloat(aiResp.Usage.TotalCost)
			markup := decimal.NewFromFloat(1 + markupPercent/100)
			totalCost = baseCost.Mul(markup)
		}

		_, newBalance, err = h.billingService.ProcessGroupTransaction(
			ctx, group.ID,
			totalCost.InexactFloat64()/(1+markupPercent/100),
			markupPercent,
			fmt.Sprintf("AI request: %s", group.SelectedModel),
		)
		if err != nil {
			if err == domain.ErrInsufficientBalance {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "❌ Недостаточно средств на балансе группы.",
				})
			}
			return
		}
	}

	// 10. Save to group context
	if group.ContextEnabled {
		h.groupService.AddContextMessage(ctx, group.ID, "user", userText)
		h.groupService.AddContextMessage(ctx, group.ID, "assistant", responseText)

		// Limit context size (keep last 20 messages)
		count, _ := h.queries.CountGroupContextMessages(ctx, group.ID)
		if count > 20 {
			toDelete := count - 20
			h.queries.DeleteOldestGroupContextMessages(ctx, sqlc.DeleteOldestGroupContextMessagesParams{
				GroupID: group.ID,
				Limit:   int32(toDelete),
			})
		}
	}

	// 11. Send response
	replyToID := msg.ID
	tg.SendLongMessage(ctx, b, chatID, responseText, &replyToID)

	// 12. Show cost if enabled
	if group.ShowCost && !model.IsFree() {
		costText := fmt.Sprintf(
			"💰 $%.6f | Баланс: $%.4f",
			totalCost.InexactFloat64(),
			newBalance.InexactFloat64(),
		)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   costText,
		})
	}
}
