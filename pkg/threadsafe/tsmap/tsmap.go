// Package tsmap provides a collection of thread-safe map functions that can be safely used between multiple goroutines.
package tsmap

import (
	"github.com/Vonage/gosrvlib/pkg/maputil"
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

// Filter is a thread-safe function that returns a new map containing
// only the elements in the input map m for which the specified function f is true.
func Filter[M ~map[K]V, K comparable, V any](mux threadsafe.RLocker, m M, f func(K, V) bool) M {
	mux.RLock()
	defer mux.RUnlock()

	return maputil.Filter(m, f)
}

// Map is a thread-safe function that returns a new map that contains
// each of the elements of the input map m mutated by the specified function.
// This function can be used to invert a map.
func Map[M ~map[K]V, K, J comparable, V, U any](mux threadsafe.RLocker, m M, f func(K, V) (J, U)) map[J]U {
	mux.RLock()
	defer mux.RUnlock()

	return maputil.Map(m, f)
}

// Reduce is a thread-safe function that applies the reducing function f
// to each element of the input map m, and returns the value of the last call to f.
// The first parameter of the reducing function f is initialized with init.
func Reduce[M ~map[K]V, K comparable, V, U any](mux threadsafe.RLocker, m M, init U, f func(K, V, U) U) U {
	mux.RLock()
	defer mux.RUnlock()

	return maputil.Reduce(m, init, f)
}

// Invert is a thread-safe function that returns a new map were keys and values are swapped.
func Invert[M ~map[K]V, K, V comparable](mux threadsafe.RLocker, m M) map[V]K {
	mux.RLock()
	defer mux.RUnlock()

	return maputil.Invert(m)
}
