package goal

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/bankaccount"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.GoalID, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	Update(ctx context.Context, userID common.UserID, id common.GoalID, set map[string]any) (bool, error)
	Delete(ctx context.Context, userID common.UserID, id common.GoalID) (bool, error)
}

// BankAccountRepository is a narrow cross-domain dependency used only to
// resolve a goal's progress from its linked bank account's balance. The
// real bankaccount.Repository satisfies this structurally.
type BankAccountRepository interface {
	GetByID(ctx context.Context, userID common.UserID, id common.BankAccountID) (*bankaccount.Model, error)
}

// InvestmentRepository is the investment-side equivalent of
// BankAccountRepository.
type InvestmentRepository interface {
	GetByID(ctx context.Context, userID common.UserID, id common.InvestmentID) (*investment.Model, error)
}
