/*
Package tsmap solves the common concurrency problem of safely operating on Go
maps shared across multiple goroutines without repeatedly writing lock/unlock
boilerplate at every call site.

# Problem

Go maps are not safe for concurrent read/write access. Teams typically wrap each
operation with a mutex, but this creates repetitive, error-prone code and makes
it easy to forget a lock around one path. Generic map transformations such as
filter, map, reduce, and invert are especially noisy when each operation also
needs synchronization.

tsmap provides lock-aware generic helpers that execute map operations under the
appropriate lock contract from github.com/tecnickcom/gogen/pkg/threadsafe.

# How It Works

Every function receives both the map and a lock interface:

  - write operations ([Set], [Delete]) require [threadsafe.Locker] and use
    `Lock`/`Unlock`.
  - read and pure-transform operations ([Get], [GetOK], [Len], [Filter], [Map],
    [Reduce], [Invert]) require [threadsafe.RLocker] and use `RLock`/`RUnlock`.

This design keeps synchronization policy explicit at the call site while
providing a concise, reusable API.

# Key Features

  - Generic API for any map type: functions are parameterized over
    `M ~map[K]V`, so they work with plain maps and named map types.
  - Clear read/write lock separation: mutation helpers use exclusive locks,
    query/transform helpers use read locks.
  - Functional map utilities with safety built in:
    [Filter], [Map], [Reduce], and [Invert] delegate to
    github.com/tecnickcom/gogen/pkg/maputil while preserving thread safety via
    the provided lock.
  - Minimal adoption cost: works directly with standard [sync.RWMutex]
    (or any compatible custom lock type) through interfaces.

# Usage

	var (
	    mu sync.RWMutex
	    m  = map[string]int{"a": 1, "b": 2}
	)

	tsmap.Set(&mu, m, "c", 3)
	v, ok := tsmap.GetOK(&mu, m, "a")
	_ = v
	_ = ok

	even := tsmap.Filter(&mu, m, func(_ string, n int) bool { return n%2 == 0 })
	total := tsmap.Reduce(&mu, m, 0, func(_ string, n int, acc int) int { return acc + n })
	_ = even
	_ = total

See also: github.com/tecnickcom/gogen/pkg/threadsafe
*/
package tsmap

import (
	"github.com/tecnickcom/gogen/pkg/maputil"
	"github.com/tecnickcom/gogen/pkg/threadsafe"
)

// Set is a thread-safe function to assign a value v to a key k in a map m.
func Set[M ~map[K]V, K comparable, V any](mux threadsafe.Locker, m M, k K, v V) {
	mux.Lock()
	defer mux.Unlock()

	m[k] = v
}

// Delete is a thread-safe function to delete the key-value pair with the specified key from the given map.
func Delete[M ~map[K]V, K comparable, V any](mux threadsafe.Locker, m M, k K) {
	mux.Lock()
	defer mux.Unlock()

	delete(m, k)
}

// Get is a thread-safe function to get a value by key k in a map m.
// See also GetOK.
func Get[M ~map[K]V, K comparable, V any](mux threadsafe.RLocker, m M, k K) V {
	mux.RLock()
	defer mux.RUnlock()

	return m[k]
}

// GetOK is a thread-safe function to get a value by key k in a map m.
// The second return value is a boolean that indicates whether the key was present in the map.
func GetOK[M ~map[K]V, K comparable, V any](mux threadsafe.RLocker, m M, k K) (V, bool) {
	mux.RLock()
	defer mux.RUnlock()

	v, ok := m[k]

	return v, ok
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
