package notification

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.NotificationID, error)
	List(ctx context.Context, userID common.UserID, onlyUnread bool) ([]Model, error)
	UnreadCount(ctx context.Context, userID common.UserID) (int64, error)
	MarkRead(ctx context.Context, userID common.UserID, id common.NotificationID) (bool, error)
	Dismiss(ctx context.Context, userID common.UserID, id common.NotificationID) (bool, error)

	// ExistsActive reports whether a non-dismissed notification already
	// exists for this exact user+type+related-entity combination.
	ExistsActive(ctx context.Context, userID common.UserID, notifType Type, relatedEntityType, relatedEntityID string) (bool, error)
}
