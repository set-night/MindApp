package handler

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/middleware"
)

func (h *Handler) handleReferral(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := update.Message.Chat.ID
	refLink := fmt.Sprintf("https://t.me/%s?start=r_%s", h.botUsername, user.ReferralCode)

	text := fmt.Sprintf(
		"👥 *Реферальная программа*\n\n"+
			"Ваша реферальная ссылка:\n`%s`\n\n"+
			"💰 Реферальный баланс: *$%.2f*\n\n"+
			"За каждого приглашённого пользователя вы получаете бонус!",
		refLink,
		user.ReferralBalance.InexactFloat64(),
	)

	// Try to send with image
	imgPath := "assets/Partners.png"
	if _, err := os.Stat(imgPath); err == nil {
		photoData, err := os.ReadFile(imgPath)
		if err == nil {
			_, sendErr := b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:    chatID,
				Photo:     &models.InputFileUpload{Filename: "Partners.png", Data: bytes.NewReader(photoData)},
				Caption:   text,
				ParseMode: models.ParseModeMarkdown,
			})
			if sendErr == nil {
				return
			}
			slog.Warn("failed to send referral photo, falling back to text", "error", sendErr)
		}
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}
