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
	"github.com/set-night/mindapp/internal/service"
	tg "github.com/set-night/mindapp/internal/telegram"
	"github.com/shopspring/decimal"
)

func (h *Handler) handlePay(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	chatType := update.Message.Chat.Type

	if chatType == "private" {
		h.handlePayPrivate(ctx, b, chatID)
	} else {
		h.handlePayGroup(ctx, b, update, chatID)
	}
}

func (h *Handler) handlePayPrivate(ctx context.Context, b *bot.Bot, chatID int64) {
	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	text := fmt.Sprintf(
		"💳 *Пополнение баланса*\n\n"+
			"💰 Текущий баланс: *$%.4f*\n\n"+
			"Выберите способ оплаты:",
		user.Balance.InexactFloat64(),
	)

	var rows [][]models.InlineKeyboardButton
	if h.cfg.StarsEnabled {
		rows = append(rows, tg.ButtonRow(tg.InlineButton("⭐ Telegram Stars", "buy_invoice_main")))
	}
	if h.cfg.CryptomusEnabled {
		rows = append(rows, tg.ButtonRow(tg.InlineButton("🪙 Криптовалюта", "buy_cryptomus_main")))
	}
	if h.cfg.FunPayEnabled {
		rows = append(rows, tg.ButtonRow(tg.URLButton("💳 FunPay (Карты)", h.cfg.FunPayURL)))
	}

	// Send with image
	imgPath := "assets/Payment.png"
	if _, err := os.Stat(imgPath); err == nil {
		photoData, err := os.ReadFile(imgPath)
		if err == nil {
			b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:      chatID,
				Photo:       &models.InputFileUpload{Filename: "Payment.png", Data: bytes.NewReader(photoData)},
				Caption:     text,
				ParseMode:   models.ParseModeMarkdown,
				ReplyMarkup: tg.InlineKeyboard(rows...),
			})
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

func (h *Handler) handlePayGroup(ctx context.Context, b *bot.Bot, update *models.Update, chatID int64) {
	user := middleware.GetUser(ctx)
	group := middleware.GetGroup(ctx)
	if user == nil || group == nil {
		return
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) < 2 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Используйте: /pay <сумма>",
		})
		return
	}

	amount, err := decimal.NewFromString(parts[1])
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Некорректная сумма.",
		})
		return
	}

	if err := h.billingService.TransferUserToGroup(ctx, user.ID, group.ID, amount); err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("❌ %s", err.Error()),
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ Переведено *$%.2f* на баланс группы.", amount.InexactFloat64()),
		ParseMode: models.ParseModeMarkdown,
	})
}

// Stars payment handlers

func (h *Handler) handleBuyInvoiceMain(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	var rows [][]models.InlineKeyboardButton
	for _, amt := range config.StarPaymentAmounts {
		stars := service.CalculateStarAmount(amt)
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(fmt.Sprintf("$%d (%d ⭐)", amt, stars), fmt.Sprintf("buy_%ddollar", amt)),
		))
	}
	rows = append(rows, tg.ButtonRow(tg.InlineButton("⬅️ Назад", "back_to_pay")))

	b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Caption:     "⭐ *Оплата через Telegram Stars*\n\nВыберите сумму:",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

func (h *Handler) handleBuyAmount(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	data := update.CallbackQuery.Data
	// Format: buy_Xdollar
	amtStr := strings.TrimPrefix(data, "buy_")
	amtStr = strings.TrimSuffix(amtStr, "dollar")
	amount, err := strconv.Atoi(amtStr)
	if err != nil {
		return
	}

	stars := service.CalculateStarAmount(amount)
	chatID := update.CallbackQuery.From.ID

	b.SendInvoice(ctx, &bot.SendInvoiceParams{
		ChatID:      chatID,
		Title:       "Пополнение баланса",
		Description: fmt.Sprintf("Пополнение на $%d", amount),
		Payload:     fmt.Sprintf("topup_%d", amount),
		Currency:    "XTR",
		Prices: []models.LabeledPrice{
			{Label: fmt.Sprintf("$%d", amount), Amount: stars},
		},
	})
}

func (h *Handler) HandlePreCheckout(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.PreCheckoutQuery == nil {
		return
	}
	b.AnswerPreCheckoutQuery(ctx, &bot.AnswerPreCheckoutQueryParams{
		PreCheckoutQueryID: update.PreCheckoutQuery.ID,
		OK:                 true,
	})
}

// HandleSuccessfulPayment should be called from the default message handler.
func (h *Handler) HandleSuccessfulPayment(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.SuccessfulPayment == nil {
		return
	}

	payment := update.Message.SuccessfulPayment
	chatID := update.Message.Chat.ID
	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	// Parse amount from payload
	payload := payment.InvoicePayload
	amtStr := strings.TrimPrefix(payload, "topup_")
	amount, err := strconv.Atoi(amtStr)
	if err != nil {
		slog.Error("parse payment payload", "error", err, "payload", payload)
		return
	}

	amountDec := decimal.NewFromInt(int64(amount))
	if err := h.billingService.CreditUser(ctx, user.ID, amountDec, fmt.Sprintf("Stars payment: $%d", amount)); err != nil {
		slog.Error("credit user after stars payment", "error", err)
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ Баланс пополнен на *$%d*!", amount),
		ParseMode: models.ParseModeMarkdown,
	})

	h.tgLogger.LogBalanceTopUp(user.TelegramID, float64(amount), "Telegram Stars")

	// Referral bonus
	if user.ReferredByID != nil {
		h.queries.UpdateUserReferralBalance(ctx, sqlc.UpdateUserReferralBalanceParams{
			ID:              *user.ReferredByID,
			ReferralBalance: amountDec,
		})
	}
}

// Cryptomus handlers

func (h *Handler) handleBuyCryptomusMain(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	var rows [][]models.InlineKeyboardButton
	for _, amt := range config.CryptomusPaymentAmounts {
		label := fmt.Sprintf("$%.0f", amt)
		if amt < 1 {
			label = fmt.Sprintf("$%.1f", amt)
		}
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(label, fmt.Sprintf("cryptomus_%.2f", amt)),
		))
	}
	rows = append(rows, tg.ButtonRow(tg.InlineButton("⬅️ Назад", "back_to_pay")))

	b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Caption:     "🪙 *Оплата криптовалютой*\n\nВыберите сумму:",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

func (h *Handler) handleCryptomusAmount(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	amtStr := strings.TrimPrefix(data, "cryptomus_")
	amount, err := strconv.ParseFloat(amtStr, 64)
	if err != nil {
		return
	}

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	// Check for pending invoice
	pending, _ := h.paymentService.GetPendingInvoice(ctx, user.TelegramID)
	if pending != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "У вас уже есть неоплаченный счёт. Оплатите его или удалите.",
			ReplyMarkup: tg.InlineKeyboard(
				tg.ButtonRow(
					tg.InlineButton("🔄 Проверить", fmt.Sprintf("check_cryptomus_%s", pending.CryptomusInvoiceID)),
					tg.InlineButton("🗑 Удалить", fmt.Sprintf("delete_cryptomus_%d", pending.ID)),
				),
			),
		})
		return
	}

	invoice, err := h.paymentService.CreateCryptomusInvoice(ctx, user.TelegramID, amount)
	if err != nil {
		slog.Error("create cryptomus invoice", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при создании счёта.",
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("🪙 Счёт на *$%.2f* создан.\n\nОплатите по ссылке и нажмите кнопку проверки.", amount),
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: tg.InlineKeyboard(
			tg.ButtonRow(tg.URLButton("💳 Оплатить", invoice.PaymentURL)),
			tg.ButtonRow(tg.InlineButton("🔄 Проверить оплату", fmt.Sprintf("check_cryptomus_%s", invoice.InvoiceID))),
		),
	})
}

func (h *Handler) handleCheckCryptomus(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	invoiceID := strings.TrimPrefix(data, "check_cryptomus_")

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	inv, err := h.paymentService.CheckCryptomusPayment(ctx, invoiceID)
	if err != nil {
		slog.Error("check cryptomus", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при проверке оплаты.",
		})
		return
	}

	switch inv.Status {
	case "paid":
		if err := h.billingService.CreditUser(ctx, user.ID, inv.Amount, "Cryptomus payment"); err != nil {
			slog.Error("credit after cryptomus", "error", err)
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      fmt.Sprintf("✅ Оплата подтверждена! Начислено *$%.2f*", inv.Amount.InexactFloat64()),
			ParseMode: models.ParseModeMarkdown,
		})
		h.tgLogger.LogBalanceTopUp(user.TelegramID, inv.Amount.InexactFloat64(), "Cryptomus")

		// Referral bonus
		if user.ReferredByID != nil {
			h.queries.UpdateUserReferralBalance(ctx, sqlc.UpdateUserReferralBalanceParams{
				ID:              *user.ReferredByID,
				ReferralBalance: inv.Amount,
			})
		}
	case "pending":
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "⏳ Оплата ещё не получена. Попробуйте позже.",
		})
	default:
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Оплата не удалась.",
		})
	}
}

func (h *Handler) handleDeleteCryptomus(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	data := update.CallbackQuery.Data
	idStr := strings.TrimPrefix(data, "delete_cryptomus_")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}

	h.paymentService.DeleteInvoice(ctx, id)

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "🗑 Счёт удалён.",
	})
}
