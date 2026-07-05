package bankaccount

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.BankAccountID, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	GetByID(ctx context.Context, userID common.UserID, id common.BankAccountID) (*Model, error)
	Update(ctx context.Context, userID common.UserID, id common.BankAccountID, set map[string]any) (bool, error)
	Delete(ctx context.Context, userID common.UserID, id common.BankAccountID) (bool, error)

	// BalanceTotal sums non-deleted account balances for a user — used by
	// the net worth calculation.
	BalanceTotal(ctx context.Context, userID common.UserID) (float64, error)

	// Adjust atomically applies delta (positive to add, negative to
	// withdraw) to an account's balance and returns the resulting balance.
	// Returns repository.ErrNotFound if the account doesn't exist, is
	// deleted, or — for a withdrawal — doesn't have enough balance to
	// cover it (checked atomically alongside the increment, not as a
	// separate read, to avoid a race between two concurrent withdrawals).
	Adjust(ctx context.Context, userID common.UserID, id common.BankAccountID, delta float64, note string) (float64, error)
}
