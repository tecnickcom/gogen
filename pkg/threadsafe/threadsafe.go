/*
Package threadsafe provides an interface for thread-safe functions that can be safely used between multiple goroutines.

See the tsmap and tsslice packages for examples of how to use this interface.
*/
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
