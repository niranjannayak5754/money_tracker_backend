package shared

import (
	"errors"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidMonth = errors.New("invalid month format, expected YYYY-MM")

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

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			ft.Time = t
			return nil
		}
	}

	return errors.New("invalid date format")
}

// MonthBounds returns the start and end timestamps for the given month
// in UTC. Month must be in "YYYY-MM" format.
func MonthBounds(month string) (time.Time, time.Time, error) {
	if len(month) != 7 {
		return time.Time{}, time.Time{}, ErrInvalidMonth
	}

	t, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidMonth
	}

	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	return start, end, nil
}

// MonthRange is a helper for domain services.
// If month is empty or invalid, it returns the *current month's* start/end automatically.
func MonthRange(month string) (time.Time, time.Time) {
	if month == "" {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	}

	start, end, err := MonthBounds(month)
	if err != nil {
		now := time.Now().UTC()
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, 0)
	}

	return start, end
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
