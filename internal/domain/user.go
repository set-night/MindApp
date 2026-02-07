package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type User struct {
	ID              int64
	TelegramID      int64
	IsAdmin         bool
	FirstName       string
	Username        string
	Balance         decimal.Decimal
	ReferralCode    string
	ReferralBalance decimal.Decimal
	ReferredByID    *int64
	PremiumUntil    *time.Time
	ActiveSessionID *int64
	LastInteraction time.Time

	// Settings
	SelectedModel    string
	FavoriteModels   []string
	Temperature      float64
	ShowCost         bool
	SendUserInfo     bool
	ContextEnabled   bool
	SessionTimeoutMs int

	LastSkysmart time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) IsPremium() bool {
	if u.PremiumUntil == nil {
		return false
	}
	return u.PremiumUntil.After(time.Now())
}
