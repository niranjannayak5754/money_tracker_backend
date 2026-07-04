package shared

import (
	"math"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// appLocation is the timezone used to interpret date/time inputs that don't
// carry their own UTC offset (e.g. "2025-11-10"). Without this, such inputs
// default to UTC, which can bucket a transaction into the wrong day/month
// for a non-UTC user. Set once at startup via SetAppTimezone.
var appLocation = time.UTC

// SetAppTimezone sets the timezone used for offset-less date/time parsing.
// Call once at startup, before serving requests.
func SetAppTimezone(loc *time.Location) {
	if loc != nil {
		appLocation = loc
	}
}

type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	str := string(b)

	// null or empty
	if str == "null" || str == `""` {
		ft.Time = time.Time{}
		return nil
	}

	// remove surrounding quotes: "2025-11-10" -> 2025-11-10
	s := str[1 : len(str)-1]

	// Correct layouts
	layouts := []string{
		time.RFC3339,          // 2025-11-10T10:30:00Z
		"2006-01-02",          // 2025-11-10
		"2006/01/02",          // 2025/11/10
		"2006-01-02 15:04:05", // 2025-11-10 10:30:00
		"2006-01-02T15:04:05", // 2025-11-10T10:30:00
		"2006-01-02T15:04",    // 2025-11-10T10:30
		"2006-01-02 15:04",    // 2025-11-10 10:30
	}

	// ParseInLocation only matters for layouts with no zone in the string
	// (RFC3339's embedded offset always wins regardless of the location arg).
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, appLocation); err == nil {
			ft.Time = t
			return nil
		}
	}

	return apperr.BadRequestErr("invalid date format")
}

// MonthBounds returns the start and end timestamps for the given month
// in UTC. Month must be in "YYYY-MM" format.
func MonthBounds(month string) (time.Time, time.Time, error) {
	if len(month) != config.SEVEN {
		return time.Time{}, time.Time{}, apperr.BadRequestErr("invalid month format, expected YYYY-MM")
	}

	t, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, apperr.BadRequestErr("invalid month format, expected YYYY-MM")
	}

	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	return start, end, nil
}

// MonthRange is a helper for domain services.
// If month is empty, it returns zero start/end to indicate a lifetime (no date filters).
// If month is invalid, it returns an error rather than silently falling back
// to the current month.
func MonthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		return time.Time{}, time.Time{}, nil
	}

	return MonthBounds(month)
}

// ChooseDate normalizes zero time values to now.
func ChooseDate(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

// Decimal128ToFloat converts a MongoDB Decimal128 value into a float64.
func Decimal128ToFloat(d primitive.Decimal128) float64 {
	bi, exp, err := d.BigInt()
	if err != nil {
		return 0
	}
	coef, _ := bi.Float64()
	return coef * math.Pow10(int(exp))
}
