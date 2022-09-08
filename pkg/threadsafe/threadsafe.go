// Package threadsafe provides a collection of thread-safe functions that can be safely used between multiple goroutines.
// Check the sub-packages for specific implementations.
package threadsafe

import (
	"sync"
)

// Locker is an interface that implements the Lock and Unlock methods.
type Locker sync.Locker

// RLocker is an interface that implements the RLock and RUnlock methods.
type RLocker interface {
	RLock()
	RUnlock()
}
