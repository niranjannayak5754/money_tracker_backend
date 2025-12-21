package summary

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Service interface {
	Get(ctx context.Context, userID common.UserID, month string) (Result, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Get(
	ctx context.Context,
	userID common.UserID,
	month string,
) (Result, error) {

	start, end := shared.MonthRange(month)

	incomeTotal, err := s.repo.IncomeTotal(ctx, userID, start, end)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate income total", err)
	}

	expenseTotal, breakdown, err := s.repo.ExpenseTotals(ctx, userID, start, end)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate expense totals", err)
	}

	return Result{
		IncomeTotal:       incomeTotal,
		ExpenseTotal:      expenseTotal,
		Savings:           incomeTotal - expenseTotal,
		CategoryBreakdown: breakdown,
	}, nil
}
