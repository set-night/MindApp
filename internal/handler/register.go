package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Register registers all command and callback handlers on the bot instance.
func (h *Handler) Register() {
	// Commands
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, h.handleStart)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/settings", bot.MatchTypePrefix, h.handleSettings)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/models", bot.MatchTypePrefix, h.handleModels)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/sessions", bot.MatchTypePrefix, h.handleSessions)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/favorite", bot.MatchTypePrefix, h.handleFavorite)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/end", bot.MatchTypePrefix, h.handleEnd)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/pay", bot.MatchTypePrefix, h.handlePay)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/premium", bot.MatchTypePrefix, h.handlePremium)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/promo", bot.MatchTypePrefix, h.handlePromo)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/referral", bot.MatchTypePrefix, h.handleReferral)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/prompt", bot.MatchTypePrefix, h.handlePrompt)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/task", bot.MatchTypePrefix, h.handlePayTasks)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/free", bot.MatchTypePrefix, h.handleFreeTasks)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/stat", bot.MatchTypePrefix, h.handleStat)
	h.bot.RegisterHandler(bot.HandlerTypeMessageText, "/promoCreate", bot.MatchTypePrefix, h.handlePromoCreate)

	// Settings callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "toggle_context", bot.MatchTypePrefix, h.handleToggleContext)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "toggle_cost", bot.MatchTypePrefix, h.handleToggleCost)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "toggle_send_user_info", bot.MatchTypePrefix, h.handleToggleSendUserInfo)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "set_temperature", bot.MatchTypePrefix, h.handleSetTemperature)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "temp_", bot.MatchTypePrefix, h.handleTempValue)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "set_timeout_", bot.MatchTypePrefix, h.handleSetTimeout)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "toggle_thread_id", bot.MatchTypePrefix, h.handleToggleThreadID)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "back_to_settings", bot.MatchTypePrefix, h.handleBackToSettings)

	// Models callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "m_", bot.MatchTypePrefix, h.handleModelSelect)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "p_", bot.MatchTypePrefix, h.handleModelPage)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "s_", bot.MatchTypePrefix, h.handleModelSort)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "cur", bot.MatchTypeExact, h.handleNoop)

	// Sessions callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "new_session", bot.MatchTypePrefix, h.handleNewSession)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "delete_current", bot.MatchTypePrefix, h.handleDeleteCurrentSession)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "delete_all", bot.MatchTypePrefix, h.handleDeleteAllSessions)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "switch_session_", bot.MatchTypePrefix, h.handleSwitchSession)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "sessions_page_", bot.MatchTypePrefix, h.handleSessionsPage)

	// Favorite callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "fav_select_", bot.MatchTypePrefix, h.handleFavSelect)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "fav_remove_", bot.MatchTypePrefix, h.handleFavRemove)

	// Payment callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "buy_invoice_main", bot.MatchTypePrefix, h.handleBuyInvoiceMain)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "buy_", bot.MatchTypePrefix, h.handleBuyAmount)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "buy_cryptomus_main", bot.MatchTypePrefix, h.handleBuyCryptomusMain)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "cryptomus_", bot.MatchTypePrefix, h.handleCryptomusAmount)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "check_cryptomus_", bot.MatchTypePrefix, h.handleCheckCryptomus)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "delete_cryptomus_", bot.MatchTypePrefix, h.handleDeleteCryptomus)

	// Premium callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "premium_", bot.MatchTypePrefix, h.handlePremiumBuy)

	// Prompt callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "choose_prompt_", bot.MatchTypePrefix, h.handleChoosePrompt)

	// Pay task callbacks
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "select_task_", bot.MatchTypePrefix, h.handleSelectTask)
	h.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "check_task_", bot.MatchTypePrefix, h.handleCheckTask)

	// Note: PreCheckoutQuery is handled via default handler in main.go
}

// handleNoop is a no-op callback handler used for pagination indicators and other
// non-interactive inline buttons. It simply acknowledges the callback query.
func (h *Handler) handleNoop(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})
	}
}

