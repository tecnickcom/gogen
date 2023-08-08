// Package tsslice provides a collection of thread-safe slice functions that can be safely used between multiple goroutines.
package tsslice

import (
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
