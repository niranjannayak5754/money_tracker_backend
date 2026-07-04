package shared

import (
	"testing"
	"time"
)

func TestMonthRange(t *testing.T) {
	tests := []struct {
		name      string
		month     string
		wantErr   bool
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:  "empty month returns lifetime zero range",
			month: "",
		},
		{
			name:      "valid month returns UTC bounds",
			month:     "2026-07",
			wantStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "garbage month returns error, not current-month fallback",
			month:   "garbage",
			wantErr: true,
		},
		{
			name:    "out-of-range month number returns error",
			month:   "2026-13",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := MonthRange(tt.month)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for month %q, got nil (start=%v end=%v)", tt.month, start, end)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for month %q: %v", tt.month, err)
			}
			if !start.Equal(tt.wantStart) || !end.Equal(tt.wantEnd) {
				t.Fatalf("month %q: got start=%v end=%v, want start=%v end=%v", tt.month, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
