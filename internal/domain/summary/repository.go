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
}
