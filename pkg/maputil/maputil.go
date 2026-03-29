/*
Package maputil provides generic functional-style helpers for Go maps.

# Problem

Go's built-in map type is efficient and ergonomic, but common transformation
patterns (filtering entries, remapping keys/values, reducing to an aggregate,
or inverting key-value direction) are often rewritten ad hoc across projects.
That leads to repetitive loops, inconsistent behavior, and subtle bugs around
map iteration order.

# Solution

This package offers a small set of generic, allocation-conscious helpers:
  - [Filter] keeps entries matching a predicate.
  - [Map] transforms key/value pairs into a new map type.
  - [Reduce] folds all entries into a single accumulator value.
  - [Invert] swaps keys and values.

All functions are pure map-to-map/map-to-value transforms: they return new
results and never mutate the input map directly.

# Important Semantics

Go map iteration order is intentionally randomized. Therefore:
  - [Reduce] results are deterministic only when the reducing function is
    order-independent (for example, commutative/associative operations).
  - [Map] and [Invert] follow "last write wins" semantics when multiple input
    entries map to the same output key.

# Benefits

These utilities remove repetitive boilerplate while preserving type safety,
making map-heavy code easier to read, review, and test.
*/
package maputil

// Filter returns new map containing only entries where predicate f returns true; input map is not modified.
func Filter[M ~map[K]V, K comparable, V any](m M, f func(K, V) bool) M {
	r := make(M, len(m))

	for k, v := range m {
		if f(k, v) {
			r[k] = v
		}
	}

	return r
}

// Map transforms each entry of m using f, with last-write-wins semantics for duplicate output keys.
func Map[M ~map[K]V, K, J comparable, V, U any](m M, f func(K, V) (J, U)) map[J]U {
	r := make(map[J]U, len(m))

	for k, v := range m {
		j, u := f(k, v)
		r[j] = u
	}

	return r
}

// Reduce folds m into single value by repeatedly applying f to each entry and accumulator; f should be order-independent for deterministic output.
func Reduce[M ~map[K]V, K comparable, V, U any](m M, init U, f func(K, V, U) U) U {
	r := init

	for k, v := range m {
		r = f(k, v, r)
	}

	return r
}

// Invert returns new map with swapped keys/values, with last-write-wins semantics for duplicate input values.
func Invert[M ~map[K]V, K, V comparable](m M) map[V]K {
	r := make(map[V]K, len(m))

	for k, v := range m {
		r[v] = k
	}

	return r
}
