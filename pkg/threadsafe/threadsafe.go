/*
Package threadsafe solves a small but important design problem in concurrent Go
code: how to write reusable, goroutine-safe data structures and helpers without
hard-coding a concrete lock type everywhere.

# Problem

Many concurrency-aware packages need only a tiny synchronization contract
(`Lock`/`Unlock` or `RLock`/`RUnlock`), but directly depending on concrete types
like [sync.Mutex] or [sync.RWMutex] makes code less composable and harder to
test. A package-level interface abstraction allows callers and dependent
packages to share thread-safety behavior while keeping implementation details
flexible.

# What This Package Provides

This package intentionally stays minimal and defines two lock interfaces:

  - [Locker]: the write-lock contract (`Lock` and `Unlock`), a defined type
    sharing the method set of [sync.Locker].
  - [RLocker]: the read-lock contract (`RLock` and `RUnlock`) used by
    read/write synchronization patterns.

These interfaces are intended to be embedded or referenced by concurrent
containers and utility types that must be safely used across multiple
goroutines.

# Why It Matters

  - Decouples concurrency contracts from concrete lock implementations.
  - Improves reuse across packages that need the same minimal synchronization
    surface.
  - Simplifies testing and composition when custom lock wrappers are used.

# Usage

See the examples in:
  - github.com/tecnickcom/nurago/pkg/threadsafe/tsmap
  - github.com/tecnickcom/nurago/pkg/threadsafe/tsslice

Those packages demonstrate how these interfaces are used to build practical,
thread-safe containers.

Read helpers in dependent packages accept [RLocker], which a plain [sync.Mutex]
does not satisfy (it has no `RLock`/`RUnlock`). This is deliberate: it forces
callers that need shared read access to supply an [sync.RWMutex]-like type at
compile time, rather than silently serializing every read behind an exclusive
lock. A [sync.RWMutex] satisfies both [Locker] and [RLocker], so the same lock
instance can drive read and write helpers.

These interfaces only describe the locking contract; they do not enforce it.
All access to the protected data must funnel through the helper functions using
the same lock instance, otherwise concurrent reads and writes are not safe.
*/
package threadsafe

import (
	"sync"
)

// Locker defines exclusive lock semantics with Lock and Unlock methods.
type Locker sync.Locker

// RLocker defines shared read-lock semantics with RLock and RUnlock methods.
type RLocker interface {
	RLock()
	RUnlock()
}
