package shared

import (
	"errors"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidMonth = errors.New("invalid month format, expected YYYY-MM")

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
		return time.Now()
	}
	return t
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
