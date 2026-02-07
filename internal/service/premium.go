package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/shopspring/decimal"
)

type PremiumService struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPremiumService(db *pgxpool.Pool, queries *sqlc.Queries) *PremiumService {
	return &PremiumService{db: db, queries: queries}
}

type PremiumOption struct {
	Label    string
	Price    float64
	Duration time.Duration
}

func GetPremiumOptions() []PremiumOption {
	return []PremiumOption{
		{Label: "1 month", Price: config.PremiumPrice1Month, Duration: config.PremiumDuration1Month},
		{Label: "6 months", Price: config.PremiumPrice6Month, Duration: config.PremiumDuration6Month},
		{Label: "12 months", Price: config.PremiumPrice12Month, Duration: config.PremiumDuration12Month},
	}
}

func (s *PremiumService) Purchase(ctx context.Context, userID int64, option PremiumOption) error {
	price := decimal.NewFromFloat(option.Price)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	user, err := qtx.GetUserForUpdate(ctx, userID)
	if err != nil {
		return fmt.Errorf("lock user: %w", err)
	}

	if user.Balance.LessThan(price) {
		return domain.ErrInsufficientBalance
	}

	negPrice := price.Neg()
	_, err = qtx.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{
		ID:      userID,
		Balance: negPrice,
	})
	if err != nil {
		return fmt.Errorf("deduct balance: %w", err)
	}

	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		UserID:      &userID,
		Amount:      negPrice,
		TxType:      string(domain.TxTypeDebit),
		Description: fmt.Sprintf("Premium subscription: %s", option.Label),
	})
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	// Calculate new premium date
	var newPremiumUntil time.Time
	if user.PremiumUntil.Valid && user.PremiumUntil.Time.After(time.Now()) {
		newPremiumUntil = user.PremiumUntil.Time.Add(option.Duration)
	} else {
		newPremiumUntil = time.Now().Add(option.Duration)
	}

	if err := qtx.SetUserPremiumUntil(ctx, sqlc.SetUserPremiumUntilParams{
		ID:           userID,
		PremiumUntil: timeToPgTimestamptz(newPremiumUntil),
	}); err != nil {
		return fmt.Errorf("set premium: %w", err)
	}

	return tx.Commit(ctx)
}
