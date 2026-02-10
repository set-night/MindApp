package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	tg "github.com/set-night/mindapp/internal/telegram"
)

func (h *Handler) handlePayTasks(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.showTasks(ctx, b, update, false)
}

func (h *Handler) handleFreeTasks(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.showTasks(ctx, b, update, true)
}

func (h *Handler) showTasks(ctx context.Context, b *bot.Bot, update *models.Update, freeOnly bool) {
	if update.Message == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	chatID := update.Message.Chat.ID

	tasks, err := h.queries.GetAvailablePayTasksForUser(ctx, user.ID)
	if err != nil {
		slog.Error("get tasks", "error", err)
		return
	}

	if len(tasks) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "📋 Нет доступных заданий.",
		})
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 *Доступные задания:*\n\n")

	var rows [][]models.InlineKeyboardButton
	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("• *%s* — $%.2f\n", t.Title, t.Reward.InexactFloat64()))
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(t.Title, fmt.Sprintf("select_task_%d", t.ID)),
		))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        sb.String(),
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: tg.InlineKeyboard(rows...),
	})
}

func (h *Handler) handleSelectTask(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	data := update.CallbackQuery.Data
	idStr := strings.TrimPrefix(data, "select_task_")
	taskID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}

	task, err := h.queries.GetPayTaskByID(ctx, taskID)
	if err != nil {
		return
	}

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("📋 *%s*\n\nНаграда: *$%.2f*\n\nПодпишитесь на канал и нажмите проверку:", task.Title, task.Reward.InexactFloat64()),
		ParseMode: models.ParseModeMarkdownV1,
		ReplyMarkup: tg.InlineKeyboard(
			tg.ButtonRow(tg.URLButton("📢 Перейти", task.TelegramLink)),
			tg.ButtonRow(tg.InlineButton("✅ Проверить", fmt.Sprintf("check_task_%d", task.ID))),
		),
	})
}

func (h *Handler) handleCheckTask(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	idStr := strings.TrimPrefix(data, "check_task_")
	taskID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}

	var chatID int64
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
	}

	task, err := h.queries.GetPayTaskByID(ctx, taskID)
	if err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Задание не найдено",
			ShowAlert:       true,
		})
		return
	}

	// Check if already completed
	completed, _ := h.queries.CheckPayTaskCompletion(ctx, sqlc.CheckPayTaskCompletionParams{
		TaskID: taskID,
		UserID: user.ID,
	})
	if completed {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Вы уже выполнили это задание",
			ShowAlert:       true,
		})
		return
	}

	// Check channel membership
	member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
		ChatID: task.ChannelID,
		UserID: user.TelegramID,
	})
	if err != nil || member == nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Вы не подписаны на канал!",
			ShowAlert:       true,
		})
		return
	}

	status := member.Member.Status
	if status == "left" || status == "kicked" {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Вы не подписаны на канал!",
			ShowAlert:       true,
		})
		return
	}

	// Mark completed and credit
	h.queries.CreatePayTaskCompletion(ctx, sqlc.CreatePayTaskCompletionParams{
		TaskID: taskID,
		UserID: user.ID,
	})

	h.billingService.CreditUser(ctx, user.ID, task.Reward, fmt.Sprintf("Task reward: %s", task.Title))

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ Задание выполнено! Начислено *$%.2f*", task.Reward.InexactFloat64()),
		ParseMode: models.ParseModeMarkdownV1,
	})

	h.tgLogger.Log(tg.LogTypeFreeBalance, fmt.Sprintf("💰 *Task Reward*\n\nUser: `%d`\nTask: %s\nReward: $%.2f", user.TelegramID, task.Title, task.Reward.InexactFloat64()))
}
