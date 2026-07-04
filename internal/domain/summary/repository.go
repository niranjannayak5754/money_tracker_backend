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

	// InvestmentTotals returns two values: total invested amount and total realized PnL
	InvestmentTotals(
		ctx context.Context,
		userID common.UserID,
		start, end time.Time,
	) (float64, float64, error)

	// BankBalanceTotal sums current non-deleted bank account balances.
	// Not date-scoped — a balance is a snapshot, not a period flow.
	BankBalanceTotal(ctx context.Context, userID common.UserID) (float64, error)

	// DebtOutstandingTotal sums current non-deleted debts' outstanding
	// balances. Not date-scoped, for the same reason as BankBalanceTotal.
	DebtOutstandingTotal(ctx context.Context, userID common.UserID) (float64, error)

	// InvestmentSnapshotAsOf sums current Amount/RealizedPnl for
	// investments whose own Date is before asOf — this correctly excludes
	// investments that didn't exist yet by that point, but (since there's
	// no dated withdrawal ledger) still uses each investment's CURRENT
	// running totals rather than what they were as of asOf.
	InvestmentSnapshotAsOf(ctx context.Context, userID common.UserID, asOf time.Time) (amount, realizedPnl float64, err error)

	// DebtSnapshotAsOf is the debt-side equivalent of
	// InvestmentSnapshotAsOf, keyed off each debt's StartDate.
	DebtSnapshotAsOf(ctx context.Context, userID common.UserID, asOf time.Time) (float64, error)
}
