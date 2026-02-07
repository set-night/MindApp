package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Promo struct {
	ID              int64
	Code            string
	Amount          decimal.Decimal
	Comment         string
	MaxUses         int
	CreatedBy       int64
	CreatedAt       time.Time
	ActivationCount int // computed field
}

type PromoActivation struct {
	ID          int64
	PromoID     int64
	UserID      int64
	ActivatedAt time.Time
}
