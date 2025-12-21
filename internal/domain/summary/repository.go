package summary

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	IncomeTotal(
		ctx context.Context,
		userID common.UserID,
		start, end time.Time,
	) (float64, error)

	ExpenseTotals(
		ctx context.Context,
		userID common.UserID,
		start, end time.Time,
	) (float64, []CategoryBreakdown, error)
}
