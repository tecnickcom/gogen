/*
Package threadsafe provides an interface for thread-safe functions that can be
safely used across multiple goroutines.

See the examples in the github.com/Vonage/gosrvlib/pkg/tsmap and
github.com/Vonage/gosrvlib/pkg/tsslice packages for usage of this interface.
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
