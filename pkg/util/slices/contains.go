package slices

import "slices"

// Contains returns true if haystack contains needle.
func Contains[S1 ~[]E, E comparable](needle E, haystack S1) bool {
	return slices.Contains(haystack, needle)
}
