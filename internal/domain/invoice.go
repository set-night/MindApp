package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type InvoiceStatus string

const (
	InvoiceStatusPending InvoiceStatus = "pending"
	InvoiceStatusPaid    InvoiceStatus = "paid"
	InvoiceStatusFailed  InvoiceStatus = "failed"
)

type Invoice struct {
	ID                 int64
	UserTelegramID     int64
	Amount             decimal.Decimal
	CryptomusInvoiceID string
	Status             InvoiceStatus
	CreatedAt          time.Time
}
