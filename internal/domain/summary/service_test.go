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

// fakeSummaryRepo lets InvestmentTotals answer differently for a
// month-scoped query vs the lifetime (zero start/end) query net worth
// relies on, so the two use cases can be told apart in assertions.
type fakeSummaryRepo struct {
	incomeTotal  float64
	expenseTotal float64
	breakdown    []CategoryBreakdown

	monthlyInvestTotal  float64
	monthlyRealizedPnl  float64
	lifetimeInvestTotal float64
	lifetimeRealizedPnl float64

	bankTotal float64
	debtTotal float64

	// optional overrides for the *AsOf methods; nil falls back to the
	// lifetime totals / debtTotal above, so existing tests that don't
	// care about dated snapshots don't need to set these.
	investmentSnapshotFn func(asOf time.Time) (amount, realizedPnl float64)
	debtSnapshotFn       func(asOf time.Time) float64
}

func (f *fakeSummaryRepo) IncomeTotal(ctx context.Context, userID common.UserID, start, end time.Time) (float64, error) {
	return f.incomeTotal, nil
}

func (f *fakeSummaryRepo) ExpenseTotals(ctx context.Context, userID common.UserID, start, end time.Time) (float64, []CategoryBreakdown, error) {
	return f.expenseTotal, f.breakdown, nil
}

func (f *fakeSummaryRepo) InvestmentTotals(ctx context.Context, userID common.UserID, start, end time.Time) (float64, float64, error) {
	if start.IsZero() && end.IsZero() {
		return f.lifetimeInvestTotal, f.lifetimeRealizedPnl, nil
	}
	return f.monthlyInvestTotal, f.monthlyRealizedPnl, nil
}

func (f *fakeSummaryRepo) BankBalanceTotal(ctx context.Context, userID common.UserID) (float64, error) {
	return f.bankTotal, nil
}

func (f *fakeSummaryRepo) DebtOutstandingTotal(ctx context.Context, userID common.UserID) (float64, error) {
	return f.debtTotal, nil
}

func (f *fakeSummaryRepo) InvestmentSnapshotAsOf(ctx context.Context, userID common.UserID, asOf time.Time) (float64, float64, error) {
	if f.investmentSnapshotFn != nil {
		amount, pnl := f.investmentSnapshotFn(asOf)
		return amount, pnl, nil
	}
	return f.lifetimeInvestTotal, f.lifetimeRealizedPnl, nil
}

func (f *fakeSummaryRepo) DebtSnapshotAsOf(ctx context.Context, userID common.UserID, asOf time.Time) (float64, error) {
	if f.debtSnapshotFn != nil {
		return f.debtSnapshotFn(asOf), nil
	}
	return f.debtTotal, nil
}

func TestGet_NetWorthUsesLifetimeTotalsNotMonthScoped(t *testing.T) {
	repo := &fakeSummaryRepo{
		incomeTotal:         5000,
		expenseTotal:        2000,
		monthlyInvestTotal:  1000,
		monthlyRealizedPnl:  50,
		lifetimeInvestTotal: 40000,
		lifetimeRealizedPnl: 3000,
		bankTotal:           10000,
		debtTotal:           8000,
	}
	svc := NewService(repo, emptyBudgets())

	result, err := svc.Get(context.Background(), common.UserID("user-1"), "2026-07")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	// Savings/InvestmentTotal use the month-scoped figures.
	if result.InvestmentTotal != 1000 {
		t.Fatalf("expected month-scoped investment total 1000, got %v", result.InvestmentTotal)
	}
	wantSavings := 5000.0 - 2000.0 - 1000.0
	if result.Savings != wantSavings {
		t.Fatalf("expected savings %v, got %v", wantSavings, result.Savings)
	}

	// NetWorth must use lifetime investment totals + bank - debt, not the
	// month-scoped investment figures.
	wantNetWorth := 10000.0 + 40000.0 + 3000.0 - 8000.0
	if result.NetWorth != wantNetWorth {
		t.Fatalf("expected net worth %v, got %v", wantNetWorth, result.NetWorth)
	}
}

func TestCompare_NetWorthConstantAcrossMonths(t *testing.T) {
	repo := &fakeSummaryRepo{
		incomeTotal:         3000,
		expenseTotal:        1000,
		monthlyInvestTotal:  500,
		lifetimeInvestTotal: 20000,
		lifetimeRealizedPnl: 1500,
		bankTotal:           5000,
		debtTotal:           2000,
	}
	svc := NewService(repo, emptyBudgets())

	results, err := svc.Compare(context.Background(), common.UserID("user-1"), 3)
	if err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 months, got %d", len(results))
	}

	wantNetWorth := 5000.0 + 20000.0 + 1500.0 - 2000.0
	for _, r := range results {
		if r.NetWorth != wantNetWorth {
			t.Fatalf("expected net worth %v for month %s, got %v", wantNetWorth, r.Month, r.NetWorth)
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

func TestNetWorthHistory_ExcludesInvestmentsNotYetCreated(t *testing.T) {
	// Simulate a single investment created on 2026-05-15 with amount 8000.
	// Months before May should see 0 investment value; May onward should
	// see 8000 — this is the month-exclusion behavior NetWorthHistory adds
	// on top of Compare's "always current totals" NetWorth.
	investedAt := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)

	repo := &fakeSummaryRepo{
		bankTotal: 1000,
		investmentSnapshotFn: func(asOf time.Time) (float64, float64) {
			if asOf.After(investedAt) {
				return 8000, 0
			}
			return 0, 0
		},
	}
	svc := NewService(repo, emptyBudgets())

	// Compute relative to a fixed "now" isn't possible since the service
	// uses time.Now() internally; instead assert on the shape: exactly one
	// of the returned months transitions from 0 to 8000, and once it does,
	// later months stay at 8000 (monotonic, no drop back to 0).
	points, err := svc.NetWorthHistory(context.Background(), common.UserID("user-1"), 12)
	if err != nil {
		t.Fatalf("net worth history failed: %v", err)
	}
	if len(points) != 12 {
		t.Fatalf("expected 12 points, got %d", len(points))
	}

	seenInvested := false
	for _, p := range points {
		if p.Investments == 8000 {
			seenInvested = true
		}
		if seenInvested && p.Investments != 8000 {
			t.Fatalf("expected investment total to stay at 8000 once the investment exists, got %v in %+v", p.Investments, points)
		}
		if p.Cash != 1000 {
			t.Fatalf("expected cash to always reflect the current balance (v1 limitation), got %v", p.Cash)
		}
	}
	if !seenInvested {
		t.Fatalf("expected at least one month to include the investment, got %+v", points)
	}
}

func TestNetWorthHistory_ComposesAllComponents(t *testing.T) {
	repo := &fakeSummaryRepo{
		bankTotal: 5000,
		investmentSnapshotFn: func(asOf time.Time) (float64, float64) {
			return 10000, 500
		},
		debtSnapshotFn: func(asOf time.Time) float64 {
			return 3000
		},
	}
	svc := NewService(repo, emptyBudgets())

	points, err := svc.NetWorthHistory(context.Background(), common.UserID("user-1"), 1)
	if err != nil {
		t.Fatalf("net worth history failed: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}

	want := 5000.0 + 10000.0 + 500.0 - 3000.0
	if points[0].NetWorth != want {
		t.Fatalf("expected net worth %v, got %v (%+v)", want, points[0].NetWorth, points[0])
	}
}
