package summary

import (
	"context"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

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
	svc := NewService(repo)

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
	svc := NewService(repo)

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
