package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/middleware"
)

func (h *Handler) handlePromo(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := update.Message.Chat.ID
	parts := strings.Fields(update.Message.Text)

	if len(parts) < 2 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Используйте: /promo <код>",
		})
		return
	}

	code := parts[1]
	amount, err := h.promoService.Activate(ctx, code, user.ID)
	if err != nil {
		var msg string
		switch err {
		case domain.ErrPromoNotFound:
			msg = "❌ Промокод не найден."
		case domain.ErrPromoAlreadyUsed:
			msg = "❌ Вы уже активировали этот промокод."
		case domain.ErrPromoMaxUses:
			msg = "❌ Промокод больше недействителен."
		default:
			msg = "❌ Ошибка при активации промокода."
			slog.Error("promo activation", "error", err)
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   msg,
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ Промокод активирован! Начислено *$%.2f*", amount.InexactFloat64()),
		ParseMode: models.ParseModeMarkdown,
	})

	h.tgLogger.LogPromoActivate(user.TelegramID, code, amount.InexactFloat64())
}
