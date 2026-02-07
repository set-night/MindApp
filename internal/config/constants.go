package config

import "time"

const (
	// Request cooldowns
	CooldownRegular = 9 * time.Second
	CooldownPremium = 5 * time.Second
	CooldownFree    = 15 * time.Second

	// Session message limits
	MaxMessagesRegular = 100
	MaxMessagesPremium = 500

	// Session count limits
	MaxSessionsRegular = 3
	MaxSessionsPremium = 50

	// Balance thresholds
	LowBalanceThreshold = 0.2
	LowPriceThreshold   = 0.002

	// Telegram limits
	MaxTelegramMessageLen = 4096

	// AI request timeout
	RequestTimeout = 90 * time.Second

	// Model cache duration
	ModelCacheDuration = 1 * time.Hour

	// Telegram Stars conversion rate
	XTRToDollarRate = 0.013

	// Default AI model
	DefaultModel = "z-ai/glm-4.5-air:free"

	// Premium pricing (USD)
	PremiumPrice1Month  = 2.0
	PremiumPrice6Month  = 10.0
	PremiumPrice12Month = 15.0

	// Premium durations
	PremiumDuration1Month  = 30 * 24 * time.Hour
	PremiumDuration6Month  = 180 * 24 * time.Hour
	PremiumDuration12Month = 360 * 24 * time.Hour

	// Rate limits (per minute)
	RateLimitRegular = 6
	RateLimitPremium = 11

	// Referral bonus on registration (USD)
	ReferralBonusRegistration = 1.0

	// Stale request cleanup interval
	StaleRequestCleanup = 60 * time.Second
	StaleRequestAge     = 3 * time.Minute

	// Default temperature
	DefaultTemperature = 1.0

	// Default session timeout
	DefaultSessionTimeoutMs = 600000

	// Models per page
	ModelsPerPage = 5

	// Sessions per page
	SessionsPerPage = 5
)

// TemperatureOptions available for premium users.
var TemperatureOptions = []float64{0.1, 0.4, 0.7, 1.0, 1.5, 2.0}

// SessionTimeoutOptions in milliseconds.
var SessionTimeoutOptions = []int{0, 600000, 1800000, 3600000, 86400000}

// StarPaymentAmounts in USD.
var StarPaymentAmounts = []int{1, 5, 15, 50}

// CryptomusPaymentAmounts in USD.
var CryptomusPaymentAmounts = []float64{0.5, 1, 5, 10, 30, 100}
