package main

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	mindapproot "github.com/set-night/mindapp"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/handler"
	"github.com/set-night/mindapp/internal/middleware"
	"github.com/set-night/mindapp/internal/repository"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/set-night/mindapp/internal/service"
	"github.com/set-night/mindapp/internal/telegram"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup context with graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Connect to database
	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Run migrations
	migrationsFS, err := fs.Sub(mindapproot.MigrationsFS, "migrations")
	if err != nil {
		slog.Error("failed to load embedded migrations", "error", err)
		os.Exit(1)
	}
	if err := repository.RunMigrations(cfg.DatabaseURL, migrationsFS); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize sqlc queries
	queries := sqlc.New(pool)

	// Initialize services
	userService := service.NewUserService(pool, queries)
	groupService := service.NewGroupService(pool, queries)
	sessionService := service.NewSessionService(pool, queries)
	billingService := service.NewBillingService(pool, queries)
	paymentService := service.NewPaymentService(pool, queries, cfg)
	promoService := service.NewPromoService(pool, queries)
	premiumService := service.NewPremiumService(pool, queries)
	openRouter := service.NewOpenRouterService(cfg.OpenRouterKey)
	skysmartService := service.NewSkysmartService()

	// Handler pointer for use in default handler closure
	var h *handler.Handler

	// Create bot
	opts := []bot.Option{
		bot.WithMiddlewares(
			middleware.Recover(),
			middleware.Logging(),
			middleware.RateLimit(queries),
			middleware.UserLoader(userService, groupService, cfg),
		),
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if h == nil {
				return
			}
			// Pre-checkout query handling
			if update.PreCheckoutQuery != nil {
				h.HandlePreCheckout(ctx, b, update)
				return
			}
		}),
	}

	b, err := bot.New(cfg.BotToken, opts...)
	if err != nil {
		slog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	// Get bot info
	me, err := b.GetMe(ctx)
	if err != nil {
		slog.Error("failed to get bot info", "error", err)
		os.Exit(1)
	}

	slog.Info("bot info retrieved", "id", me.ID, "username", me.Username)

	// Initialize telegram logger
	tgLogger := telegram.NewTelegramLogger(b, cfg)

	// Initialize handler
	h = handler.New(handler.Deps{
		Bot:             b,
		Cfg:             cfg,
		UserService:     userService,
		GroupService:    groupService,
		SessionService:  sessionService,
		BillingService:  billingService,
		PaymentService:  paymentService,
		PromoService:    promoService,
		PremiumService:  premiumService,
		OpenRouter:      openRouter,
		SkysmartService: skysmartService,
		Queries:         queries,
		TgLogger:        tgLogger,
		BotUsername:      me.Username,
	})

	// Register all handlers
	h.Register()

	// Register default text handler for AI messages
	b.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}
		// Skip commands
		if len(update.Message.Text) > 0 && update.Message.Text[0] == '/' {
			return
		}
		// Route to appropriate text handler
		if update.Message.Chat.Type == "private" {
			h.HandleTextPrivate(ctx, b, update)
		} else {
			h.HandleTextGroup(ctx, b, update)
		}
	})

	// Start stale request cleanup goroutine
	go func() {
		ticker := time.NewTicker(config.StaleRequestCleanup)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := queries.CleanupStaleRequests(context.Background()); err != nil {
					slog.Error("cleanup stale requests", "error", err)
				}
			}
		}
	}()

	// Start bot
	slog.Info("starting bot", "username", me.Username, "id", me.ID)
	b.Start(ctx)

	// Graceful shutdown
	slog.Info("bot stopped gracefully")
}
