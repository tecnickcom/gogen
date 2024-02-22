/*
Package sliceutil provides a collection of slice utility functions, including descriptive statistics functions for numerical slices.
*/
package sliceutil

// Filter returns a new slice containing
// only the elements in the input slice s for which the specified function f is true.
func Filter[S ~[]E, E any](s S, f func(int, E) bool) S {
	r := make(S, 0)

	for k, v := range s {
		if f(k, v) {
			r = append(r, v)
		}
	}

	return r
}

// Map returns a new slice that contains
// each of the elements of the input slice s mutated by the specified function.
func Map[S ~[]E, E any, U any](s S, f func(int, E) U) []U {
	r := make([]U, len(s))

	for k, v := range s {
		r[k] = f(k, v)
	}

	return r
}

// Reduce applies the reducing function f
// to each element of the input slice s, and returns the value of the last call to f.
// The first parameter of the reducing function f is initialized with init.
func Reduce[S ~[]E, E any, U any](s S, init U, f func(int, E, U) U) U {
	r := init

	for k, v := range s {
		r = f(k, v, r)
	}

	return r
}
