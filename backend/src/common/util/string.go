package util

import "math"

// Truncate the provided string up to provided size
func Truncate(s string, size float64) string {
	return s[:int(math.Min(float64(len(s)), size))]
}
