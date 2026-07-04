package expense

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.ExpenseID, error)
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

	CountByCategory(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
	) (int64, error)

	ReassignCategory(
		ctx context.Context,
		userID common.UserID,
		fromCategoryID, toCategoryID common.CategoryID,
	) (int64, error)

	// SumByCategoryForMonth totals non-deleted expenses for one category
	// within [start, end) — used by the real-time budget-exceeded check.
	SumByCategoryForMonth(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
		start, end time.Time,
	) (float64, error)
}

// Cross-domain dependency (category)
type CategoryRepository interface {
	ExistsForUser(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
	) (bool, error)
}
