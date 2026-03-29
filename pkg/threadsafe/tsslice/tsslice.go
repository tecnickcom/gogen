/*
Package tsslice solves the common concurrency problem of safely operating on
Go slices shared across multiple goroutines without repeating lock boilerplate
around every access.

# Problem

Slices are lightweight and ubiquitous, but concurrent reads and writes to shared
slice state can cause data races and undefined behavior. In practice, teams end
up duplicating `Lock`/`Unlock` and `RLock`/`RUnlock` blocks around simple
operations (set, get, append, transform), making code noisy and increasing the
chance of forgetting synchronization on one code path.

tsslice provides lock-aware generic helpers that keep synchronization explicit
and consistent while preserving the ergonomics of slice utility operations.

# How It Works

Each function takes both the slice and a lock interface from
github.com/tecnickcom/gogen/pkg/threadsafe:

  - write operations ([Set], [Append]) require [threadsafe.Locker] and use
    exclusive `Lock`/`Unlock`.
  - read and pure-transform operations ([Get], [Len], [Filter], [Map],
    [Reduce]) require [threadsafe.RLocker] and use shared `RLock`/`RUnlock`.

The helpers delegate functional operations to github.com/tecnickcom/gogen/pkg/sliceutil
while enforcing synchronization around the read window.

# Key Features

  - Generic API for any slice type via `S ~[]E`.
  - Clear separation of mutation vs read/transform lock requirements.
  - Thread-safe functional utilities:
    [Filter], [Map], and [Reduce] for expressive transformations on shared
    slice state.
  - Minimal adoption cost: compatible with standard [sync.RWMutex] and custom
    lock implementations that satisfy the interfaces.

# Usage

	var (
	    mu sync.RWMutex
	    s  = []int{1, 2, 3}
	)

	tsslice.Set(&mu, s, 0, 10)
	v := tsslice.Get(&mu, s, 1)
	_ = v

	tsslice.Append(&mu, &s, 4, 5)
	even := tsslice.Filter(&mu, s, func(_ int, n int) bool { return n%2 == 0 })
	total := tsslice.Reduce(&mu, s, 0, func(_ int, n int, acc int) int { return acc + n })
	_ = even
	_ = total

See also: github.com/tecnickcom/gogen/pkg/threadsafe
*/
package tsslice

import (
	"github.com/tecnickcom/gogen/pkg/sliceutil"
	"github.com/tecnickcom/gogen/pkg/threadsafe"
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
