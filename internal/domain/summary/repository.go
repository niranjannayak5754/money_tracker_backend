package summary

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	ExpenseTotals(
		ctx context.Context,
		userID common.UserID,
		start, end time.Time,
	) (float64, []CategoryBreakdown, error)

	// BankBalanceTotal sums current non-deleted bank account (savings
	// account) balances. Not date-scoped — a balance is a snapshot, not a
	// period flow.
	BankBalanceTotal(ctx context.Context, userID common.UserID) (float64, error)
}
