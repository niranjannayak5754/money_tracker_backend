package investment

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.InvestmentID, error)
	ListByMonth(ctx context.Context, userID common.UserID, start, end time.Time) ([]Model, error)
	GetByID(ctx context.Context, userID common.UserID, id common.InvestmentID) (*Model, error)
	Update(ctx context.Context, userID common.UserID, id common.InvestmentID, set map[string]any) (bool, error)
	Delete(ctx context.Context, userID common.UserID, id common.InvestmentID) (bool, error)
	GetTypes(ctx context.Context) ([]TypeDoc, error)

	// ListMaturingBefore returns non-closed, non-deleted investments
	// (across all users) with a MaturityDate before the given time — used
	// by the background notification scanner.
	ListMaturingBefore(ctx context.Context, before time.Time) ([]Model, error)
}
