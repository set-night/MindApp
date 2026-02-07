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
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	tg "github.com/set-night/mindapp/internal/telegram"
	"github.com/shopspring/decimal"
)

func (h *Handler) handleSettings(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	chatType := update.Message.Chat.Type

	if chatType == "private" {
		h.sendUserSettings(ctx, b, chatID)
	} else {
		h.sendGroupSettings(ctx, b, chatID, update)
	}
}

func (h *Handler) sendUserSettings(ctx context.Context, b *bot.Bot, chatID int64) {
	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	contextStatus := "❌ Выкл"
	if user.ContextEnabled {
		contextStatus = "✅ Вкл"
	}
	costStatus := "❌ Выкл"
	if user.ShowCost {
		costStatus = "✅ Вкл"
	}
	userInfoStatus := "❌ Выкл"
	if user.SendUserInfo {
		userInfoStatus = "✅ Вкл"
	}

	premiumStatus := "Нет"
	if user.IsPremium() {
		premiumStatus = fmt.Sprintf("До %s", user.PremiumUntil.Format("02.01.2006"))
	}

	text := fmt.Sprintf(
		"⚙️ *Настройки*\n\n"+
			"💰 Баланс: *$%.4f*\n"+
			"🤖 Модель: `%s`\n"+
			"🌡 Температура: *%.1f*\n"+
			"⭐ Премиум: *%s*\n",
		user.Balance.InexactFloat64(),
		user.SelectedModel,
		user.Temperature,
		premiumStatus,
	)

	var rows [][]models.InlineKeyboardButton

	rows = append(rows, tg.ButtonRow(
		tg.InlineButton(fmt.Sprintf("🔄 Контекст: %s", contextStatus), "toggle_context"),
	))
	rows = append(rows, tg.ButtonRow(
		tg.InlineButton(fmt.Sprintf("💰 Показ стоимости: %s", costStatus), "toggle_cost"),
	))
	rows = append(rows, tg.ButtonRow(
		tg.InlineButton(fmt.Sprintf("👤 Отправка данных: %s", userInfoStatus), "toggle_send_user_info"),
	))

	if user.IsPremium() {
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton("🌡 Температура", "set_temperature"),
		))

		// Session timeout
		timeoutStr := "Выкл"
		switch user.SessionTimeoutMs {
		case 600000:
			timeoutStr = "10 мин"
		case 1800000:
			timeoutStr = "30 мин"
		case 3600000:
			timeoutStr = "1 час"
		case 86400000:
			timeoutStr = "1 день"
		}
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(fmt.Sprintf("⏱ Таймаут: %s", timeoutStr), "set_timeout_menu"),
		))
	}

	// Send with image
	imgPath := "assets/Settings.png"
	if _, err := os.Stat(imgPath); err == nil {
		photoData, err := os.ReadFile(imgPath)
		if err == nil {
			_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:      chatID,
				Photo:       &models.InputFileUpload{Filename: "Settings.png", Data: bytes.NewReader(photoData)},
				Caption:     text,
				ParseMode:   models.ParseModeMarkdown,
				ReplyMarkup: tg.InlineKeyboard(rows...),
			})
			if err != nil {
				slog.Error("send settings photo", "error", err)
			}
			return
		}
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

func (h *Handler) sendGroupSettings(ctx context.Context, b *bot.Bot, chatID int64, update *models.Update) {
	group := middleware.GetGroup(ctx)
	if group == nil {
		return
	}

	contextStatus := "❌ Выкл"
	if group.ContextEnabled {
		contextStatus = "✅ Вкл"
	}
	costStatus := "❌ Выкл"
	if group.ShowCost {
		costStatus = "✅ Вкл"
	}

	threadStr := "Не задан"
	if group.ThreadID != nil {
		threadStr = fmt.Sprintf("%d", *group.ThreadID)
	}

	text := fmt.Sprintf(
		"⚙️ *Настройки группы*\n\n"+
			"💰 Баланс: *$%.4f*\n"+
			"🤖 Модель: `%s`\n"+
			"📌 Топик: %s\n",
		group.Balance.InexactFloat64(),
		group.SelectedModel,
		threadStr,
	)

	var rows [][]models.InlineKeyboardButton
	rows = append(rows, tg.ButtonRow(
		tg.InlineButton(fmt.Sprintf("🔄 Контекст: %s", contextStatus), "toggle_context"),
	))
	rows = append(rows, tg.ButtonRow(
		tg.InlineButton(fmt.Sprintf("💰 Показ стоимости: %s", costStatus), "toggle_cost"),
	))
	rows = append(rows, tg.ButtonRow(
		tg.InlineButton("📌 Привязать к этому топику", "toggle_thread_id"),
	))

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

// Callback handlers for settings

func (h *Handler) handleToggleContext(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	group := middleware.GetGroup(ctx)

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	if group != nil {
		h.queries.ToggleGroupContextEnabled(ctx, group.ID)
	} else if user != nil {
		h.queries.ToggleUserContextEnabled(ctx, user.ID)
	}

	h.sendUserSettings(ctx, b, chatID)
}

func (h *Handler) handleToggleCost(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	group := middleware.GetGroup(ctx)

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	if group != nil {
		h.queries.ToggleGroupShowCost(ctx, group.ID)
	} else if user != nil {
		h.queries.ToggleUserShowCost(ctx, user.ID)
	}

	h.sendUserSettings(ctx, b, chatID)
}

func (h *Handler) handleToggleSendUserInfo(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	h.queries.ToggleUserSendUserInfo(ctx, user.ID)
	h.sendUserSettings(ctx, b, chatID)
}

func (h *Handler) handleSetTemperature(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	var chatID int64
	var messageID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		messageID = msg.ID
	}

	var buttons [][]models.InlineKeyboardButton
	for _, temp := range config.TemperatureOptions {
		buttons = append(buttons, tg.ButtonRow(
			tg.InlineButton(fmt.Sprintf("%.1f", temp), fmt.Sprintf("temp_%.1f", temp)),
		))
	}
	buttons = append(buttons, tg.ButtonRow(tg.InlineButton("⬅️ Назад", "back_to_settings")))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        "🌡 *Выберите температуру:*\n\nНизкая — более точные ответы\nВысокая — более творческие ответы",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: tg.InlineKeyboard(buttons...),
	})
}

func (h *Handler) handleTempValue(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	tempStr := strings.TrimPrefix(data, "temp_")
	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return
	}

	h.queries.SetUserTemperature(ctx, sqlc.SetUserTemperatureParams{
		ID:          user.ID,
		Temperature: decimal.NewFromFloat(temp),
	})

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	h.sendUserSettings(ctx, b, chatID)
}

func (h *Handler) handleSetTimeout(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data

	// Handle menu
	if data == "set_timeout_menu" {
		var buttons [][]models.InlineKeyboardButton
		labels := []string{"Выкл", "10 мин", "30 мин", "1 час", "1 день"}
		for i, timeout := range config.SessionTimeoutOptions {
			buttons = append(buttons, tg.ButtonRow(
				tg.InlineButton(labels[i], fmt.Sprintf("set_timeout_%d", timeout)),
			))
		}
		buttons = append(buttons, tg.ButtonRow(tg.InlineButton("⬅️ Назад", "back_to_settings")))

		var chatID int64
		var messageID int
		if msg := update.CallbackQuery.Message.Message; msg != nil {
			chatID = msg.Chat.ID
			messageID = msg.ID
		}

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        "⏱ *Выберите таймаут сессии:*\n\nПо истечении таймаута контекст будет автоматически сброшен.",
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: tg.InlineKeyboard(buttons...),
		})
		return
	}

	// Handle value selection
	timeoutStr := strings.TrimPrefix(data, "set_timeout_")
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return
	}

	h.queries.SetUserSessionTimeout(ctx, sqlc.SetUserSessionTimeoutParams{
		ID:               user.ID,
		SessionTimeoutMs: int32(timeout),
	})

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}
	h.sendUserSettings(ctx, b, chatID)
}

func (h *Handler) handleToggleThreadID(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	group := middleware.GetGroup(ctx)
	if group == nil {
		return
	}

	var threadID *int32
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		if msg.MessageThreadID != 0 {
			tid := int32(msg.MessageThreadID)
			threadID = &tid
		}
	}

	h.queries.SetGroupThreadID(ctx, sqlc.SetGroupThreadIDParams{
		ID:       group.ID,
		ThreadID: threadID,
	})

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "✅ Топик обновлён.",
	})
}

func (h *Handler) handleBackToSettings(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	h.sendUserSettings(ctx, b, chatID)
}
