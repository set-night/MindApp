package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	tg "github.com/set-night/mindapp/internal/telegram"
)

func (h *Handler) handleFavorite(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Chat.Type != "private" {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := update.Message.Chat.ID

	if len(user.FavoriteModels) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "⭐ У вас нет избранных моделей.\n\nИспользуйте /models для выбора моделей.",
		})
		return
	}

	var rows [][]models.InlineKeyboardButton
	for _, modelID := range user.FavoriteModels {
		model, err := h.openRouter.GetModel(ctx, modelID)
		name := modelID
		if err == nil {
			name = model.Name
		}

		selected := ""
		if modelID == user.SelectedModel {
			selected = " ✅"
		}

		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(fmt.Sprintf("🤖 %s%s", name, selected), fmt.Sprintf("fav_select_%s", modelID)),
			tg.InlineButton("❌", fmt.Sprintf("fav_remove_%s", modelID)),
		))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "⭐ *Избранные модели:*",
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

func (h *Handler) handleFavSelect(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	modelID := strings.TrimPrefix(update.CallbackQuery.Data, "fav_select_")

	h.queries.SetUserSelectedModel(ctx, sqlc.SetUserSelectedModelParams{
		ID:            user.ID,
		SelectedModel: modelID,
	})

	if user.ActiveSessionID != nil {
		h.queries.UpdateSessionModel(ctx, sqlc.UpdateSessionModelParams{
			ID:    *user.ActiveSessionID,
			Model: modelID,
		})
	}

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ Модель изменена на `%s`", modelID),
		ParseMode: models.ParseModeMarkdownV1,
	})
}

func (h *Handler) handleFavRemove(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	modelID := strings.TrimPrefix(update.CallbackQuery.Data, "fav_remove_")

	// Remove from favorites
	newFavorites := make([]string, 0)
	for _, fav := range user.FavoriteModels {
		if fav != modelID {
			newFavorites = append(newFavorites, fav)
		}
	}

	h.queries.SetUserFavoriteModels(ctx, sqlc.SetUserFavoriteModelsParams{
		ID:             user.ID,
		FavoriteModels: newFavorites,
	})

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "Удалено из избранного",
	})
}
