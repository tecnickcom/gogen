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

  - [Locker]: the write-lock contract (`Lock` and `Unlock`), aliased from
    [sync.Locker].
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
  - github.com/tecnickcom/gogen/pkg/tsmap
  - github.com/tecnickcom/gogen/pkg/tsslice

Those packages demonstrate how these interfaces are used to build practical,
thread-safe containers.
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
