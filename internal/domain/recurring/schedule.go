package recurring

import "time"

// nextRunDate computes the occurrence after `from`, preserving the original
// target day (and, for yearly, month) rather than drifting when a
// clamped short month becomes the base for the next computation — e.g. a
// template targeting the 31st goes Jan 31 -> Feb 28 -> Mar 31, not
// Jan 31 -> Feb 28 -> Mar 28 (which naive from.AddDate(0,1,0) chaining
// would produce since it repeatedly derives the next target from an
// already-clamped day).
func nextRunDate(from time.Time, freq Frequency, dayOfMonth int, anchorMonth time.Month) time.Time {
	switch freq {
	case FrequencyYearly:
		return clampToMonth(from.Year()+1, anchorMonth, dayOfMonth, from)
	default: // monthly
		totalMonths := int(from.Month()) + 1
		year := from.Year() + (totalMonths-1)/12
		month := time.Month((totalMonths-1)%12 + 1)
		return clampToMonth(year, month, dayOfMonth, from)
	}
}

// clampToMonth builds a date in the given year/month, clamping day to that
// month's actual last day (e.g. day=31 in February becomes 28 or 29).
func clampToMonth(year int, month time.Month, day int, ref time.Time) time.Time {
	firstOfMonth := time.Date(year, month, 1, ref.Hour(), ref.Minute(), ref.Second(), 0, ref.Location())
	lastDay := firstOfMonth.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	if day < 1 {
		day = 1
	}
	return time.Date(year, month, day, ref.Hour(), ref.Minute(), ref.Second(), 0, ref.Location())
}
