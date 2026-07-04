package shared

import (
	"math"
	"testing"
	"time"
)

func TestXIRR_KnownAnswerSimpleTwoCashflowCase(t *testing.T) {
	// Invest 1000, get back 1100 exactly one year later -> 10% annualized.
	flows := []CashFlow{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Amount: -1000},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Amount: 1100},
	}

	rate, err := XIRR(flows)
	if err != nil {
		t.Fatalf("xirr failed: %v", err)
	}

	// ~365 days / 365 = ~1.0 year, so rate should be very close to 0.10.
	if math.Abs(rate-0.10) > 0.001 {
		t.Fatalf("expected rate close to 0.10, got %v", rate)
	}
}

func TestXIRR_KnownAnswerHalfYearDoubleRate(t *testing.T) {
	// Invest 1000, get back 1050 exactly half a year later -> ~10.25% (1.05^2 - 1).
	flows := []CashFlow{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Amount: -1000},
		{Date: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC), Amount: 1050},
	}

	rate, err := XIRR(flows)
	if err != nil {
		t.Fatalf("xirr failed: %v", err)
	}

	want := math.Pow(1.05, 2) - 1
	if math.Abs(rate-want) > 0.005 {
		t.Fatalf("expected rate close to %v, got %v", want, rate)
	}
}

func TestXIRR_IrregularCashFlowsConverges(t *testing.T) {
	// SIP-like irregular contributions plus a final redemption.
	flows := []CashFlow{
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), Amount: -5000},
		{Date: time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC), Amount: -5000},
		{Date: time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC), Amount: -3000},
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Amount: 16000},
	}

	rate, err := XIRR(flows)
	if err != nil {
		t.Fatalf("expected irregular cash flows to converge, got error: %v", err)
	}
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		t.Fatalf("expected a finite rate, got %v", rate)
	}
}

func TestXIRR_AllPositiveCashFlows_ReturnsErrorNotHang(t *testing.T) {
	flows := []CashFlow{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Amount: 1000},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Amount: 500},
	}

	// The bounded iteration count guarantees this returns rather than
	// hanging — the assertion here is just that it terminates with an
	// error rather than a nonsensical answer.
	_, err := XIRR(flows)
	if err == nil {
		t.Fatalf("expected an error for all-positive cash flows (no real solution)")
	}
}

func TestXIRR_AllNegativeCashFlows_ReturnsErrorNotHang(t *testing.T) {
	flows := []CashFlow{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Amount: -1000},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Amount: -500},
	}

	_, err := XIRR(flows)
	if err == nil {
		t.Fatalf("expected an error for all-negative cash flows (no real solution)")
	}
}

func TestXIRR_ZeroTimeSpan_ReturnsError(t *testing.T) {
	sameDay := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	flows := []CashFlow{
		{Date: sameDay, Amount: -1000},
		{Date: sameDay, Amount: 1100},
	}

	_, err := XIRR(flows)
	if err == nil {
		t.Fatalf("expected an error when all cash flows share the same date")
	}
}

func TestXIRR_FewerThanTwoCashFlows_ReturnsError(t *testing.T) {
	_, err := XIRR([]CashFlow{{Date: time.Now(), Amount: -1000}})
	if err == nil {
		t.Fatalf("expected an error with fewer than 2 cash flows")
	}
	_, err = XIRR(nil)
	if err == nil {
		t.Fatalf("expected an error with no cash flows")
	}
}
