package math2

import (
	"cmp"
)

// Max is equivalent to built-in max.
// Deprecated: use max directly
//
//go:fix inline
func Max[N cmp.Ordered](a, b N) N {
	return max(a, b)
}

// Min is equivalent to built-in min.
// Deprecated: use min directly
//
//go:fix inline
func Min[N cmp.Ordered](a, b N) N {
	return min(a, b)
}
