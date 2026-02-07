package handler

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/middleware"
)

func (h *Handler) handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatType := update.Message.Chat.Type
	chatID := update.Message.Chat.ID

	if chatType == "private" {
		h.handleStartPrivate(ctx, b, update)
	} else if chatType == "supergroup" || chatType == "group" {
		h.handleStartGroup(ctx, b, update, chatID)
	}
}

func (h *Handler) handleStartPrivate(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := update.Message.Chat.ID

	// Parse deep link payload
	text := update.Message.Text
	parts := strings.SplitN(text, " ", 2)
	if len(parts) > 1 {
		payload := parts[1]

		switch {
		case strings.HasPrefix(payload, "r_"):
			// Referral link - already handled during FindOrCreate in middleware

		case strings.HasPrefix(payload, "p_"):
			// Promo code activation via deep link
			code := strings.TrimPrefix(payload, "p_")
			amount, err := h.promoService.Activate(ctx, code, user.ID)
			if err != nil {
				slog.Error("promo activation failed", "error", err)
			} else {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   fmt.Sprintf("✅ Промокод активирован! Начислено $%.2f", amount.InexactFloat64()),
				})
				h.tgLogger.LogPromoActivate(user.TelegramID, code, amount.InexactFloat64())
			}

		case strings.HasPrefix(payload, "s_"):
			// System prompt activation via deep link
			h.activatePrompt(ctx, b, chatID, user, strings.TrimPrefix(payload, "s_"))
			return
		}
	}

	// Send welcome message
	welcomeText := fmt.Sprintf(
		"👋 Привет, *%s*!\n\n"+
			"Я — AI-ассистент с поддержкой множества моделей.\n\n"+
			"📋 *Команды:*\n"+
			"/models — Выбрать AI-модель\n"+
			"/sessions — Управление сессиями\n"+
			"/settings — Настройки\n"+
			"/favorite — Избранные модели\n"+
			"/pay — Пополнить баланс\n"+
			"/premium — Премиум подписка\n"+
			"/referral — Реферальная программа\n"+
			"/prompt — Системные промпты\n"+
			"/end — Сбросить контекст\n\n"+
			"Просто отправьте сообщение, чтобы начать диалог!",
		user.FirstName,
	)

	// Try to send welcome image
	imgPath := "assets/Welcome.png"
	if _, err := os.Stat(imgPath); err == nil {
		photoData, err := os.ReadFile(imgPath)
		if err == nil {
			_, sendErr := b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:    chatID,
				Photo:     &models.InputFileUpload{Filename: "Welcome.png", Data: bytes.NewReader(photoData)},
				Caption:   welcomeText,
				ParseMode: models.ParseModeMarkdown,
			})
			if sendErr == nil {
				return
			}
			slog.Warn("failed to send welcome photo, falling back to text", "error", sendErr)
		}
	}

	// Fallback: text only
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      welcomeText,
		ParseMode: models.ParseModeMarkdown,
	})
}

func (h *Handler) handleStartGroup(ctx context.Context, b *bot.Bot, update *models.Update, chatID int64) {
	group := middleware.GetGroup(ctx)
	if group == nil {
		return
	}

	text := "👋 Привет! Я AI-ассистент для этой группы.\n\n" +
		"📋 *Команды:*\n" +
		"/models — Выбрать модель (админ)\n" +
		"/settings — Настройки (админ)\n" +
		"/pay <сумма> — Перевести на баланс группы\n" +
		"/end — Сбросить контекст\n\n" +
		"Просто напишите сообщение для общения с AI!"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}

// activatePrompt resets the current session and applies a system prompt by ID.
func (h *Handler) activatePrompt(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, promptIDStr string) {
	promptID, err := strconv.ParseInt(promptIDStr, 10, 64)
	if err != nil {
		return
	}

	prompt, err := h.queries.GetPromptByID(ctx, promptID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Промпт не найден.",
		})
		return
	}

	// Reset session and create a new one with the system prompt
	session, err := h.sessionService.Reset(ctx, user)
	if err != nil {
		slog.Error("reset session for prompt", "error", err)
		return
	}

	// Gemini models don't support "system" role; use "user" instead
	role := "system"
	if strings.Contains(strings.ToLower(session.Model), "gemini") {
		role = "user"
	}

	_, err = h.sessionService.AddMessage(ctx, session.ID, role, prompt.PromptText, nil, true)
	if err != nil {
		slog.Error("add system prompt message", "error", err)
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ Промпт *%s* активирован!\n\n_%s_", prompt.Title, prompt.Description),
		ParseMode: models.ParseModeMarkdown,
	})
}
