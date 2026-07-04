package category

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.CategoryID, error)

	Update(
		ctx context.Context,
		userID common.UserID,
		id common.CategoryID,
		set map[string]any,
	) (bool, error)

	List(
		ctx context.Context,
		userID common.UserID,
		archived *bool,
	) ([]Model, error)

	ExistsForUser(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
	) (bool, error)
}
