package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Group struct {
	ID              int64
	TelegramID      int64
	Balance         decimal.Decimal
	GroupUsername    string
	GroupName       string
	PremiumUntil    *time.Time
	LastInteraction time.Time
	ThreadID        *int
	SelectedModel   string
	ShowCost        bool
	ContextEnabled  bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (g *Group) IsPremium() bool {
	if g.PremiumUntil == nil {
		return false
	}
	return g.PremiumUntil.After(time.Now())
}

type GroupContextMessage struct {
	ID        int64
	GroupID   int64
	Role      string
	Text      string
	CreatedAt time.Time
}
