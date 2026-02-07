package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/middleware"
	tg "github.com/set-night/mindapp/internal/telegram"
)

func (h *Handler) handlePrompt(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID

	prompts, err := h.queries.GetOfficialPrompts(ctx)
	if err != nil {
		slog.Error("get prompts", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при загрузке промптов.",
		})
		return
	}

	if len(prompts) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Промпты пока не добавлены.",
		})
		return
	}

	var rows [][]models.InlineKeyboardButton
	for _, p := range prompts {
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(p.Title, fmt.Sprintf("choose_prompt_%d", p.ID)),
		))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "📝 *Выберите системный промпт:*",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

func (h *Handler) handleChoosePrompt(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	idStr := strings.TrimPrefix(data, "choose_prompt_")

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	h.activatePrompt(ctx, b, chatID, user, idStr)
}
