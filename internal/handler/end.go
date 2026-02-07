package handler

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/middleware"
)

func (h *Handler) handleEnd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	chatType := update.Message.Chat.Type

	if chatType == "private" {
		user := middleware.GetUser(ctx)
		if user == nil {
			return
		}
		_, err := h.sessionService.Reset(ctx, user)
		if err != nil {
			slog.Error("reset session", "error", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Ошибка при сбросе сессии.",
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "🔄 Контекст сброшен. Начат новый диалог.",
		})
	} else {
		group := middleware.GetGroup(ctx)
		if group == nil {
			return
		}
		if err := h.groupService.ClearContext(ctx, group.ID); err != nil {
			slog.Error("clear group context", "error", err)
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "🔄 Контекст группы сброшен.",
		})
	}
}
