package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/set-night/mindapp/internal/config"
)

type TelegramLogger struct {
	bot *bot.Bot
	cfg *config.Config
}

func NewTelegramLogger(b *bot.Bot, cfg *config.Config) *TelegramLogger {
	return &TelegramLogger{bot: b, cfg: cfg}
}

type LogType string

const (
	LogTypeError           LogType = "error"
	LogTypeRegistration    LogType = "registration"
	LogTypeBalanceTopUp    LogType = "balanceTopUp"
	LogTypePromoActivate   LogType = "promoActivate"
	LogTypePremiumPurchase LogType = "premiumPurchase"
	LogTypeFreeBalance     LogType = "freeBalance"
)

func (l *TelegramLogger) Log(logType LogType, message string) {
	if l.cfg.LogTelegramChatID == 0 {
		return
	}

	topicID := l.getTopicID(logType)
	if topicID == 0 {
		return
	}

	// Truncate if too long
	if len([]rune(message)) > MaxMessageLen {
		message = string([]rune(message)[:MaxMessageLen-20]) + "\n\n... (truncated)"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := l.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          l.cfg.LogTelegramChatID,
		Text:            message,
		ParseMode:       "Markdown",
		MessageThreadID: topicID,
	})
	if err != nil {
		slog.Error("failed to send telegram log", "type", logType, "error", err)
	}
}

func (l *TelegramLogger) LogError(err error, context string) {
	msg := fmt.Sprintf("❌ *Error*\n\n*Context:* %s\n*Error:* `%s`\n*Time:* %s",
		context, err.Error(), time.Now().Format("2006-01-02 15:04:05"))
	l.Log(LogTypeError, msg)
}

func (l *TelegramLogger) LogRegistration(telegramID int64, name, username string, referredBy string) {
	msg := fmt.Sprintf("👤 *New Registration*\n\n*ID:* `%d`\n*Name:* %s\n*Username:* @%s",
		telegramID, name, username)
	if referredBy != "" {
		msg += fmt.Sprintf("\n*Referred by:* %s", referredBy)
	}
	l.Log(LogTypeRegistration, msg)
}

func (l *TelegramLogger) LogBalanceTopUp(telegramID int64, amount float64, method string) {
	msg := fmt.Sprintf("💰 *Balance Top-Up*\n\n*User:* `%d`\n*Amount:* $%.2f\n*Method:* %s",
		telegramID, amount, method)
	l.Log(LogTypeBalanceTopUp, msg)
}

func (l *TelegramLogger) LogPromoActivate(telegramID int64, code string, amount float64) {
	msg := fmt.Sprintf("🎟 *Promo Activated*\n\n*User:* `%d`\n*Code:* `%s`\n*Amount:* $%.2f",
		telegramID, code, amount)
	l.Log(LogTypePromoActivate, msg)
}

func (l *TelegramLogger) LogPremiumPurchase(telegramID int64, plan string, price float64) {
	msg := fmt.Sprintf("⭐ *Premium Purchase*\n\n*User:* `%d`\n*Plan:* %s\n*Price:* $%.2f",
		telegramID, plan, price)
	l.Log(LogTypePremiumPurchase, msg)
}

func (l *TelegramLogger) getTopicID(logType LogType) int {
	switch logType {
	case LogTypeError:
		return l.cfg.LogTopicError
	case LogTypeRegistration:
		return l.cfg.LogTopicRegistration
	case LogTypeBalanceTopUp:
		return l.cfg.LogTopicBalanceTopUp
	case LogTypePromoActivate:
		return l.cfg.LogTopicPromoActivate
	case LogTypePremiumPurchase:
		return l.cfg.LogTopicPremiumPurchase
	case LogTypeFreeBalance:
		return l.cfg.LogTopicFreeBalance
	default:
		return 0
	}
}
