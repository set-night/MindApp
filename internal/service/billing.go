package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/shopspring/decimal"
)

type BillingService struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewBillingService(db *pgxpool.Pool, queries *sqlc.Queries) *BillingService {
	return &BillingService{db: db, queries: queries}
}

// ProcessUserTransaction atomically deducts from user balance and records the transaction.
func (s *BillingService) ProcessUserTransaction(ctx context.Context, userID int64, baseCost float64, markupPercent float64, description string) (totalCost decimal.Decimal, newBalance decimal.Decimal, err error) {
	markup := decimal.NewFromFloat(1 + markupPercent/100)
	totalCost = decimal.NewFromFloat(baseCost).Mul(markup)
	negAmount := totalCost.Neg()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Lock user row and check balance
	user, err := qtx.GetUserForUpdate(ctx, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("lock user: %w", err)
	}

	if user.Balance.Add(negAmount).LessThan(decimal.Zero) {
		return decimal.Zero, decimal.Zero, domain.ErrInsufficientBalance
	}

	newBalance, err = qtx.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{
		ID:      userID,
		Balance: negAmount,
	})
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("update balance: %w", err)
	}

	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		UserID:      &userID,
		Amount:      negAmount,
		TxType:      string(domain.TxTypeDebit),
		Description: description,
	})
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("create transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("commit: %w", err)
	}

	return totalCost, newBalance, nil
}

// ProcessGroupTransaction atomically deducts from group balance.
func (s *BillingService) ProcessGroupTransaction(ctx context.Context, groupID int64, baseCost float64, markupPercent float64, description string) (totalCost decimal.Decimal, newBalance decimal.Decimal, err error) {
	markup := decimal.NewFromFloat(1 + markupPercent/100)
	totalCost = decimal.NewFromFloat(baseCost).Mul(markup)
	negAmount := totalCost.Neg()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	group, err := qtx.GetGroupForUpdate(ctx, groupID)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("lock group: %w", err)
	}

	if group.Balance.Add(negAmount).LessThan(decimal.Zero) {
		return decimal.Zero, decimal.Zero, domain.ErrInsufficientBalance
	}

	newBalance, err = qtx.UpdateGroupBalance(ctx, sqlc.UpdateGroupBalanceParams{
		ID:      groupID,
		Balance: negAmount,
	})
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("update group balance: %w", err)
	}

	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		GroupID:     &groupID,
		Amount:      negAmount,
		TxType:      string(domain.TxTypeDebit),
		Description: description,
	})
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("create transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("commit: %w", err)
	}

	return totalCost, newBalance, nil
}

// CreditUser adds funds to user balance.
func (s *BillingService) CreditUser(ctx context.Context, userID int64, amount decimal.Decimal, description string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	_, err = qtx.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{
		ID:      userID,
		Balance: amount,
	})
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		UserID:      &userID,
		Amount:      amount,
		TxType:      string(domain.TxTypeCredit),
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// CreditGroup adds funds to group balance.
func (s *BillingService) CreditGroup(ctx context.Context, groupID int64, amount decimal.Decimal, description string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	_, err = qtx.UpdateGroupBalance(ctx, sqlc.UpdateGroupBalanceParams{
		ID:      groupID,
		Balance: amount,
	})
	if err != nil {
		return fmt.Errorf("update group balance: %w", err)
	}

	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		GroupID:     &groupID,
		Amount:      amount,
		TxType:      string(domain.TxTypeCredit),
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// TransferUserToGroup transfers funds from user to group.
func (s *BillingService) TransferUserToGroup(ctx context.Context, userID, groupID int64, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return domain.ErrInvalidAmount
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Lock and check user balance
	user, err := qtx.GetUserForUpdate(ctx, userID)
	if err != nil {
		return fmt.Errorf("lock user: %w", err)
	}
	if user.Balance.LessThan(amount) {
		return domain.ErrInsufficientBalance
	}

	negAmount := amount.Neg()
	_, err = qtx.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{
		ID:      userID,
		Balance: negAmount,
	})
	if err != nil {
		return fmt.Errorf("deduct user balance: %w", err)
	}

	_, err = qtx.UpdateGroupBalance(ctx, sqlc.UpdateGroupBalanceParams{
		ID:      groupID,
		Balance: amount,
	})
	if err != nil {
		return fmt.Errorf("credit group balance: %w", err)
	}

	description := fmt.Sprintf("Transfer to group %d", groupID)
	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		UserID:      &userID,
		Amount:      negAmount,
		TxType:      string(domain.TxTypeDebit),
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("create user transaction: %w", err)
	}

	groupDesc := fmt.Sprintf("Transfer from user %d", userID)
	_, err = qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		GroupID:     &groupID,
		Amount:      amount,
		TxType:      string(domain.TxTypeCredit),
		Description: groupDesc,
	})
	if err != nil {
		return fmt.Errorf("create group transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// CalculateCost calculates AI request cost with markup.
func CalculateCost(promptTokens, completionTokens int, promptPrice, completionPrice float64, markupPercent float64) decimal.Decimal {
	promptCost := decimal.NewFromFloat(float64(promptTokens) * promptPrice / 1_000_000)
	completionCost := decimal.NewFromFloat(float64(completionTokens) * completionPrice / 1_000_000)
	baseCost := promptCost.Add(completionCost)
	markup := decimal.NewFromFloat(1 + markupPercent/100)
	return baseCost.Mul(markup)
}
