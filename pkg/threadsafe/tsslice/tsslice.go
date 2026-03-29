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

// Set assigns value v at index k using an exclusive lock.
func Set[S ~[]E, E any](mux threadsafe.Locker, s S, k int, v E) {
	mux.Lock()
	defer mux.Unlock()

	s[k] = v
}

// Get returns value at index k under a read lock.
func Get[S ~[]E, E any](mux threadsafe.RLocker, s S, k int) E {
	mux.RLock()
	defer mux.RUnlock()

	return s[k]
}

// Len returns slice length under a read lock.
func Len[S ~[]E, E any](mux threadsafe.RLocker, s S) int {
	mux.RLock()
	defer mux.RUnlock()

	return len(s)
}

// Append appends one or more values using an exclusive lock.
func Append[S ~[]E, E any](mux threadsafe.Locker, s *S, v ...E) {
	mux.Lock()
	defer mux.Unlock()

	*s = append(*s, v...)
}

// Filter returns a new slice containing elements for which predicate f is true, under a read lock.
func Filter[S ~[]E, E any](mux threadsafe.RLocker, s S, f func(int, E) bool) S {
	mux.RLock()
	defer mux.RUnlock()

	return sliceutil.Filter(s, f)
}

// Map transforms each element under a read lock and returns a new slice.
func Map[S ~[]E, E any, U any](mux threadsafe.RLocker, s S, f func(int, E) U) []U {
	mux.RLock()
	defer mux.RUnlock()

	return sliceutil.Map(s, f)
}

// Reduce folds slice elements under a read lock starting from init.
func Reduce[S ~[]E, E any, U any](mux threadsafe.RLocker, s S, init U, f func(int, E, U) U) U {
	mux.RLock()
	defer mux.RUnlock()

	return sliceutil.Reduce(s, init, f)
}
