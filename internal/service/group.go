package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/set-night/mindapp/internal/domain"
	"github.com/set-night/mindapp/internal/repository/sqlc"
)

type GroupService struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewGroupService(db *pgxpool.Pool, queries *sqlc.Queries) *GroupService {
	return &GroupService{db: db, queries: queries}
}

func (s *GroupService) FindOrCreate(ctx context.Context, telegramID int64, groupUsername, groupName string) (*domain.Group, bool, error) {
	row, err := s.queries.GetGroupByTelegramID(ctx, telegramID)
	if err == nil {
		group := rowToGroup(row)
		return group, false, nil
	}
	if err != pgx.ErrNoRows {
		return nil, false, fmt.Errorf("get group: %w", err)
	}

	row, err = s.queries.CreateGroup(ctx, sqlc.CreateGroupParams{
		TelegramID:   telegramID,
		GroupUsername: groupUsername,
		GroupName:    groupName,
	})
	if err != nil {
		return nil, false, fmt.Errorf("create group: %w", err)
	}

	group := rowToGroup(row)
	return group, true, nil
}

func (s *GroupService) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Group, error) {
	row, err := s.queries.GetGroupByTelegramID(ctx, telegramID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group: %w", err)
	}
	return rowToGroup(row), nil
}

func (s *GroupService) UpdateInfo(ctx context.Context, groupID int64, groupUsername, groupName string) error {
	return s.queries.UpdateGroupInfo(ctx, sqlc.UpdateGroupInfoParams{
		ID:           groupID,
		GroupUsername: groupUsername,
		GroupName:    groupName,
	})
}

func (s *GroupService) AddContextMessage(ctx context.Context, groupID int64, role, text string) error {
	return s.queries.AddGroupContextMessage(ctx, sqlc.AddGroupContextMessageParams{
		GroupID: groupID,
		Role:    role,
		Text:    text,
	})
}

func (s *GroupService) GetContextMessages(ctx context.Context, groupID int64) ([]domain.GroupContextMessage, error) {
	rows, err := s.queries.GetGroupContextMessages(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get context messages: %w", err)
	}
	msgs := make([]domain.GroupContextMessage, len(rows))
	for i, r := range rows {
		msgs[i] = domain.GroupContextMessage{
			ID:        r.ID,
			GroupID:   r.GroupID,
			Role:      r.Role,
			Text:      r.Text,
			CreatedAt: pgTimestamptzToTime(r.CreatedAt),
		}
	}
	return msgs, nil
}

func (s *GroupService) ClearContext(ctx context.Context, groupID int64) error {
	return s.queries.DeleteGroupContextMessages(ctx, groupID)
}

func rowToGroup(row sqlc.Group) *domain.Group {
	return &domain.Group{
		ID:              row.ID,
		TelegramID:      row.TelegramID,
		Balance:         row.Balance,
		GroupUsername:    row.GroupUsername,
		GroupName:       row.GroupName,
		PremiumUntil:    pgTimestamptzToTimePtr(row.PremiumUntil),
		LastInteraction: pgTimestamptzToTime(row.LastInteraction),
		ThreadID:        int32PtrToIntPtr(row.ThreadID),
		SelectedModel:   row.SelectedModel,
		ShowCost:        row.ShowCost,
		ContextEnabled:  row.ContextEnabled,
		CreatedAt:       pgTimestamptzToTime(row.CreatedAt),
		UpdatedAt:       pgTimestamptzToTime(row.UpdatedAt),
	}
}
