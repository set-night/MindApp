package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// Core
	BotToken      string `env:"BOT_TOKEN,required"`
	DatabaseURL   string `env:"DATABASE_URL,required"`
	OpenRouterKey string `env:"OPENROUTER_API_KEY,required"`

	// Payment: Telegram Stars
	StarsEnabled bool `env:"STARS_ENABLED" envDefault:"true"`

	// Payment: Cryptomus (Crypto)
	CryptomusEnabled    bool   `env:"CRYPTOMUS_ENABLED" envDefault:"false"`
	CryptomusMerchantID string `env:"CRYPTOMUS_MERCHANT_ID"`
	CryptomusAPIKey     string `env:"CRYPTOMUS_API_KEY"`
	CryptomusURL        string `env:"CRYPTOMUS_API_URL" envDefault:"https://api.cryptomus.com/v1"`

	// Payment: FunPay (Cards) — external link only
	FunPayEnabled bool   `env:"FUNPAY_ENABLED" envDefault:"false"`
	FunPayURL     string `env:"FUNPAY_URL" envDefault:"https://funpay.com"`

	// Admin
	AdminIDs []int64 `env:"ADMIN_IDS" envSeparator:","`

	// Markup
	MarkupPercentNormal  float64 `env:"MARKUP_PERCENT_NORMAL" envDefault:"30"`
	MarkupPercentPremium float64 `env:"MARKUP_PERCENT_PREMIUM" envDefault:"15"`

	// Server
	Port int `env:"PORT" envDefault:"3000"`

	// Bot behavior
	DropPendingUpdates bool `env:"BOT_DROP_PENDING_UPDATES" envDefault:"false"`

	// Telegram logging
	LogTelegramChatID      int64 `env:"LOG_TELEGRAM_CHAT_ID"`
	LogTopicError          int   `env:"LOG_TOPIC_ERROR"`
	LogTopicBalanceTopUp   int   `env:"LOG_TOPIC_BALANCE_TOPUP"`
	LogTopicPromoActivate  int   `env:"LOG_TOPIC_PROMO_ACTIVATE"`
	LogTopicPremiumPurchase int  `env:"LOG_TOPIC_PREMIUM_PURCHASE"`
	LogTopicFreeBalance    int   `env:"LOG_TOPIC_FREE_BALANCE"`
	LogTopicRegistration   int   `env:"LOG_TOPIC_REGISTRATION"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (c *Config) IsAdmin(telegramID int64) bool {
	for _, id := range c.AdminIDs {
		if id == telegramID {
			return true
		}
	}
	return false
}

func (c *Config) AdminIDsString() string {
	parts := make([]string, len(c.AdminIDs))
	for i, id := range c.AdminIDs {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ",")
}
