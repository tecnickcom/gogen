// Package tsslice provides a collection of thread-safe slice functions that can be safely used between multiple goroutines.
package tsslice

import (
	"github.com/nexmoinc/gosrvlib/pkg/threadsafe"
)

// Append is a thread-safe version of the Go built-in append function.
func Append[T any](mux threadsafe.Locker, slice *[]T, item ...T) {
	mux.Lock()
	defer mux.Unlock()

	*slice = append(*slice, item...)
}

// Set is a thread-safe function to assign a value to a key in a slice.
func Set[T any](mux threadsafe.Locker, slice []T, key int, value T) {
	mux.Lock()
	defer mux.Unlock()

	slice[key] = value
}

// Get is a thread-safe function to get a value by key in a slice.
func Get[T any](mux threadsafe.RLocker, slice []T, key int) T {
	mux.RLock()
	defer mux.RUnlock()

	return slice[key]
}

// Len is a thread-safe function to get the length of a slice.
func Len[T any](mux threadsafe.RLocker, slice []T) int {
	mux.RLock()
	defer mux.RUnlock()

	return len(slice)
}
