/*
Package tsslice provides a collection of generic thread-safe slice utility functions that can be safely used between multiple goroutines.

The provided functions are intended to simplify the process of working with slices in a thread-safe manner.
*/
package tsslice

import (
	"github.com/Vonage/gosrvlib/pkg/sliceutil"
	"github.com/Vonage/gosrvlib/pkg/threadsafe"
)

// Set is a thread-safe function to assign a value v to a key k in a slice s.
func Set[S ~[]E, E any](mux threadsafe.Locker, s S, k int, v E) {
	mux.Lock()
	defer mux.Unlock()

	s[k] = v
}

// Get is a thread-safe function to get a value by key k in a slice.
func Get[S ~[]E, E any](mux threadsafe.RLocker, s S, k int) E {
	mux.RLock()
	defer mux.RUnlock()

	return s[k]
}

// Len is a thread-safe function to get the length of a slice.
func Len[S ~[]E, E any](mux threadsafe.RLocker, s S) int {
	mux.RLock()
	defer mux.RUnlock()

	return len(s)
}

// Append is a thread-safe version of the Go built-in append function.
// Appends the value v to the slice s.
func Append[S ~[]E, E any](mux threadsafe.Locker, s *S, v ...E) {
	mux.Lock()
	defer mux.Unlock()

	*s = append(*s, v...)
}

// Filter is a thread-safe function that returns a new slice containing
// only the elements in the input slice s for which the specified function f is true.
func Filter[S ~[]E, E any](mux threadsafe.RLocker, s S, f func(int, E) bool) S {
	mux.RLock()
	defer mux.RUnlock()

	return sliceutil.Filter(s, f)
}

// Map is a thread-safe function that returns a new slice that contains
// each of the elements of the input slice s mutated by the specified function.
func Map[S ~[]E, E any, U any](mux threadsafe.RLocker, s S, f func(int, E) U) []U {
	mux.RLock()
	defer mux.RUnlock()

	return sliceutil.Map(s, f)
}

// Reduce is a thread-safe function that applies the reducing function f
// to each element of the input slice s, and returns the value of the last call to f.
// The first parameter of the reducing function f is initialized with init.
func Reduce[S ~[]E, E any, U any](mux threadsafe.RLocker, s S, init U, f func(int, E, U) U) U {
	mux.RLock()
	defer mux.RUnlock()

	return sliceutil.Reduce(s, init, f)
}
