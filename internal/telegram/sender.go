package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const MaxMessageLen = 4096

// SendLongMessage sends a potentially long message, splitting it into parts if needed.
// Falls back to plain text if Markdown parsing fails.
func SendLongMessage(ctx context.Context, b *bot.Bot, chatID int64, text string, replyToID *int) error {
	text = FixMarkdown(text)
	parts := SplitMessage(text, MaxMessageLen)

	for _, part := range parts {
		params := &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      part,
			ParseMode: models.ParseModeMarkdown,
		}
		if replyToID != nil {
			params.ReplyParameters = &models.ReplyParameters{
				MessageID: *replyToID,
			}
			replyToID = nil // only reply to first part
		}

		_, err := b.SendMessage(ctx, params)
		if err != nil {
			// Fallback to plain text
			slog.Warn("markdown send failed, falling back to plain text", "error", err)
			params.ParseMode = ""
			_, err = b.SendMessage(ctx, params)
			if err != nil {
				return fmt.Errorf("send message: %w", err)
			}
		}
	}

	return nil
}

// EditLongMessage edits a message with potentially long text.
func EditLongMessage(ctx context.Context, b *bot.Bot, chatID int64, messageID int, text string) error {
	text = FixMarkdown(text)
	if len([]rune(text)) > MaxMessageLen {
		text = string([]rune(text)[:MaxMessageLen-3]) + "..."
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		// Fallback to plain text
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      text,
		})
	}
	return err
}

// SendPhoto sends a photo with caption.
func SendPhoto(ctx context.Context, b *bot.Bot, chatID int64, photoPath string, caption string) error {
	_, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:  chatID,
		Photo:   &models.InputFileString{Data: "attach://" + photoPath},
		Caption: caption,
	})
	return err
}

// SendPhotoFromFile sends a photo from a local file.
func SendPhotoFromFile(ctx context.Context, b *bot.Bot, chatID int64, filePath string, caption string, replyMarkup models.ReplyMarkup) (*models.Message, error) {
	params := &bot.SendPhotoParams{
		ChatID:  chatID,
		Photo:   &models.InputFileUpload{Filename: filePath, Data: nil},
		Caption: caption,
	}
	if replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}
	return b.SendPhoto(ctx, params)
}
