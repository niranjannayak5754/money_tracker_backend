package summary

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/budget"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Service interface {
	Get(ctx context.Context, userID common.UserID, month string) (Result, error)
	Compare(ctx context.Context, userID common.UserID, months int) ([]MonthlyComparison, error)
}

type service struct {
	repo    Repository
	budgets budget.Service
}

func NewService(repo Repository, budgets budget.Service) Service {
	return &service{repo: repo, budgets: budgets}
}

// buildBudgetStatus resolves the effective budget for month and joins it
// against the category spend breakdown — including budgeted categories
// with zero spend so far, not just ones that already have expenses.
func (s *service) buildBudgetStatus(
	ctx context.Context,
	userID common.UserID,
	month string,
	expenseTotal float64,
	breakdown []CategoryBreakdown,
) ([]BudgetStatus, *BudgetStatus, error) {
	resolved, err := s.budgets.ResolveForMonth(ctx, userID, month)
	if err != nil {
		return nil, nil, apperr.InternalErr("failed to resolve budgets", err)
	}

	spentByCategory := make(map[string]float64, len(breakdown))
	for _, cb := range breakdown {
		spentByCategory[cb.CategoryID] = cb.Total
	}

	var statuses []BudgetStatus
	for cid, budgeted := range resolved.ByCategory {
		spent := spentByCategory[string(cid)]
		var percent float64
		if budgeted > 0 {
			percent = spent / budgeted * 100
		}
		statuses = append(statuses, BudgetStatus{
			CategoryID:  string(cid),
			Budgeted:    budgeted,
			Spent:       spent,
			PercentUsed: percent,
			Exceeded:    spent > budgeted,
		})
	}

	var overall *BudgetStatus
	if resolved.Overall != nil {
		var percent float64
		if *resolved.Overall > 0 {
			percent = expenseTotal / *resolved.Overall * 100
		}
		overall = &BudgetStatus{
			Budgeted:    *resolved.Overall,
			Spent:       expenseTotal,
			PercentUsed: percent,
			Exceeded:    expenseTotal > *resolved.Overall,
		}
	}

	return statuses, overall, nil
}

func (s *service) Get(
	ctx context.Context,
	userID common.UserID,
	month string,
) (Result, error) {

	start, end, err := shared.MonthRange(month)
	if err != nil {
		return Result{}, err
	}

	expenseTotal, breakdown, err := s.repo.ExpenseTotals(ctx, userID, start, end)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate expense totals", err)
	}

	savingsBalance, err := s.repo.BankBalanceTotal(ctx, userID)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate savings account balance total", err)
	}

	var budgetStatus []BudgetStatus
	var overallBudget *BudgetStatus
	if month != "" {
		budgetStatus, overallBudget, err = s.buildBudgetStatus(ctx, userID, month, expenseTotal, breakdown)
		if err != nil {
			return Result{}, err
		}
	}

	return Result{
		ExpenseTotal:      expenseTotal,
		SavingsBalance:    savingsBalance,
		CategoryBreakdown: breakdown,
		BudgetStatus:      budgetStatus,
		OverallBudget:     overallBudget,
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

		expenseTotal, breakdown, err := s.repo.ExpenseTotals(ctx, userID, start, end)
		if err != nil {
			return nil, apperr.InternalErr("failed to calculate expense totals", err)
		}

		monthStr := start.Format(config.STANDARD_YEAR_MONTH)
		budgetStatus, overallBudget, err := s.buildBudgetStatus(ctx, userID, monthStr, expenseTotal, breakdown)
		if err != nil {
			return nil, err
		}

		out = append(out, MonthlyComparison{
			Month:         monthStr,
			Expense:       expenseTotal,
			BudgetStatus:  budgetStatus,
			OverallBudget: overallBudget,
		})
	}

	return out, nil
}
