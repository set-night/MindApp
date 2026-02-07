package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type TxType string

const (
	TxTypeDebit  TxType = "debit"
	TxTypeCredit TxType = "credit"
)

type Transaction struct {
	ID          int64
	UserID      *int64
	GroupID     *int64
	Amount      decimal.Decimal
	TxType      TxType
	Description string
	CreatedAt   time.Time
}
