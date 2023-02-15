// Package tsmap provides a collection of thread-safe map functions that can be safely used between multiple goroutines.
package tsmap

import (
	"github.com/Vonage/gosrvlib/pkg/threadsafe"
)

// Set is a thread-safe function to assign a value to a key in a map.
func Set[K comparable, V any](mux threadsafe.Locker, m map[K]V, key K, value V) {
	mux.Lock()
	defer mux.Unlock()

	m[key] = value
}

// Get is a thread-safe function to get a value by key in a map.
func Get[K comparable, V any](mux threadsafe.RLocker, m map[K]V, key K) V {
	mux.RLock()
	defer mux.RUnlock()

	return m[key]
}

// Len is a thread-safe function to get the length of a map.
func Len[K comparable, V any](mux threadsafe.RLocker, m map[K]V) int {
	mux.RLock()
	defer mux.RUnlock()

	return len(m)
}
