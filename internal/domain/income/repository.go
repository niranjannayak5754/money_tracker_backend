package income

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
}
