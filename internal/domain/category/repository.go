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
