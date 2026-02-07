package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Prompt struct {
	ID          int64
	Title       string
	Description string
	PromptText  string
	IsOfficial  bool
	OwnerID     *int64
	Price       decimal.Decimal
	CreatedAt   time.Time
}
