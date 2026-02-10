package handler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/set-night/mindapp/internal/service"
	tg "github.com/set-night/mindapp/internal/telegram"
)

func (h *Handler) handlePremium(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Chat.Type != "private" {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := update.Message.Chat.ID

	premiumStatus := "Нет"
	if user.IsPremium() {
		premiumStatus = fmt.Sprintf("Активен до %s", user.PremiumUntil.Format("02.01.2006"))
	}

	text := fmt.Sprintf(
		"⭐ *Премиум подписка*\n\n"+
			"Статус: *%s*\n\n"+
			"*Преимущества:*\n"+
			"• Сессий: 3 → 50\n"+
			"• Наценка: 30%% → 15%%\n"+
			"• Кулдаун: 9с → 5с\n"+
			"• Сообщений/сессию: 100 → 500\n"+
			"• Настройка температуры\n"+
			"• Таймаут сессии\n\n"+
			"💰 Баланс: *$%.4f*",
		premiumStatus,
		user.Balance.InexactFloat64(),
	)

	options := service.GetPremiumOptions()
	var rows [][]models.InlineKeyboardButton
	for i, opt := range options {
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(fmt.Sprintf("%s — $%.0f", opt.Label, opt.Price), fmt.Sprintf("premium_%d", i)),
		))
	}

	imgPath := "assets/Premium.png"
	if _, err := os.Stat(imgPath); err == nil {
		photoData, err := os.ReadFile(imgPath)
		if err == nil {
			b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:      chatID,
				Photo:       &models.InputFileUpload{Filename: "Premium.png", Data: bytes.NewReader(photoData)},
				Caption:     text,
				ParseMode:   models.ParseModeMarkdownV1,
				ReplyMarkup: tg.InlineKeyboard(rows...),
			})
			return
		}
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

func (h *Handler) handlePremiumBuy(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	idxStr := strings.TrimPrefix(data, "premium_")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return
	}

	options := service.GetPremiumOptions()
	if idx < 0 || idx >= len(options) {
		return
	}
	option := options[idx]

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	if err := h.premiumService.Purchase(ctx, user.ID, option); err != nil {
		msg := "❌ Ошибка при покупке."
		if err == domain.ErrInsufficientBalance {
			msg = fmt.Sprintf("❌ Недостаточно средств. Нужно $%.0f, у вас $%.2f", option.Price, user.Balance.InexactFloat64())
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   msg,
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ Премиум *%s* активирован!", option.Label),
		ParseMode: models.ParseModeMarkdownV1,
	})

	h.tgLogger.LogPremiumPurchase(user.TelegramID, option.Label, option.Price)
}
