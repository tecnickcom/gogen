/*
Package sliceutil provides generic, allocation-conscious helpers for common
slice operations and numeric dataset summarization.

# Problem

Go intentionally keeps slice operations minimal in the standard library. Teams
typically re-implement the same small helpers for filter/map/reduce in multiple
packages, and statistical summary code is often copied with subtle differences.
This package centralizes those patterns into a reusable, type-safe API.

# What It Provides

Functional slice primitives:

  - [Filter]: returns a new slice containing elements that satisfy a predicate.
  - [Map]: returns a new slice with each element transformed to a new type.
  - [Reduce]: folds a slice into a single value using an accumulator.

Descriptive statistics for numeric slices:

  - [Stats]: computes a [DescStats] summary for any numeric slice type.
  - [DescStats] includes count, sum, min/max (+ indexes), range, mode,
    mean/median, entropy, variance, standard deviation, skewness, and excess
    kurtosis.

# Key Features

  - Generic APIs (`S ~[]E`) that work with native and named slice types.
  - Functional helpers include element index in callbacks for context-aware
    transforms.
  - Numeric statistics are available for all [typeutil.Number] types.
  - Safe error behavior for invalid input: [Stats] returns an error for empty
    slices.
  - Clear composition model: use [Filter], [Map], and [Reduce] in pipelines,
    then summarize results with [Stats].

# Usage

	adults := sliceutil.Filter(users, func(_ int, u User) bool { return u.Age >= 18 })
	names := sliceutil.Map(adults, func(_ int, u User) string { return u.Name })
	total := sliceutil.Reduce([]int{1, 2, 3, 4}, 0, func(_ int, v int, acc int) int {
	    return acc + v
	})
	_ = names
	_ = total

	ds, err := sliceutil.Stats([]int{53, 83, 13, 79})
	if err != nil {
	    return err
	}
	_ = ds.Mean

This package is ideal for Go services and libraries that want concise,
predictable slice transformations and lightweight statistical summaries without
pulling in heavy data-processing dependencies.
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
