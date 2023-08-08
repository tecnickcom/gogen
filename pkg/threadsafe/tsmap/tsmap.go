// Package tsmap provides a collection of thread-safe map functions that can be safely used between multiple goroutines.
package tsmap

import (
	"github.com/Vonage/gosrvlib/pkg/threadsafe"
)

// Set is a thread-safe function to assign a value v to a key k in a map m.
func Set[M ~map[K]V, K comparable, V any](mux threadsafe.Locker, m M, k K, v V) {
	mux.Lock()
	defer mux.Unlock()

	m[k] = v
}

// Get is a thread-safe function to get a value by key k in a map m.
func Get[M ~map[K]V, K comparable, V any](mux threadsafe.RLocker, m M, k K) V {
	mux.RLock()
	defer mux.RUnlock()

	return m[k]
}

// Len is a thread-safe function to get the length of a map m.
func Len[M ~map[K]V, K comparable, V any](mux threadsafe.RLocker, m M) int {
	mux.RLock()
	defer mux.RUnlock()

	return len(m)
}
