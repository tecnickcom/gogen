package tsmap

import (
	"maps"
	"sync"

	"github.com/tecnickcom/nurago/pkg/maputil"
)

// Guarded is a map bundled with its own [sync.RWMutex]. All access goes through
// its methods, which take the appropriate lock, so the lock can never be
// mismatched with the data or forgotten.
//
// The zero value is ready to use and holds an empty map; write methods allocate
// the backing map lazily on first use. A Guarded must not be copied after first
// use; pass it by pointer (see [NewGuarded]).
//
// Method callbacks passed to [Guarded.Filter], [Guarded.Do], and [Guarded.RDo]
// run while the lock is held: they must be cheap, non-blocking, must not call
// back into the same Guarded (doing so deadlocks), and must not retain the map
// they receive beyond their return (using it after the lock is released races
// with other goroutines).
type Guarded[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// NewGuarded returns a [Guarded] that takes ownership of m. Pass nil for an
// initially empty map.
//
// Ownership must be exclusive: after the call the caller must not read or write
// m except through the returned Guarded, otherwise those accesses race with the
// Guarded's methods.
func NewGuarded[K comparable, V any](m map[K]V) *Guarded[K, V] {
	return &Guarded[K, V]{m: m}
}

// Len returns the map length under a read lock.
func (g *Guarded[K, V]) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.m)
}

// Get returns the value for key k under a read lock, or the zero value if the
// key is absent.
func (g *Guarded[K, V]) Get(k K) V {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.m[k]
}

// GetOK returns the value and presence flag for key k under a read lock.
func (g *Guarded[K, V]) GetOK(k K) (V, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	v, ok := g.m[k]

	return v, ok
}

// Set stores value v at key k under an exclusive lock.
func (g *Guarded[K, V]) Set(k K, v V) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.m == nil {
		g.m = make(map[K]V)
	}

	g.m[k] = v
}

// Delete removes key k under an exclusive lock. Deleting an absent key is a
// no-op.
func (g *Guarded[K, V]) Delete(k K) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.m, k)
}

// Filter returns a new map containing entries for which predicate f is true,
// under a read lock.
func (g *Guarded[K, V]) Filter(f func(K, V) bool) map[K]V {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return maputil.Filter(g.m, f)
}

// Snapshot returns a shallow copy of the map taken under a read lock. The copy
// is safe to read after the lock is released, but any reference-typed values
// remain shared with the guarded map.
func (g *Guarded[K, V]) Snapshot() map[K]V {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return maps.Clone(g.m)
}

// Do runs f under an exclusive lock, giving it the map for atomic compound
// operations (check-then-set, bulk mutation). The map passed to f is never nil.
// f must not call back into this Guarded, nor retain the map beyond its own
// return.
func (g *Guarded[K, V]) Do(f func(m map[K]V)) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.m == nil {
		g.m = make(map[K]V)
	}

	f(g.m)
}

// RDo runs f under a read lock for atomic multi-step reads (including Map,
// Reduce, or Invert via github.com/tecnickcom/nurago/pkg/maputil). f must not
// mutate the map, call back into this Guarded, or retain the map beyond its own
// return.
func (g *Guarded[K, V]) RDo(f func(m map[K]V)) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	f(g.m)
}
