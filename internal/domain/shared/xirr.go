package shared

import (
	"errors"
	"math"
	"sort"
	"time"
)

// CashFlow is one dated cash movement for an XIRR calculation. Amount is
// negative for money going out (invested) and positive for money coming in
// (retrieved, or current value treated as a terminal "as-of-today" inflow).
type CashFlow struct {
	Date   time.Time
	Amount float64
}

const (
	xirrMaxIterations = 100
	xirrTolerance     = 1e-7
	xirrInitialGuess  = 0.1
)

// XIRR computes the annualized internal rate of return for a series of
// irregular cash flows via Newton-Raphson. The iteration count is bounded,
// so this can never hang — it returns an error instead of converging on
// pathological inputs (e.g. all-positive or all-negative cash flows, which
// have no real solution).
func XIRR(flows []CashFlow) (float64, error) {
	if len(flows) < 2 {
		return 0, errors.New("at least two cash flows are required")
	}

	sorted := make([]CashFlow, len(flows))
	copy(sorted, flows)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })

	base := sorted[0].Date
	if sorted[len(sorted)-1].Date.Equal(base) {
		return 0, errors.New("cash flows span zero time, cannot compute an annualized rate")
	}

	years := make([]float64, len(sorted))
	for i, cf := range sorted {
		years[i] = cf.Date.Sub(base).Hours() / 24 / 365
	}

	npv := func(rate float64) float64 {
		total := 0.0
		for i, cf := range sorted {
			total += cf.Amount / math.Pow(1+rate, years[i])
		}
		return total
	}

	dnpv := func(rate float64) float64 {
		total := 0.0
		for i, cf := range sorted {
			if years[i] == 0 {
				continue
			}
			total += -years[i] * cf.Amount / math.Pow(1+rate, years[i]+1)
		}
		return total
	}

	rate := xirrInitialGuess
	for i := 0; i < xirrMaxIterations; i++ {
		f := npv(rate)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, errors.New("xirr computation diverged")
		}
		if math.Abs(f) < xirrTolerance {
			return rate, nil
		}

		d := dnpv(rate)
		if d == 0 || math.IsNaN(d) || math.IsInf(d, 0) {
			return 0, errors.New("xirr did not converge")
		}

		newRate := rate - f/d
		if math.IsNaN(newRate) || math.IsInf(newRate, 0) {
			return 0, errors.New("xirr did not converge")
		}
		// keep (1+rate) away from <=0, where Pow with a fractional
		// exponent produces NaN.
		if newRate <= -0.999999 {
			newRate = (rate - 1) / 2
		}

		if math.Abs(newRate-rate) < xirrTolerance {
			return newRate, nil
		}
		rate = newRate
	}

	return 0, errors.New("xirr did not converge after max iterations")
}
