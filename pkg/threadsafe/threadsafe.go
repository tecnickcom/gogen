// Package threadsafe provides a collection of thread-safe functions that can be safely used between multiple goroutines.
package threadsafe

import (
	"sync"
)

// Append is a thread-safe version of the Go built-in append function.
func Append[T any](mux sync.Locker, slice *[]T, item ...T) {
	mux.Lock()
	defer mux.Unlock()

	*slice = append(*slice, item...)
}
