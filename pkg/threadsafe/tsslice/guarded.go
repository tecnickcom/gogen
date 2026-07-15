package tsslice

import (
	"slices"
	"sync"

	"github.com/tecnickcom/nurago/pkg/sliceutil"
)

// Guarded is a slice bundled with its own [sync.RWMutex]. All access goes
// through its methods, which take the appropriate lock, so the lock can never
// be mismatched with the data or forgotten.
//
// The zero value is ready to use and holds an empty slice. A Guarded must not
// be copied after first use; pass it by pointer (see [NewGuarded]).
//
// Method callbacks passed to [Guarded.Filter], [Guarded.Do], and [Guarded.RDo]
// run while the lock is held: they must be cheap, non-blocking, must not call
// back into the same Guarded (doing so deadlocks), and must not retain the slice
// or pointer they receive beyond their return (using it after the lock is
// released races with other goroutines).
type Guarded[E any] struct {
	mu sync.RWMutex
	s  []E
}

// NewGuarded returns a [Guarded] that takes ownership of s. Pass nil for an
// initially empty slice.
//
// Ownership must be exclusive: after the call the caller must not read or write
// s (nor any other slice that aliases the same backing array, such as a
// spare-capacity re-slice) except through the returned Guarded, otherwise those
// accesses race with the Guarded's methods.
func NewGuarded[E any](s []E) *Guarded[E] {
	return &Guarded[E]{s: s}
}

// Len returns the slice length under a read lock.
func (g *Guarded[E]) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.s)
}

// Get returns the value at index k under a read lock. It panics if k is out of
// range, mirroring built-in slice indexing; use [Guarded.GetOK] for a
// bounds-checked, non-panicking alternative.
func (g *Guarded[E]) Get(k int) E {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.s[k]
}

// GetOK returns the value at index k under a read lock and reports whether k was
// within range. It never panics: an out-of-range index returns the zero value
// and false.
func (g *Guarded[E]) GetOK(k int) (E, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if k < 0 || k >= len(g.s) {
		var zero E

		return zero, false
	}

	return g.s[k], true
}

// Set assigns value v at index k under an exclusive lock. It panics if k is out
// of range; use [Guarded.SetOK] for a bounds-checked, non-panicking alternative.
func (g *Guarded[E]) Set(k int, v E) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.s[k] = v
}

// SetOK assigns value v at index k under an exclusive lock and reports whether k
// was within range. It never panics: an out-of-range index is a no-op that
// returns false.
func (g *Guarded[E]) SetOK(k int, v E) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if k < 0 || k >= len(g.s) {
		return false
	}

	g.s[k] = v

	return true
}

// Append appends one or more values under an exclusive lock.
func (g *Guarded[E]) Append(v ...E) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.s = append(g.s, v...)
}

// Delete removes the element at index k under an exclusive lock, preserving the
// order of the remaining elements. It reports whether k was within range; an
// out-of-range index is a no-op that returns false, so Delete never panics.
func (g *Guarded[E]) Delete(k int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if k < 0 || k >= len(g.s) {
		return false
	}

	g.s = slices.Delete(g.s, k, k+1)

	return true
}

// Filter returns a new slice containing elements for which predicate f is true,
// under a read lock.
func (g *Guarded[E]) Filter(f func(int, E) bool) []E {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return sliceutil.Filter(g.s, f)
}

// Snapshot returns a shallow copy of the slice taken under a read lock. The copy
// is safe to read after the lock is released, but any reference-typed elements
// remain shared with the guarded slice.
func (g *Guarded[E]) Snapshot() []E {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return slices.Clone(g.s)
}

// Do runs f under an exclusive lock, giving it a pointer to the slice for atomic
// compound operations (read-modify-write, conditional growth, transforms to a
// different type). f must not call back into this Guarded, nor retain the
// pointer beyond its own return.
func (g *Guarded[E]) Do(f func(s *[]E)) {
	g.mu.Lock()
	defer g.mu.Unlock()

	f(&g.s)
}

// RDo runs f under a read lock, giving it the current slice value for atomic
// multi-step reads (including Map/Reduce to a different type via
// github.com/tecnickcom/nurago/pkg/sliceutil). f must not mutate the slice, call
// back into this Guarded, or retain the slice beyond its own return.
func (g *Guarded[E]) RDo(f func(s []E)) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	f(g.s)
}
