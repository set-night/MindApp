package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/shopspring/decimal"
)

type PaymentService struct {
	db         *pgxpool.Pool
	queries    *sqlc.Queries
	cfg        *config.Config
	httpClient *http.Client
}

func NewPaymentService(db *pgxpool.Pool, queries *sqlc.Queries, cfg *config.Config) *PaymentService {
	return &PaymentService{
		db:         db,
		queries:    queries,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type CryptomusInvoice struct {
	PaymentURL string
	InvoiceID  string
}

func (s *PaymentService) CreateCryptomusInvoice(ctx context.Context, userTelegramID int64, amount float64) (*CryptomusInvoice, error) {
	orderID := uuid.New().String()
	amountStr := fmt.Sprintf("%.2f", amount)

	payload := map[string]interface{}{
		"amount":   amountStr,
		"currency": "USD",
		"order_id": orderID,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	sign := createCryptomusSign(payloadJSON, s.cfg.CryptomusAPIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.CryptomusURL+"/payment", bytes.NewReader(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("merchant", s.cfg.CryptomusMerchantID)
	req.Header.Set("sign", sign)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Result struct {
			URL  string `json:"url"`
			UUID string `json:"uuid"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Store invoice
	_, err = s.queries.CreateInvoice(ctx, sqlc.CreateInvoiceParams{
		UserTelegramID:     userTelegramID,
		Amount:             decimal.NewFromFloat(amount),
		CryptomusInvoiceID: result.Result.UUID,
	})
	if err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	return &CryptomusInvoice{
		PaymentURL: result.Result.URL,
		InvoiceID:  result.Result.UUID,
	}, nil
}

func (s *PaymentService) CheckCryptomusPayment(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	payload := map[string]interface{}{
		"uuid": invoiceID,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	sign := createCryptomusSign(payloadJSON, s.cfg.CryptomusAPIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.CryptomusURL+"/payment/info", bytes.NewReader(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("merchant", s.cfg.CryptomusMerchantID)
	req.Header.Set("sign", sign)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Result struct {
			PaymentStatus string `json:"payment_status"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Get invoice from DB
	inv, err := s.queries.GetInvoiceByCryptomusID(ctx, invoiceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}

	// Update status based on Cryptomus response
	var newStatus domain.InvoiceStatus
	switch result.Result.PaymentStatus {
	case "paid", "paid_over":
		newStatus = domain.InvoiceStatusPaid
	case "wrong_amount", "fail", "cancel", "system_fail", "refund_process", "refund_fail", "refund_paid":
		newStatus = domain.InvoiceStatusFailed
	default:
		newStatus = domain.InvoiceStatusPending
	}

	if err := s.queries.UpdateInvoiceStatus(ctx, sqlc.UpdateInvoiceStatusParams{
		ID:     inv.ID,
		Status: string(newStatus),
	}); err != nil {
		return nil, fmt.Errorf("update invoice status: %w", err)
	}

	return &domain.Invoice{
		ID:                 inv.ID,
		UserTelegramID:     inv.UserTelegramID,
		Amount:             inv.Amount,
		CryptomusInvoiceID: inv.CryptomusInvoiceID,
		Status:             newStatus,
		CreatedAt:          pgTimestamptzToTime(inv.CreatedAt),
	}, nil
}

func (s *PaymentService) GetPendingInvoice(ctx context.Context, userTelegramID int64) (*domain.Invoice, error) {
	row, err := s.queries.GetPendingInvoiceByUser(ctx, userTelegramID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get pending invoice: %w", err)
	}
	return &domain.Invoice{
		ID:                 row.ID,
		UserTelegramID:     row.UserTelegramID,
		Amount:             row.Amount,
		CryptomusInvoiceID: row.CryptomusInvoiceID,
		Status:             domain.InvoiceStatus(row.Status),
		CreatedAt:          pgTimestamptzToTime(row.CreatedAt),
	}, nil
}

func (s *PaymentService) DeleteInvoice(ctx context.Context, invoiceID int64) error {
	return s.queries.DeleteInvoice(ctx, invoiceID)
}

// CalculateStarAmount converts USD to Telegram Stars (XTR).
func CalculateStarAmount(usdAmount int) int {
	return int(float64(usdAmount) / config.XTRToDollarRate)
}

// CalculateUSDFromStars converts Telegram Stars to USD.
func CalculateUSDFromStars(stars int) float64 {
	return float64(stars) * config.XTRToDollarRate
}

func createCryptomusSign(payload []byte, apiKey string) string {
	encoded := base64.StdEncoding.EncodeToString(payload)
	hash := md5.Sum([]byte(encoded + apiKey))
	return hex.EncodeToString(hash[:])
}
