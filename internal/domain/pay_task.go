package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type PayTask struct {
	ID             int64
	Title          string
	TelegramLink   string
	ChannelID      string
	Reward         decimal.Decimal
	TimeLimit      *time.Time
	MaxPeople      *int
	CompletedCount int // computed field
}

type PayTaskCompletion struct {
	ID          int64
	TaskID      int64
	UserID      int64
	CompletedAt time.Time
}
