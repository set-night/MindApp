package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/shopspring/decimal"
)

type PromoService struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPromoService(db *pgxpool.Pool, queries *sqlc.Queries) *PromoService {
	return &PromoService{db: db, queries: queries}
}

func (s *PromoService) Activate(ctx context.Context, code string, userID int64) (decimal.Decimal, error) {
	code = strings.ToUpper(strings.TrimSpace(code))

	row, err := s.queries.GetPromoByCode(ctx, code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return decimal.Zero, domain.ErrPromoNotFound
		}
		return decimal.Zero, fmt.Errorf("get promo: %w", err)
	}

	// Check if max uses reached
	if int(row.ActivationCount) >= int(row.MaxUses) {
		return decimal.Zero, domain.ErrPromoMaxUses
	}

	// Check if already activated by this user
	activated, err := s.queries.CheckPromoActivation(ctx, sqlc.CheckPromoActivationParams{
		PromoID: row.ID,
		UserID:  userID,
	})
	if err != nil {
		return decimal.Zero, fmt.Errorf("check activation: %w", err)
	}
	if activated {
		return decimal.Zero, domain.ErrPromoAlreadyUsed
	}

	// Activate in a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	if err := qtx.CreatePromoActivation(ctx, sqlc.CreatePromoActivationParams{
		PromoID: row.ID,
		UserID:  userID,
	}); err != nil {
		return decimal.Zero, fmt.Errorf("create activation: %w", err)
	}

	_, err = qtx.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{
		ID:      userID,
		Balance: row.Amount,
	})
	if err != nil {
		return decimal.Zero, fmt.Errorf("update balance: %w", err)
	}

	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		UserID:      &userID,
		Amount:      row.Amount,
		TxType:      string(domain.TxTypeCredit),
		Description: fmt.Sprintf("Promo code: %s", code),
	})
	if err != nil {
		return decimal.Zero, fmt.Errorf("create transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return decimal.Zero, fmt.Errorf("commit: %w", err)
	}

	return row.Amount, nil
}

func (s *PromoService) Create(ctx context.Context, amount decimal.Decimal, count int, comment string, createdBy int64) ([]string, error) {
	codes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		code, err := generatePromoCode()
		if err != nil {
			return nil, fmt.Errorf("generate promo code: %w", err)
		}
		_, err = s.queries.CreatePromo(ctx, sqlc.CreatePromoParams{
			Code:      strings.ToUpper(code),
			Amount:    amount,
			Comment:   comment,
			MaxUses:   1,
			CreatedBy: createdBy,
		})
		if err != nil {
			return nil, fmt.Errorf("create promo: %w", err)
		}
		codes = append(codes, code)
	}
	return codes, nil
}

const promoCodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func generatePromoCode() (string, error) {
	code := make([]byte, 16)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(promoCodeCharset))))
		if err != nil {
			return "", err
		}
		code[i] = promoCodeCharset[n.Int64()]
	}
	return string(code), nil
}
