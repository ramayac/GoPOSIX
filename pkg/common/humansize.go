// Package common provides shared helpers for goposix utilities.
package common

import (
	"fmt"
	"math"
)

// HumanSize formats a byte count in human-readable form (e.g. "1.5K").
//
// When round is true, the value is rounded to one decimal place (du style).
// When round is false, the value is truncated to one decimal place (ls style).
func HumanSize(n int64, round bool) string {
	units := []string{"B", "K", "M", "G", "T", "P"}
	f := float64(n)
	idx := 0
	for f >= 1024 && idx < len(units)-1 {
		f /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%.0f%s", f, units[idx])
	}
	if round {
		f = math.Round(f*10) / 10
	}
	return fmt.Sprintf("%.1f%s", f, units[idx])
}
