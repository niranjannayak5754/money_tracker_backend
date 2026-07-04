package budget

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	// Set upserts a row keyed on (userID, CategoryID, EffectiveFrom) —
	// calling it again for the same month updates that row's amount;
	// a different EffectiveFrom creates a new row, preserving history.
	Set(ctx context.Context, m Model) (common.BudgetID, error)

	// List returns every raw budget row for a user, across all categories
	// and effective dates — resolution (which row applies "as of" a given
	// month) happens in the service layer, not here.
	List(ctx context.Context, userID common.UserID) ([]Model, error)

	Delete(ctx context.Context, userID common.UserID, id common.BudgetID) (bool, error)
}
