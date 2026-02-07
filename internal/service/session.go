package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/repository/sqlc"
	"github.com/shopspring/decimal"
)

type SessionService struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewSessionService(db *pgxpool.Pool, queries *sqlc.Queries) *SessionService {
	return &SessionService{db: db, queries: queries}
}

func (s *SessionService) FindOrCreate(ctx context.Context, user *domain.User) (*domain.ChatSession, error) {
	if user.ActiveSessionID != nil {
		row, err := s.queries.GetSessionByID(ctx, *user.ActiveSessionID)
		if err == nil {
			return rowToSession(row), nil
		}
		if err != pgx.ErrNoRows {
			return nil, fmt.Errorf("get session: %w", err)
		}
	}
	return s.CreateNew(ctx, user)
}

func (s *SessionService) CreateNew(ctx context.Context, user *domain.User) (*domain.ChatSession, error) {
	// Enforce session limit
	maxSessions := config.MaxSessionsRegular
	if user.IsPremium() {
		maxSessions = config.MaxSessionsPremium
	}

	count, err := s.queries.CountSessionsByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("count sessions: %w", err)
	}

	if count >= int64(maxSessions) {
		toDelete := count - int64(maxSessions) + 1
		if err := s.queries.DeleteOldestUserSessions(ctx, sqlc.DeleteOldestUserSessionsParams{
			UserID: user.ID,
			Limit:  int32(toDelete),
		}); err != nil {
			return nil, fmt.Errorf("delete oldest sessions: %w", err)
		}
	}

	row, err := s.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:      user.ID,
		Model:       user.SelectedModel,
		Temperature: decimal.NewFromFloat(user.Temperature),
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	if err := s.queries.SetUserActiveSession(ctx, sqlc.SetUserActiveSessionParams{
		ID:              user.ID,
		ActiveSessionID: &row.ID,
	}); err != nil {
		return nil, fmt.Errorf("set active session: %w", err)
	}

	return rowToSession(row), nil
}

func (s *SessionService) Reset(ctx context.Context, user *domain.User) (*domain.ChatSession, error) {
	user.ActiveSessionID = nil
	if err := s.queries.SetUserActiveSession(ctx, sqlc.SetUserActiveSessionParams{
		ID:              user.ID,
		ActiveSessionID: nil,
	}); err != nil {
		return nil, fmt.Errorf("clear active session: %w", err)
	}
	return s.CreateNew(ctx, user)
}

func (s *SessionService) GetByID(ctx context.Context, sessionID int64) (*domain.ChatSession, error) {
	row, err := s.queries.GetSessionByID(ctx, sessionID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return rowToSession(row), nil
}

func (s *SessionService) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.ChatSession, error) {
	rows, err := s.queries.GetSessionsByUserID(ctx, sqlc.GetSessionsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	sessions := make([]domain.ChatSession, len(rows))
	for i, r := range rows {
		sessions[i] = *rowToSession(r)
	}
	return sessions, nil
}

func (s *SessionService) CountByUser(ctx context.Context, userID int64) (int64, error) {
	return s.queries.CountSessionsByUserID(ctx, userID)
}

func (s *SessionService) Delete(ctx context.Context, sessionID int64) error {
	return s.queries.DeleteSession(ctx, sessionID)
}

func (s *SessionService) DeleteAll(ctx context.Context, userID int64) error {
	if err := s.queries.SetUserActiveSession(ctx, sqlc.SetUserActiveSessionParams{
		ID:              userID,
		ActiveSessionID: nil,
	}); err != nil {
		return fmt.Errorf("clear active session: %w", err)
	}
	return s.queries.DeleteAllUserSessions(ctx, userID)
}

func (s *SessionService) SwitchTo(ctx context.Context, userID, sessionID int64) error {
	return s.queries.SetUserActiveSession(ctx, sqlc.SetUserActiveSessionParams{
		ID:              userID,
		ActiveSessionID: &sessionID,
	})
}

func (s *SessionService) IsExpired(user *domain.User) bool {
	if user.SessionTimeoutMs <= 0 {
		return false
	}
	timeout := time.Duration(user.SessionTimeoutMs) * time.Millisecond
	return time.Since(user.LastInteraction) > timeout
}

func (s *SessionService) AddMessage(ctx context.Context, sessionID int64, role, text string, images []string, isSystem bool) (*domain.SessionMessage, error) {
	if images == nil {
		images = []string{}
	}
	row, err := s.queries.AddSessionMessage(ctx, sqlc.AddSessionMessageParams{
		SessionID: sessionID,
		Role:      role,
		Text:      text,
		Images:    images,
		IsSystem:  isSystem,
	})
	if err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}
	return &domain.SessionMessage{
		ID:        row.ID,
		SessionID: row.SessionID,
		Role:      row.Role,
		Text:      row.Text,
		Images:    row.Images,
		IsSystem:  row.IsSystem,
		CreatedAt: pgTimestamptzToTime(row.CreatedAt),
	}, nil
}

func (s *SessionService) GetMessages(ctx context.Context, sessionID int64) ([]domain.SessionMessage, error) {
	rows, err := s.queries.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	msgs := make([]domain.SessionMessage, len(rows))
	for i, r := range rows {
		msgs[i] = domain.SessionMessage{
			ID:        r.ID,
			SessionID: r.SessionID,
			Role:      r.Role,
			Text:      r.Text,
			Images:    r.Images,
			IsSystem:  r.IsSystem,
			CreatedAt: pgTimestamptzToTime(r.CreatedAt),
		}
	}
	return msgs, nil
}

func (s *SessionService) CountMessages(ctx context.Context, sessionID int64) (int64, error) {
	return s.queries.CountSessionMessages(ctx, sessionID)
}

func (s *SessionService) GetFirstMessage(ctx context.Context, sessionID int64) (*domain.SessionMessage, error) {
	row, err := s.queries.GetFirstSessionMessage(ctx, sessionID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get first message: %w", err)
	}
	return &domain.SessionMessage{
		ID:        row.ID,
		SessionID: row.SessionID,
		Role:      row.Role,
		Text:      row.Text,
	}, nil
}

func (s *SessionService) AddMessageFile(ctx context.Context, messageID int64, fileType, url, name string) error {
	return s.queries.AddMessageFile(ctx, sqlc.AddMessageFileParams{
		MessageID: messageID,
		FileType:  fileType,
		Url:       url,
		Name:      name,
	})
}

func rowToSession(row sqlc.ChatSession) *domain.ChatSession {
	return &domain.ChatSession{
		ID:          row.ID,
		UserID:      row.UserID,
		Model:       row.Model,
		Temperature: decimalToFloat(row.Temperature),
		CreatedAt:   pgTimestamptzToTime(row.CreatedAt),
		UpdatedAt:   pgTimestamptzToTime(row.UpdatedAt),
	}
}
