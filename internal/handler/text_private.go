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

// HandleTextPrivate processes private text messages for AI requests.
func (h *Handler) HandleTextPrivate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Chat.Type != "private" {
		return
	}

	msg := update.Message

	// Skip commands
	if strings.HasPrefix(msg.Text, "/") {
		return
	}

	// Skip successful payments
	if msg.SuccessfulPayment != nil {
		h.HandleSuccessfulPayment(ctx, b, update)
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := msg.Chat.ID

	// 1. Check active request
	_, err := h.queries.TrySetActiveRequest(ctx, chatID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "⏳ Дождитесь ответа на предыдущий запрос.",
		})
		return
	}
	defer h.queries.RemoveActiveRequest(ctx, chatID)

	// 2. Get model info
	model, err := h.openRouter.GetModel(ctx, user.SelectedModel)
	if err != nil {
		slog.Error("get model", "error", err, "model", user.SelectedModel)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Модель не найдена. Используйте /models для выбора.",
		})
		return
	}

	// 3. Check balance for paid models
	if !model.IsFree() {
		if user.Balance.LessThan(decimal.Zero) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Недостаточно средств. Пополните баланс: /pay",
			})
			return
		}
		// Low balance + expensive model check
		if !user.IsPremium() {
			avgPrice := (model.PromptPrice + model.CompletionPrice) / 2 / 1_000_000
			if user.Balance.InexactFloat64() < config.LowBalanceThreshold && avgPrice > config.LowPriceThreshold {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "❌ Баланс слишком мал для этой модели. Выберите бесплатную модель или пополните баланс.",
				})
				return
			}
		}
	}

	// 4. Check cooldown
	cooldown := config.CooldownRegular
	if user.IsPremium() {
		cooldown = config.CooldownPremium
	} else if model.IsFree() && user.Balance.InexactFloat64() < config.LowBalanceThreshold {
		cooldown = config.CooldownFree
	}

	timeSinceLast := time.Since(user.LastInteraction)
	if timeSinceLast < cooldown {
		remaining := cooldown - timeSinceLast
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("⏳ Подождите %d секунд.", int(remaining.Seconds())+1),
		})
		return
	}

	// 5. Update last interaction
	h.userService.UpdateLastInteraction(ctx, user.ID)

	// 6. Handle session
	// Auto-reset if timeout
	if h.sessionService.IsExpired(user) {
		h.sessionService.Reset(ctx, user)
	}
	// Auto-reset if context disabled
	if !user.ContextEnabled {
		h.sessionService.Reset(ctx, user)
	}

	session, err := h.sessionService.FindOrCreate(ctx, user)
	if err != nil {
		slog.Error("find or create session", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при создании сессии.",
		})
		return
	}

	// Update session model if different
	if session.Model != user.SelectedModel {
		h.queries.UpdateSessionModel(ctx, sqlc.UpdateSessionModelParams{
			ID:    session.ID,
			Model: user.SelectedModel,
		})
		session.Model = user.SelectedModel
	}

	// 7. Check message limit
	maxMessages := config.MaxMessagesRegular
	if user.IsPremium() {
		maxMessages = config.MaxMessagesPremium
	}

	msgCount, err := h.sessionService.CountMessages(ctx, session.ID)
	if err != nil {
		slog.Error("count messages", "error", err)
		return
	}

	if msgCount >= int64(maxMessages) {
		session, err = h.sessionService.Reset(ctx, user)
		if err != nil {
			slog.Error("reset session on limit", "error", err)
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("📝 Достигнут лимит сообщений (%d). Контекст сброшен.", maxMessages),
		})
	}

	// 8. Process files/images
	var fileURLs []string
	if msg.Photo != nil && len(msg.Photo) > 0 {
		// Get highest resolution photo
		photo := msg.Photo[len(msg.Photo)-1]
		url, err := tg.GetFileURL(ctx, b, photo.FileID)
		if err == nil {
			fileURLs = append(fileURLs, url)
		}
	}
	if msg.Document != nil {
		url, err := tg.GetFileURL(ctx, b, msg.Document.FileID)
		if err == nil {
			fileURLs = append(fileURLs, url)
		}
	}

	// 9. Build messages for AI
	history, err := h.sessionService.GetMessages(ctx, session.ID)
	if err != nil {
		slog.Error("get session messages", "error", err)
		return
	}

	var chatMessages []service.ChatMessage
	for _, m := range history {
		chatMessages = append(chatMessages, service.ChatMessage{
			Role:    m.Role,
			Content: m.Text,
		})
	}

	// Add current user message
	userText := msg.Text
	if msg.Caption != "" {
		userText = msg.Caption
	}
	if userText == "" {
		userText = "[File]"
	}

	// Build content with images if present
	var userContent interface{} = userText
	if len(fileURLs) > 0 {
		parts := []interface{}{
			map[string]interface{}{"type": "text", "text": userText},
		}
		for _, url := range fileURLs {
			parts = append(parts, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]string{"url": url},
			})
		}
		userContent = parts
	}

	chatMessages = append(chatMessages, service.ChatMessage{
		Role:    "user",
		Content: userContent,
	})

	// 10. Send typing indicator (repeats every 4s until stopped)
	stopTyping := tg.StartTyping(ctx, b, chatID)
	defer stopTyping()

	statusText := "⏳ Обрабатываю запрос..."
	if model.IsFree() {
		statusText = "⏳ Обрабатываю запрос...\n\n⚠️ Бесплатная модель — ответ может занять больше времени."
	}
	statusMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   statusText,
	})

	// 11. Call OpenRouter API
	reqCtx, cancel := context.WithTimeout(ctx, config.RequestTimeout)
	defer cancel()

	var temperature *float64
	temp := user.Temperature
	temperature = &temp

	aiResp, err := h.openRouter.Chat(reqCtx, chatMessages, user.SelectedModel, temperature)
	if err != nil {
		slog.Error("openrouter chat", "error", err)
		errText := "❌ Ошибка при обработке запроса."
		if strings.Contains(err.Error(), "429") {
			errText = "⏳ Слишком много запросов к AI. Попробуйте позже."
		} else if strings.Contains(err.Error(), "503") {
			errText = "❌ Сервис AI временно недоступен."
		} else if reqCtx.Err() != nil {
			errText = "⏳ Превышено время ожидания ответа."
		}
		if statusMsg != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      errText,
			})
		}
		return
	}

	if len(aiResp.Choices) == 0 {
		if statusMsg != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      "❌ AI не вернул ответ.",
			})
		}
		return
	}

	responseText := aiResp.Choices[0].Message.Content

	// 12. Calculate cost
	markupPercent := h.cfg.MarkupPercentNormal
	if user.IsPremium() {
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

		// Use API-provided total_cost if available
		if aiResp.Usage.TotalCost > 0 {
			baseCost := decimal.NewFromFloat(aiResp.Usage.TotalCost)
			markup := decimal.NewFromFloat(1 + markupPercent/100)
			totalCost = baseCost.Mul(markup)
		}

		// 13. Process transaction
		negCost := totalCost.Neg()
		newBalance, err = h.queries.UpdateUserBalanceWithCheck(ctx, sqlc.UpdateUserBalanceWithCheckParams{
			ID:      user.ID,
			Balance: negCost,
		})
		if err != nil {
			slog.Error("deduct balance", "error", err)
			if statusMsg != nil {
				b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:    chatID,
					MessageID: statusMsg.ID,
					Text:      "❌ Недостаточно средств для этого запроса.",
				})
			}
			return
		}

		// Record transaction
		h.queries.CreateTransaction(ctx, sqlc.CreateTransactionParams{
			UserID:      &user.ID,
			Amount:      negCost,
			TxType:      string(domain.TxTypeDebit),
			Description: fmt.Sprintf("AI request: %s", user.SelectedModel),
		})
	}

	// 14. Save messages to session
	h.sessionService.AddMessage(ctx, session.ID, "user", userText, fileURLs, false)
	h.sessionService.AddMessage(ctx, session.ID, "assistant", responseText, nil, false)

	// 15. Delete status message
	if statusMsg != nil {
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
		})
	}

	// 16. Send response
	tg.SendLongMessage(ctx, b, chatID, responseText, nil)

	// 17. Show cost if enabled
	if user.ShowCost && !model.IsFree() {
		costText := fmt.Sprintf(
			"💰 Стоимость: $%.6f | Баланс: $%.4f\n📊 Токены: %d→%d",
			totalCost.InexactFloat64(),
			newBalance.InexactFloat64(),
			aiResp.Usage.PromptTokens,
			aiResp.Usage.CompletionTokens,
		)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   costText,
		})
	}
}
