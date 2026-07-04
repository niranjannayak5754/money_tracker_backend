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

	incomeTotal, err := s.repo.IncomeTotal(ctx, userID, start, end)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate income total", err)
	}

	expenseTotal, breakdown, err := s.repo.ExpenseTotals(ctx, userID, start, end)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate expense totals", err)
	}

	investTotal, _, err := s.repo.InvestmentTotals(ctx, userID, start, end)
	if err != nil {
		return Result{}, apperr.InternalErr("failed to calculate investment totals", err)
	}

	netWorth, err := s.currentNetWorth(ctx, userID)
	if err != nil {
		return Result{}, err
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
		IncomeTotal:       incomeTotal,
		ExpenseTotal:      expenseTotal,
		InvestmentTotal:   investTotal,
		Savings:           incomeTotal - expenseTotal - investTotal,
		NetWorth:          netWorth,
		CategoryBreakdown: breakdown,
		BudgetStatus:      budgetStatus,
		OverallBudget:     overallBudget,
	}, nil
}

// currentNetWorth computes a balance-sheet snapshot as of now — bank
// balances and investments' remaining value are current holdings, not a
// period flow, so this deliberately ignores any month filter.
//
// Known v1 limitation: bank balances and debt have no historical ledger, so
// this always reflects the current snapshot; a true point-in-time
// reconstruction for past months is Phase 7's net-worth-history endpoint.
func (s *service) currentNetWorth(ctx context.Context, userID common.UserID) (float64, error) {
	bankTotal, err := s.repo.BankBalanceTotal(ctx, userID)
	if err != nil {
		return 0, apperr.InternalErr("failed to calculate bank balance total", err)
	}

	investAmount, realizedPnl, err := s.repo.InvestmentTotals(ctx, userID, time.Time{}, time.Time{})
	if err != nil {
		return 0, apperr.InternalErr("failed to calculate lifetime investment totals", err)
	}

	debtTotal, err := s.repo.DebtOutstandingTotal(ctx, userID)
	if err != nil {
		return 0, apperr.InternalErr("failed to calculate debt outstanding total", err)
	}

	return bankTotal + investAmount + realizedPnl - debtTotal, nil
}

func (s *service) Compare(
	ctx context.Context,
	userID common.UserID,
	months int,
) ([]MonthlyComparison, error) {
	if months > config.TWELVE {
		return nil, apperr.ValidationErr("months cannot be greater than 12")
	}

	// NetWorth is a current snapshot (see currentNetWorth) — computed once
	// and reused across every historical month entry, since bank/debt
	// balances aren't ledgered historically in v1.
	netWorth, err := s.currentNetWorth(ctx, userID)
	if err != nil {
		return nil, err
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

		expenseTotal, breakdown, err := s.repo.ExpenseTotals(ctx, userID, start, end)
		if err != nil {
			return nil, apperr.InternalErr("failed to calculate expense totals", err)
		}

		investTotal, _, err := s.repo.InvestmentTotals(ctx, userID, start, end)
		if err != nil {
			return nil, apperr.InternalErr("failed to calculate investment totals", err)
		}

		monthStr := start.Format(config.STANDARD_YEAR_MONTH)
		budgetStatus, overallBudget, err := s.buildBudgetStatus(ctx, userID, monthStr, expenseTotal, breakdown)
		if err != nil {
			return nil, err
		}

		out = append(out, MonthlyComparison{
			Month:         monthStr,
			Income:        incomeTotal,
			Expense:       expenseTotal,
			Investment:    investTotal,
			Savings:       incomeTotal - expenseTotal - investTotal,
			NetWorth:      netWorth,
			BudgetStatus:  budgetStatus,
			OverallBudget: overallBudget,
		})
	}

	return out, nil
}
