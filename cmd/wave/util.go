package main

// Clamp returns n constrained to the inclusive range [lo, hi].
// If lo > hi the bounds are swapped before clamping.
func Clamp(n, lo, hi int) int {
	if lo > hi {
		lo, hi = hi, lo
	}
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}
