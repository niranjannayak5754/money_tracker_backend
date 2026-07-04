package recurring

import (
	"testing"
	"time"
)

func TestNextRunDate_MonthlyClampsAtShortMonthWithoutDrifting(t *testing.T) {
	// A template targeting the 31st: Jan 31 -> Feb 28 (non-leap) -> Mar 31.
	// A naive from.AddDate(0,1,0) chain would drift to Mar 28 instead,
	// since it would derive March from February's already-clamped day.
	jan31 := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)

	feb := nextRunDate(jan31, FrequencyMonthly, 31, time.January)
	wantFeb := time.Date(2026, 2, 28, 9, 0, 0, 0, time.UTC)
	if !feb.Equal(wantFeb) {
		t.Fatalf("expected Feb clamp to %v, got %v", wantFeb, feb)
	}

	mar := nextRunDate(feb, FrequencyMonthly, 31, time.January)
	wantMar := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	if !mar.Equal(wantMar) {
		t.Fatalf("expected Mar to bounce back to 31st (%v), got %v — target day must not drift from a clamped intermediate", wantMar, mar)
	}
}

func TestNextRunDate_MonthlyClampsOnLeapYearFebruary(t *testing.T) {
	jan31 := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC) // 2024 is a leap year
	feb := nextRunDate(jan31, FrequencyMonthly, 31, time.January)
	want := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	if !feb.Equal(want) {
		t.Fatalf("expected leap-year Feb clamp to %v, got %v", want, feb)
	}
}

func TestNextRunDate_MonthlyMidMonthNoClampingNeeded(t *testing.T) {
	from := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	next := nextRunDate(from, FrequencyMonthly, 15, time.January)
	want := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("expected %v, got %v", want, next)
	}
}

func TestNextRunDate_MonthlyWrapsYearBoundary(t *testing.T) {
	from := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	next := nextRunDate(from, FrequencyMonthly, 10, time.January)
	want := time.Date(2027, 1, 10, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("expected %v, got %v", want, next)
	}
}

func TestNextRunDate_YearlyClampsLeapDayInNonLeapYear(t *testing.T) {
	from := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC) // leap year
	next := nextRunDate(from, FrequencyYearly, 29, time.February)
	want := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC) // 2025 is not a leap year
	if !next.Equal(want) {
		t.Fatalf("expected %v, got %v", want, next)
	}
}

func TestNextRunDate_YearlyRegular(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	next := nextRunDate(from, FrequencyYearly, 1, time.June)
	want := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("expected %v, got %v", want, next)
	}
}
