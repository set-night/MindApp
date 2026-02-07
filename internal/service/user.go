package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/shopspring/decimal"
)

type UserService struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewUserService(db *pgxpool.Pool, queries *sqlc.Queries) *UserService {
	return &UserService{db: db, queries: queries}
}

func (s *UserService) FindOrCreate(ctx context.Context, telegramID int64, firstName, username string, referredByCode string, isAdmin bool) (*domain.User, bool, error) {
	// Try to find existing user
	row, err := s.queries.GetUserByTelegramID(ctx, telegramID)
	if err == nil {
		user := rowToUser(row)
		return user, false, nil
	}
	if err != pgx.ErrNoRows {
		return nil, false, fmt.Errorf("get user: %w", err)
	}

	// Create new user
	refCode, err := generateUniqueReferralCode(ctx, s.queries)
	if err != nil {
		return nil, false, fmt.Errorf("generate referral code: %w", err)
	}

	var referredByID *int64
	if referredByCode != "" {
		referrer, err := s.queries.GetUserByReferralCode(ctx, referredByCode)
		if err == nil {
			referredByID = &referrer.ID
		}
	}

	row, err = s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		TelegramID:   telegramID,
		FirstName:    firstName,
		Username:     username,
		ReferralCode: refCode,
		ReferredByID: referredByID,
		IsAdmin:      isAdmin,
	})
	if err != nil {
		return nil, false, fmt.Errorf("create user: %w", err)
	}

	// Grant referral bonus
	if referredByID != nil {
		bonus := decimal.NewFromFloat(1.0) // $1 registration bonus
		if err := s.queries.UpdateUserReferralBalance(ctx, sqlc.UpdateUserReferralBalanceParams{
			ID:              *referredByID,
			ReferralBalance: bonus,
		}); err != nil {
			slog.Error("failed to grant referral bonus", "error", err, "referrer_id", *referredByID)
		}
	}

	user := rowToUser(row)
	return user, true, nil
}

func (s *UserService) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	row, err := s.queries.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return rowToUser(row), nil
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return rowToUser(row), nil
}

func (s *UserService) UpdateInfo(ctx context.Context, userID int64, firstName, username string) error {
	return s.queries.UpdateUserInfo(ctx, sqlc.UpdateUserInfoParams{
		ID:        userID,
		FirstName: firstName,
		Username:  username,
	})
}

func (s *UserService) UpdateLastInteraction(ctx context.Context, userID int64) error {
	return s.queries.UpdateUserLastInteraction(ctx, userID)
}

// rowToUser converts a sqlc-generated row to a domain.User.
func rowToUser(row sqlc.User) *domain.User {
	return &domain.User{
		ID:               row.ID,
		TelegramID:       row.TelegramID,
		IsAdmin:          row.IsAdmin,
		FirstName:        row.FirstName,
		Username:         row.Username,
		Balance:          row.Balance,
		ReferralCode:     row.ReferralCode,
		ReferralBalance:  row.ReferralBalance,
		ReferredByID:     row.ReferredByID,
		PremiumUntil:     pgTimestamptzToTimePtr(row.PremiumUntil),
		ActiveSessionID:  row.ActiveSessionID,
		LastInteraction:  pgTimestamptzToTime(row.LastInteraction),
		SelectedModel:    row.SelectedModel,
		FavoriteModels:   row.FavoriteModels,
		Temperature:      decimalToFloat(row.Temperature),
		ShowCost:         row.ShowCost,
		SendUserInfo:     row.SendUserInfo,
		ContextEnabled:   row.ContextEnabled,
		SessionTimeoutMs: int(row.SessionTimeoutMs),
		LastSkysmart:     pgTimestamptzToTime(row.LastSkysmart),
		CreatedAt:        pgTimestamptzToTime(row.CreatedAt),
		UpdatedAt:        pgTimestamptzToTime(row.UpdatedAt),
	}
}
