package middleware

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/repository/sqlc"
)

// RateLimit returns middleware that enforces per-minute rate limits.
func RateLimit(queries *sqlc.Queries) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			// Only rate limit messages (not callbacks or other updates)
			if update.Message == nil {
				next(ctx, b, update)
				return
			}

			chatID := update.Message.Chat.ID

			// Check rate limit
			count, err := queries.CheckAndIncrementRateLimit(ctx, chatID)
			if err != nil {
				slog.Error("rate limit check failed", "error", err, "chat_id", chatID)
				next(ctx, b, update)
				return
			}

			// Default limit for unknown users
			limit := int64(config.RateLimitRegular)

			// TODO: Check if premium user and use RateLimitPremium
			// This requires user context which is loaded after this middleware
			// For now, use the regular limit

			if int64(count) > limit {
				slog.Debug("rate limited", "chat_id", chatID, "count", count, "limit", limit)
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "⏳ Слишком много запросов. Подождите немного.",
				})
				return
			}

			next(ctx, b, update)
		}
	}
}
