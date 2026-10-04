// Package common provides shared helpers for goposix utilities.
package common

import "math/big"

// RatToInt64 truncates r toward zero and returns the result as an int64.
// This matches the integer-truncation semantics used by bc and dc.
func RatToInt64(r *big.Rat) int64 {
	intPart := new(big.Int).Quo(r.Num(), r.Denom())
	return intPart.Int64()
}

// RatTruncate truncates r toward zero to the given number of decimal places.
// A negative scale is treated as zero. A zero value stays zero.
func RatTruncate(r *big.Rat, scale int) *big.Rat {
	if r.Sign() == 0 {
		return new(big.Rat)
	}
	if scale < 0 {
		scale = 0
	}
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	scaled := new(big.Rat).Set(r)
	scaled.Mul(scaled, new(big.Rat).SetInt(factor))

	// Use Quo (truncation toward zero), not Div (floor toward -infinity).
	intPart := new(big.Int).Quo(scaled.Num(), scaled.Denom())

	return new(big.Rat).SetFrac(intPart, factor)
}
