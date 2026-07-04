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

	GetByID(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
	) (Model, error)

	ExistsForUser(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
	) (bool, error)

	// ExistsForUserWithType is like ExistsForUser but also requires the
	// category's Type to match — used so an expense/income/budget can't
	// silently attach to a category of the wrong kind (e.g. an expense
	// pointing at an income-only category).
	ExistsForUserWithType(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
		categoryType string,
	) (bool, error)
}

// ExpenseRepository is a narrow cross-domain dependency used only to keep
// category archiving safe (block/redirect archiving a category that still
// has expenses pointing at it).
type ExpenseRepository interface {
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
}

// IncomeRepository mirrors ExpenseRepository — a narrow cross-domain
// dependency so archiving an income-type category can block/redirect
// income entries that still reference it, the same way expense-type
// categories are guarded against expenses.
type IncomeRepository interface {
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
}
