package handler

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	tg "github.com/set-night/mindapp/internal/telegram"
)

func (h *Handler) handleSessions(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Chat.Type != "private" {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := update.Message.Chat.ID
	h.sendSessionsPage(ctx, b, chatID, user, 0, false, 0)
}

func (h *Handler) sendSessionsPage(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, page int, edit bool, messageID int) {
	total, err := h.sessionService.CountByUser(ctx, user.ID)
	if err != nil {
		slog.Error("count sessions", "error", err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(config.SessionsPerPage)))
	if totalPages == 0 {
		totalPages = 1
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	sessions, err := h.sessionService.ListByUser(ctx, user.ID, config.SessionsPerPage, page*config.SessionsPerPage)
	if err != nil {
		slog.Error("list sessions", "error", err)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📂 *Сессии* (%d шт.)\n\n", total))

	var rows [][]models.InlineKeyboardButton

	for _, s := range sessions {
		firstMsg, _ := h.sessionService.GetFirstMessage(ctx, s.ID)
		label := fmt.Sprintf("📝 %s", s.CreatedAt.Format("02.01 15:04"))
		if firstMsg != nil && firstMsg.Text != "" {
			snippet := firstMsg.Text
			if len(snippet) > 30 {
				snippet = snippet[:30] + "..."
			}
			label = snippet
		}
		active := ""
		if user.ActiveSessionID != nil && *user.ActiveSessionID == s.ID {
			active = " ✅"
		}
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(fmt.Sprintf("%s%s", label, active), fmt.Sprintf("switch_session_%d", s.ID)),
		))
	}

	// Action buttons
	actionRow := []models.InlineKeyboardButton{
		tg.InlineButton("➕ Новая", "new_session"),
		tg.InlineButton("🗑 Текущую", "delete_current"),
		tg.InlineButton("🗑 Все", "delete_all"),
	}
	rows = append(rows, actionRow)

	// Pagination
	if totalPages > 1 {
		var pageRow []models.InlineKeyboardButton
		if page > 0 {
			pageRow = append(pageRow, tg.InlineButton("⬅️", fmt.Sprintf("sessions_page_%d", page-1)))
		}
		pageRow = append(pageRow, tg.InlineButton(fmt.Sprintf("%d/%d", page+1, totalPages), "cur"))
		if page < totalPages-1 {
			pageRow = append(pageRow, tg.InlineButton("➡️", fmt.Sprintf("sessions_page_%d", page+1)))
		}
		rows = append(rows, pageRow)
	}

	keyboard := tg.InlineKeyboard(rows...)
	text := sb.String()

	if edit && messageID != 0 {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: keyboard,
		})
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: keyboard,
		})
	}
}

func (h *Handler) handleNewSession(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	_, err := h.sessionService.CreateNew(ctx, user)
	if err != nil {
		slog.Error("create session", "error", err)
		return
	}

	var chatID int64
	var messageID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		messageID = msg.ID
	}

	// Re-fetch user to get updated active_session_id
	user, _ = h.userService.GetByTelegramID(ctx, user.TelegramID)
	h.sendSessionsPage(ctx, b, chatID, user, 0, true, messageID)
}

func (h *Handler) handleDeleteCurrentSession(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil || user.ActiveSessionID == nil {
		return
	}

	h.sessionService.Delete(ctx, *user.ActiveSessionID)
	h.queries.SetUserActiveSession(ctx, sqlc.SetUserActiveSessionParams{ID: user.ID, ActiveSessionID: nil})

	var chatID int64
	var messageID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		messageID = msg.ID
	}

	user.ActiveSessionID = nil
	h.sendSessionsPage(ctx, b, chatID, user, 0, true, messageID)
}

func (h *Handler) handleDeleteAllSessions(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	h.sessionService.DeleteAll(ctx, user.ID)

	var chatID int64
	var messageID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		messageID = msg.ID
	}

	user.ActiveSessionID = nil
	h.sendSessionsPage(ctx, b, chatID, user, 0, true, messageID)
}

func (h *Handler) handleSwitchSession(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	idStr := strings.TrimPrefix(data, "switch_session_")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}

	h.sessionService.SwitchTo(ctx, user.ID, sessionID)
	user.ActiveSessionID = &sessionID

	var chatID int64
	var messageID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		messageID = msg.ID
	}

	h.sendSessionsPage(ctx, b, chatID, user, 0, true, messageID)
}

func (h *Handler) handleSessionsPage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	pageStr := strings.TrimPrefix(data, "sessions_page_")
	page, _ := strconv.Atoi(pageStr)

	var chatID int64
	var messageID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		messageID = msg.ID
	}

	h.sendSessionsPage(ctx, b, chatID, user, page, true, messageID)
}
