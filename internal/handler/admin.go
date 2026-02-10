package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/shopspring/decimal"
)

func (h *Handler) handlePromoCreate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil || !user.IsAdmin {
		return
	}

	chatID := update.Message.Chat.ID
	parts := strings.Fields(update.Message.Text)

	// /promoCreate amount count [comment]
	if len(parts) < 3 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Использование: /promoCreate <сумма> <количество> [комментарий]",
		})
		return
	}

	amount, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || amount <= 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Некорректная сумма.",
		})
		return
	}

	count, err := strconv.Atoi(parts[2])
	if err != nil || count <= 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Некорректное количество.",
		})
		return
	}

	comment := ""
	if len(parts) > 3 {
		comment = strings.Join(parts[3:], " ")
	}

	codes, err := h.promoService.Create(ctx, decimal.NewFromFloat(amount), count, comment, user.TelegramID)
	if err != nil {
		slog.Error("create promos", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при создании промокодов.",
		})
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Создано %d промокодов на $%.2f:\n\n", count, amount))
	for _, code := range codes {
		sb.WriteString(fmt.Sprintf("`%s`\nhttps://t.me/%s?start=p_%s\n\n", code, h.botUsername, code))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      sb.String(),
		ParseMode: models.ParseModeMarkdownV1,
	})
}

func (h *Handler) handleStat(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil || !user.IsAdmin {
		return
	}

	chatID := update.Message.Chat.ID

	totalUsers, _ := h.queries.CountTotalUsers(ctx)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(now.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	todayUsers, _ := h.queries.CountUsersCreatedAfter(ctx, pgtype.Timestamptz{Time: todayStart, Valid: true})
	weekUsers, _ := h.queries.CountUsersCreatedAfter(ctx, pgtype.Timestamptz{Time: weekStart, Valid: true})
	monthUsers, _ := h.queries.CountUsersCreatedAfter(ctx, pgtype.Timestamptz{Time: monthStart, Valid: true})
	premiumUsers, _ := h.queries.CountPremiumUsers(ctx)
	totalPromos, _ := h.queries.CountPromos(ctx)
	totalActivations, _ := h.queries.CountPromoActivations(ctx)

	text := fmt.Sprintf(
		"📊 *Статистика*\n\n"+
			"👥 *Пользователи:*\n"+
			"Всего: %d\n"+
			"Сегодня: %d\n"+
			"За неделю: %d\n"+
			"За месяц: %d\n"+
			"Премиум: %d\n\n"+
			"🎟 *Промокоды:*\n"+
			"Создано: %d\n"+
			"Активаций: %d",
		totalUsers,
		todayUsers,
		weekUsers,
		monthUsers,
		premiumUsers,
		totalPromos,
		totalActivations,
	)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdownV1,
	})
}
