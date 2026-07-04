package income

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.IncomeID, error)
	ListByMonth(
		ctx context.Context,
		userID common.UserID,
		start, end time.Time,
	) ([]Model, error)
	Update(
		ctx context.Context,
		userID common.UserID,
		id common.IncomeID,
		set map[string]any,
	) (bool, error)
	Delete(
		ctx context.Context,
		userID common.UserID,
		id common.IncomeID,
	) (bool, error)

	// ReassignCategory mirrors expense's — used for the opt-in
	// "reassign to another category" step of archiving.
	ReassignCategory(
		ctx context.Context,
		userID common.UserID,
		fromCategoryID, toCategoryID common.CategoryID,
	) (int64, error)
}

// CategoryRepository is a narrow cross-domain dependency used to validate
// that an income's category exists, isn't archived, and is income-type.
type CategoryRepository interface {
	ExistsForUserWithType(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
		categoryType string,
	) (bool, error)
}
