package domain

import (
	"math"
	"strconv"
)

// Float returns a pointer to v. Nullable JSON numbers are modelled as *float64.
func Float(v float64) *float64 {
	return &v
}

// Round rounds v to the given number of decimal places.
func Round(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}

// Percent returns covered/total as a percentage rounded to two decimals, or
// nil when the denominator is zero. An empty denominator is never 100%.
func Percent(covered, total int) *float64 {
	if total == 0 {
		return nil
	}
	return Float(Round(float64(covered)/float64(total)*100, 2))
}

// FormatNumber renders a value without a trailing ".0" for whole numbers.
func FormatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
