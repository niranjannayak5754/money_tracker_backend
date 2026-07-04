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
}
