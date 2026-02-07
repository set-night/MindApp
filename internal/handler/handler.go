package handler

import (
	"github.com/go-telegram/bot"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/set-night/mindapp/internal/service"
	"github.com/set-night/mindapp/internal/telegram"
)

// Handler holds all dependencies needed by command and callback handlers.
type Handler struct {
	bot             *bot.Bot
	cfg             *config.Config
	userService     *service.UserService
	groupService    *service.GroupService
	sessionService  *service.SessionService
	billingService  *service.BillingService
	paymentService  *service.PaymentService
	promoService    *service.PromoService
	premiumService  *service.PremiumService
	openRouter      *service.OpenRouterService
	skysmartService *service.SkysmartService
	queries         *sqlc.Queries
	tgLogger        *telegram.TelegramLogger
	botUsername      string
}

// Deps contains all dependencies required to construct a Handler.
type Deps struct {
	Bot             *bot.Bot
	Cfg             *config.Config
	UserService     *service.UserService
	GroupService    *service.GroupService
	SessionService  *service.SessionService
	BillingService  *service.BillingService
	PaymentService  *service.PaymentService
	PromoService    *service.PromoService
	PremiumService  *service.PremiumService
	OpenRouter      *service.OpenRouterService
	SkysmartService *service.SkysmartService
	Queries         *sqlc.Queries
	TgLogger        *telegram.TelegramLogger
	BotUsername      string
}

// New creates a new Handler from the provided dependencies.
func New(deps Deps) *Handler {
	return &Handler{
		bot:             deps.Bot,
		cfg:             deps.Cfg,
		userService:     deps.UserService,
		groupService:    deps.GroupService,
		sessionService:  deps.SessionService,
		billingService:  deps.BillingService,
		paymentService:  deps.PaymentService,
		promoService:    deps.PromoService,
		premiumService:  deps.PremiumService,
		openRouter:      deps.OpenRouter,
		skysmartService: deps.SkysmartService,
		queries:         deps.Queries,
		tgLogger:        deps.TgLogger,
		botUsername:      deps.BotUsername,
	}
}
