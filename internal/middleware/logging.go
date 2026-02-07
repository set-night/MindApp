package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Logging returns middleware that logs update processing time.
func Logging() bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			start := time.Now()

			updateType := "unknown"
			var chatID int64
			var userID int64

			if update.Message != nil {
				updateType = "message"
				chatID = update.Message.Chat.ID
				if update.Message.From != nil {
					userID = update.Message.From.ID
				}
			} else if update.CallbackQuery != nil {
				updateType = "callback_query"
				if update.CallbackQuery.Message.Message != nil {
					chatID = update.CallbackQuery.Message.Message.Chat.ID
				}
				userID = update.CallbackQuery.From.ID
			} else if update.PreCheckoutQuery != nil {
				updateType = "pre_checkout_query"
				userID = update.PreCheckoutQuery.From.ID
			}

			next(ctx, b, update)

			slog.Debug("update processed",
				"type", updateType,
				"chat_id", chatID,
				"user_id", userID,
				"duration", time.Since(start),
			)
		}
	}
}
