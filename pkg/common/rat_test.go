package common

import (
	"math/big"
	"testing"
)

func TestRatToInt64(t *testing.T) {
	cases := []struct {
		num, den int64
		want     int64
	}{
		{1, 2, 0},  // truncates toward zero
		{-1, 2, 0}, // truncates toward zero
		{3, 2, 1},
		{-3, 2, -1},
		{10, 1, 10},
	}
	for _, c := range cases {
		r := big.NewRat(c.num, c.den)
		if got := RatToInt64(r); got != c.want {
			t.Errorf("RatToInt64(%d/%d) = %d, want %d", c.num, c.den, got, c.want)
		}
	}
}

func TestRatTruncate(t *testing.T) {
	cases := []struct {
		num, den int64
		scale    int
		wantNum  int64
		wantDen  int64
	}{
		{0, 1, 5, 0, 1},         // zero stays zero
		{1, 1, -1, 1, 1},        // negative scale treated as zero
		{125, 100, 1, 12, 10},   // 1.25 -> 1.2
		{-125, 100, 1, -12, 10}, // -1.25 -> -1.2 (toward zero)
		{1, 8, 2, 12, 100},      // 0.125 -> 0.12
	}
	for _, c := range cases {
		r := big.NewRat(c.num, c.den)
		got := RatTruncate(r, c.scale)
		want := big.NewRat(c.wantNum, c.wantDen)
		if got.Cmp(want) != 0 {
			t.Errorf("RatTruncate(%d/%d, %d) = %s, want %s", c.num, c.den, c.scale, got.RatString(), want.RatString())
		}
	}
}
