package middleware

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Recover returns middleware that recovers from panics.
func Recover() bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic recovered in handler",
						"panic", r,
						"stack", string(debug.Stack()),
					)
				}
			}()
			next(ctx, b, update)
		}
	}
}
