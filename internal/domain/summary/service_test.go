package summary

import (
	"context"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/budget"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type fakeBudgetService struct {
	resolved budget.Resolved
}

func (f *fakeBudgetService) Set(ctx context.Context, userID common.UserID, in budget.SetInput) (budget.Model, error) {
	return budget.Model{}, nil
}
func (f *fakeBudgetService) List(ctx context.Context, userID common.UserID) ([]budget.Model, error) {
	return nil, nil
}
func (f *fakeBudgetService) Delete(ctx context.Context, userID common.UserID, id common.BudgetID) error {
	return nil
}
func (f *fakeBudgetService) ResolveForMonth(ctx context.Context, userID common.UserID, month string) (budget.Resolved, error) {
	return f.resolved, nil
}

func emptyBudgets() *fakeBudgetService {
	return &fakeBudgetService{resolved: budget.Resolved{ByCategory: map[common.CategoryID]float64{}}}
}

type fakeSummaryRepo struct {
	expenseTotal float64
	breakdown    []CategoryBreakdown
	bankTotal    float64
}

func (f *fakeSummaryRepo) ExpenseTotals(ctx context.Context, userID common.UserID, start, end time.Time) (float64, []CategoryBreakdown, error) {
	return f.expenseTotal, f.breakdown, nil
}

func (f *fakeSummaryRepo) BankBalanceTotal(ctx context.Context, userID common.UserID) (float64, error) {
	return f.bankTotal, nil
}

func TestGet_ReturnsExpenseTotalAndSavingsBalance(t *testing.T) {
	repo := &fakeSummaryRepo{
		expenseTotal: 2000,
		bankTotal:    10000,
	}
	svc := NewService(repo, emptyBudgets())

	result, err := svc.Get(context.Background(), common.UserID("user-1"), "2026-07")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if result.ExpenseTotal != 2000 {
		t.Fatalf("expected expense total 2000, got %v", result.ExpenseTotal)
	}
	if result.SavingsBalance != 10000 {
		t.Fatalf("expected savings balance 10000, got %v", result.SavingsBalance)
	}
}

func TestCompare_ReturnsExpensePerMonth(t *testing.T) {
	repo := &fakeSummaryRepo{
		expenseTotal: 1000,
		bankTotal:    5000,
	}
	svc := NewService(repo, emptyBudgets())

	results, err := svc.Compare(context.Background(), common.UserID("user-1"), 3)
	if err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 months, got %d", len(results))
	}

	for _, r := range results {
		if r.Expense != 1000 {
			t.Fatalf("expected expense 1000 for month %s, got %v", r.Month, r.Expense)
		}
	}
}

func TestGet_BudgetStatusIncludesZeroSpendBudgetedCategories(t *testing.T) {
	repo := &fakeSummaryRepo{
		expenseTotal: 800,
		breakdown: []CategoryBreakdown{
			{CategoryID: "cat-1", CategoryName: "Food", Total: 800},
		},
	}
	overall := 5000.0
	budgets := &fakeBudgetService{resolved: budget.Resolved{
		Overall: &overall,
		ByCategory: map[common.CategoryID]float64{
			"cat-1": 1000, // has spend
			"cat-2": 500,  // budgeted but zero spend this month
		},
	}}
	svc := NewService(repo, budgets)

	result, err := svc.Get(context.Background(), common.UserID("user-1"), "2026-07")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if len(result.BudgetStatus) != 2 {
		t.Fatalf("expected 2 budget statuses (including zero-spend category), got %d: %+v", len(result.BudgetStatus), result.BudgetStatus)
	}

	byID := map[string]BudgetStatus{}
	for _, bs := range result.BudgetStatus {
		byID[bs.CategoryID] = bs
	}

	cat1 := byID["cat-1"]
	if cat1.Spent != 800 || cat1.Budgeted != 1000 || cat1.Exceeded || cat1.PercentUsed != 80 {
		t.Fatalf("unexpected cat-1 status: %+v", cat1)
	}

	cat2 := byID["cat-2"]
	if cat2.Spent != 0 || cat2.Budgeted != 500 || cat2.Exceeded || cat2.PercentUsed != 0 {
		t.Fatalf("expected cat-2 (budgeted, zero spend) to show Spent=0, PercentUsed=0, got %+v", cat2)
	}

	if result.OverallBudget == nil {
		t.Fatalf("expected overall budget to be present")
	}
	if result.OverallBudget.Budgeted != 5000 || result.OverallBudget.Spent != 800 || result.OverallBudget.Exceeded {
		t.Fatalf("unexpected overall budget status: %+v", result.OverallBudget)
	}
}

func TestGet_NoBudgetSet_OmitsBudgetStatus(t *testing.T) {
	repo := &fakeSummaryRepo{
		expenseTotal: 800,
		breakdown:    []CategoryBreakdown{{CategoryID: "cat-1", CategoryName: "Food", Total: 800}},
	}
	svc := NewService(repo, emptyBudgets())

	result, err := svc.Get(context.Background(), common.UserID("user-1"), "2026-07")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if len(result.BudgetStatus) != 0 {
		t.Fatalf("expected no budget status when nothing is budgeted, got %+v", result.BudgetStatus)
	}
	if result.OverallBudget != nil {
		t.Fatalf("expected no overall budget when none is set, got %+v", result.OverallBudget)
	}
}

func TestGet_ExceededBudgetFlagged(t *testing.T) {
	repo := &fakeSummaryRepo{
		expenseTotal: 1200,
		breakdown:    []CategoryBreakdown{{CategoryID: "cat-1", CategoryName: "Food", Total: 1200}},
	}
	budgets := &fakeBudgetService{resolved: budget.Resolved{
		ByCategory: map[common.CategoryID]float64{"cat-1": 1000},
	}}
	svc := NewService(repo, budgets)

	result, err := svc.Get(context.Background(), common.UserID("user-1"), "2026-07")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if len(result.BudgetStatus) != 1 || !result.BudgetStatus[0].Exceeded {
		t.Fatalf("expected cat-1 to be flagged as exceeded, got %+v", result.BudgetStatus)
	}
}
