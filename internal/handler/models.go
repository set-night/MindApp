package handler

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
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

type sortType string

const (
	sortPriceAsc  sortType = "price_asc"
	sortPriceDesc sortType = "price_desc"
	sortPopular   sortType = "popular"
	sortContext   sortType = "context"
	sortFreeOnly  sortType = "free"
)

func (h *Handler) handleModels(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	// Parse search query
	text := update.Message.Text
	parts := strings.SplitN(text, " ", 2)
	searchQuery := ""
	if len(parts) > 1 {
		searchQuery = strings.TrimSpace(parts[1])
	}

	h.sendModelsPage(ctx, b, chatID, user, 0, string(sortPriceAsc), searchQuery, false, 0)
}

func (h *Handler) sendModelsPage(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, page int, sortBy string, search string, edit bool, messageID int) {
	allModels, err := h.openRouter.ListModels(ctx)
	if err != nil {
		slog.Error("list models", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при загрузке моделей.",
		})
		return
	}

	// Filter by search
	filtered := allModels
	if search != "" {
		filtered = filterModels(allModels, search)
	}

	// Sort
	sortModels(filtered, sortType(sortBy))

	// Calculate markup
	markupPercent := h.cfg.MarkupPercentNormal
	if user.IsPremium() {
		markupPercent = h.cfg.MarkupPercentPremium
	}

	// Paginate
	totalPages := int(math.Ceil(float64(len(filtered)) / float64(config.ModelsPerPage)))
	if totalPages == 0 {
		totalPages = 1
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}

	start := page * config.ModelsPerPage
	end := start + config.ModelsPerPage
	if end > len(filtered) {
		end = len(filtered)
	}
	pageModels := filtered[start:end]

	// Build text
	var sb strings.Builder
	sb.WriteString("🤖 *Выберите модель:*\n\n")

	for _, m := range pageModels {
		caps := modelCapsEmoji(m.Capabilities)
		priceStr := "Бесплатно"
		if !m.IsFree() {
			avgPrice := (m.PromptPrice + m.CompletionPrice) / 2 * (1 + markupPercent/100) / 1_000_000
			priceStr = fmt.Sprintf("~$%.6f/токен", avgPrice)
		}
		selected := ""
		if m.ID == user.SelectedModel {
			selected = " ✅"
		}
		sb.WriteString(fmt.Sprintf("%s *%s*%s\n💰 %s | 📝 %dk ctx\n\n",
			caps, m.Name, selected, priceStr, m.ContextLength/1000))
	}

	// Build buttons
	var rows [][]models.InlineKeyboardButton

	// Model selection buttons
	for _, m := range pageModels {
		label := m.Name
		if len(label) > 30 {
			label = label[:30] + "..."
		}
		rows = append(rows, tg.ButtonRow(
			tg.InlineButton(label, fmt.Sprintf("m_%s_%d", m.ID, page)),
		))
	}

	// Sort buttons
	sortRow := []models.InlineKeyboardButton{
		tg.InlineButton("💰↑", fmt.Sprintf("s_%s_%d", sortPriceAsc, page)),
		tg.InlineButton("💰↓", fmt.Sprintf("s_%s_%d", sortPriceDesc, page)),
		tg.InlineButton("🔥", fmt.Sprintf("s_%s_%d", sortPopular, page)),
		tg.InlineButton("📝", fmt.Sprintf("s_%s_%d", sortContext, page)),
		tg.InlineButton("🆓", fmt.Sprintf("s_%s_%d", sortFreeOnly, page)),
	}
	rows = append(rows, sortRow)

	// Pagination
	if totalPages > 1 {
		var pageRow []models.InlineKeyboardButton
		if page > 0 {
			pageRow = append(pageRow, tg.InlineButton("⬅️", fmt.Sprintf("p_%d_%s", page-1, sortBy)))
		}
		pageRow = append(pageRow, tg.InlineButton(fmt.Sprintf("%d/%d", page+1, totalPages), "cur"))
		if page < totalPages-1 {
			pageRow = append(pageRow, tg.InlineButton("➡️", fmt.Sprintf("p_%d_%s", page+1, sortBy)))
		}
		rows = append(rows, pageRow)
	}

	keyboard := tg.InlineKeyboard(rows...)

	if edit && messageID != 0 {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        sb.String(),
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: keyboard,
		})
	} else {
		// Send with image on first load
		imgPath := "assets/Models.png"
		if _, err := os.Stat(imgPath); err == nil {
			photoData, err := os.ReadFile(imgPath)
			if err == nil {
				_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
					ChatID:      chatID,
					Photo:       &models.InputFileUpload{Filename: "Models.png", Data: bytes.NewReader(photoData)},
					Caption:     sb.String(),
					ParseMode:   models.ParseModeMarkdown,
					ReplyMarkup: keyboard,
				})
				if err != nil {
					slog.Error("send models photo", "error", err)
				}
				return
			}
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        sb.String(),
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: keyboard,
		})
	}
}

func (h *Handler) handleModelSelect(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	data := update.CallbackQuery.Data
	// Format: m_MODEL_ID_PAGE
	parts := strings.SplitN(strings.TrimPrefix(data, "m_"), "_", -1)
	if len(parts) < 2 {
		return
	}

	// Last part is page number, everything before is model ID
	pageStr := parts[len(parts)-1]
	modelID := strings.Join(parts[:len(parts)-1], "_")

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// Check if model exists
	model, err := h.openRouter.GetModel(ctx, modelID)
	if err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Модель не найдена",
			ShowAlert:       true,
		})
		return
	}

	// Check balance restriction
	if !model.IsFree() && !user.IsPremium() {
		avgPrice := (model.PromptPrice + model.CompletionPrice) / 2 / 1_000_000
		if user.Balance.InexactFloat64() < config.LowBalanceThreshold && avgPrice > config.LowPriceThreshold {
			b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "Недостаточно средств для этой модели. Пополните баланс.",
				ShowAlert:       true,
			})
			return
		}
	}

	// Set model
	h.queries.SetUserSelectedModel(ctx, sqlc.SetUserSelectedModelParams{
		ID:            user.ID,
		SelectedModel: modelID,
	})

	// Update active session model if exists
	if user.ActiveSessionID != nil {
		h.queries.UpdateSessionModel(ctx, sqlc.UpdateSessionModelParams{
			ID:    *user.ActiveSessionID,
			Model: modelID,
		})
	}

	var chatID int64
	var msgID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		msgID = msg.ID
	}

	page, _ := strconv.Atoi(pageStr)
	// Re-fetch user with updated model
	user.SelectedModel = modelID
	h.sendModelsPage(ctx, b, chatID, user, page, string(sortPriceAsc), "", true, msgID)
}

func (h *Handler) handleModelPage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	data := update.CallbackQuery.Data
	// Format: p_PAGE_SORTTYPE
	parts := strings.Split(strings.TrimPrefix(data, "p_"), "_")
	if len(parts) < 2 {
		return
	}

	page, _ := strconv.Atoi(parts[0])
	sortBy := strings.Join(parts[1:], "_")

	var chatID int64
	var msgID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		msgID = msg.ID
	}

	h.sendModelsPage(ctx, b, chatID, user, page, sortBy, "", true, msgID)
}

func (h *Handler) handleModelSort(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	data := update.CallbackQuery.Data
	// Format: s_SORTTYPE_PAGE
	parts := strings.Split(strings.TrimPrefix(data, "s_"), "_")
	if len(parts) < 2 {
		return
	}

	sortBy := strings.Join(parts[:len(parts)-1], "_")

	var chatID int64
	var msgID int
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		chatID = msg.Chat.ID
		msgID = msg.ID
	}

	h.sendModelsPage(ctx, b, chatID, user, 0, sortBy, "", true, msgID)
}

func filterModels(aiModels []domain.AIModel, query string) []domain.AIModel {
	query = strings.ToLower(query)
	var filtered []domain.AIModel
	for _, m := range aiModels {
		if strings.Contains(strings.ToLower(m.Name), query) ||
			strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Description), query) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func sortModels(aiModels []domain.AIModel, s sortType) {
	switch s {
	case sortPriceAsc:
		sort.Slice(aiModels, func(i, j int) bool {
			return aiModels[i].PromptPrice+aiModels[i].CompletionPrice < aiModels[j].PromptPrice+aiModels[j].CompletionPrice
		})
	case sortPriceDesc:
		sort.Slice(aiModels, func(i, j int) bool {
			return aiModels[i].PromptPrice+aiModels[i].CompletionPrice > aiModels[j].PromptPrice+aiModels[j].CompletionPrice
		})
	case sortPopular:
		sort.Slice(aiModels, func(i, j int) bool {
			return aiModels[i].UsageCount > aiModels[j].UsageCount
		})
	case sortContext:
		sort.Slice(aiModels, func(i, j int) bool {
			return aiModels[i].ContextLength > aiModels[j].ContextLength
		})
	case sortFreeOnly:
		var free []domain.AIModel
		for _, m := range aiModels {
			if m.IsFree() {
				free = append(free, m)
			}
		}
		copy(aiModels, free)
		// Truncate
		for i := len(free); i < len(aiModels); i++ {
			aiModels[i] = domain.AIModel{}
		}
	}
}

func modelCapsEmoji(caps domain.ModelCapabilities) string {
	var emojis []string
	if caps.Vision {
		emojis = append(emojis, "👁")
	}
	if caps.Audio {
		emojis = append(emojis, "🔊")
	}
	if caps.ImageGeneration {
		emojis = append(emojis, "🎨")
	}
	if caps.Files {
		emojis = append(emojis, "📎")
	}
	if len(emojis) == 0 {
		return "💬"
	}
	return strings.Join(emojis, "")
}
