package notification

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID, onlyUnread bool) ([]Model, error)
	UnreadCount(ctx context.Context, userID common.UserID) (int64, error)
	MarkRead(ctx context.Context, userID common.UserID, id common.NotificationID) error
	Dismiss(ctx context.Context, userID common.UserID, id common.NotificationID) error

	// CreateIfNotExists creates a notification only if no active
	// (non-dismissed) one already exists for the same user+type+entity,
	// returning whether it actually created one. This is what the
	// real-time budget trigger and batch scanner call, so the same
	// situation doesn't spam a new notification on every check.
	CreateIfNotExists(ctx context.Context, userID common.UserID, in CreateInput) (bool, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	Type              Type
	Title             string
	Message           string
	RelatedEntityType string
	RelatedEntityID   string
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.Title == "" {
		return Model{}, apperr.ValidationErr("title is required")
	}

	m := Model{
		UserID:            userID,
		Type:              in.Type,
		Title:             in.Title,
		Message:           in.Message,
		RelatedEntityType: in.RelatedEntityType,
		RelatedEntityID:   in.RelatedEntityID,
		CreatedAt:         time.Now().UTC(),
	}

	id, err := s.repo.Create(ctx, m)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to create notification", err)
	}
	m.ID = id

	return m, nil
}

func (s *service) List(ctx context.Context, userID common.UserID, onlyUnread bool) ([]Model, error) {
	items, err := s.repo.List(ctx, userID, onlyUnread)
	if err != nil {
		return nil, apperr.InternalErr("failed to list notifications", err)
	}
	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) UnreadCount(ctx context.Context, userID common.UserID) (int64, error) {
	count, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return 0, apperr.InternalErr("failed to count unread notifications", err)
	}
	return count, nil
}

func (s *service) MarkRead(ctx context.Context, userID common.UserID, id common.NotificationID) error {
	ok, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to mark notification read", err)
	}
	if !ok {
		return apperr.NotFoundErr("notification not found")
	}
	return nil
}

func (s *service) Dismiss(ctx context.Context, userID common.UserID, id common.NotificationID) error {
	ok, err := s.repo.Dismiss(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to dismiss notification", err)
	}
	if !ok {
		return apperr.NotFoundErr("notification not found")
	}
	return nil
}

func (s *service) CreateIfNotExists(ctx context.Context, userID common.UserID, in CreateInput) (bool, error) {
	exists, err := s.repo.ExistsActive(ctx, userID, in.Type, in.RelatedEntityType, in.RelatedEntityID)
	if err != nil {
		return false, apperr.InternalErr("failed to check existing notification", err)
	}
	if exists {
		return false, nil
	}

	if _, err := s.Create(ctx, userID, in); err != nil {
		return false, err
	}
	return true, nil
}
