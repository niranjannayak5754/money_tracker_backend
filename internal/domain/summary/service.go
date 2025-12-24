package summary

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Service interface {
	Get(ctx context.Context, userID common.UserID, month string) (Result, error)
	Compare(ctx context.Context, userID common.UserID, months int) ([]MonthlyComparison, error)
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

	investTotal, err := s.repo.InvestmentTotal(ctx, userID, start, end)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate investment totals", err)
	}

	return Result{
		IncomeTotal:       incomeTotal,
		ExpenseTotal:      expenseTotal,
		Savings:           incomeTotal - expenseTotal - investTotal,
		CategoryBreakdown: breakdown,
	}, nil
}

func (s *service) Compare(
	ctx context.Context,
	userID common.UserID,
	months int,
) ([]MonthlyComparison, error) {
	if months > config.TWELVE {
		return nil, apperr.ValidationErr("months cannot be greater than 12")
	}

	now := time.Now().UTC()
	out := make([]MonthlyComparison, 0, months)

	for i := months - 1; i >= 0; i-- {
		t := now.AddDate(0, -i, 0)
		start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)

		incomeTotal, err := s.repo.IncomeTotal(ctx, userID, start, end)
		if err != nil {
			return nil, apperr.InternalErr("failed to calculate income total", err)
		}

		expenseTotal, _, err := s.repo.ExpenseTotals(ctx, userID, start, end)
		if err != nil {
			return nil, apperr.InternalErr("failed to calculate expense totals", err)
		}

		invTotal, err := s.repo.InvestmentTotal(ctx, userID, start, end)
		if err != nil {
			return nil, apperr.InternalErr("failed to calculate investment totals", err)
		}

		out = append(out, MonthlyComparison{
			Month:   start.Format(config.STANDARD_YEAR_MONTH),
			Income:  incomeTotal,
			Expense: expenseTotal,
			Savings: incomeTotal - expenseTotal - invTotal,
		})
	}

	return out, nil
}
