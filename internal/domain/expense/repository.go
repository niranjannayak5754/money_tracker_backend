package expense

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) error
	ListByMonth(
		ctx context.Context,
		userID common.UserID,
		start, end time.Time,
		categoryID *common.CategoryID,
	) ([]Model, error)

	Update(
		ctx context.Context,
		userID common.UserID,
		id common.ExpenseID,
		set map[string]any,
	) (bool, error)

	Delete(
		ctx context.Context,
		userID common.UserID,
		id common.ExpenseID,
	) (bool, error)
}

// Cross-domain dependency (category)
type CategoryRepository interface {
	ExistsForUser(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
	) (bool, error)
}
