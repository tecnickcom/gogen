package threadsafe_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/threadsafe"
)

// Compile-time assertions documenting the intended lock implementers.
var (
	_ threadsafe.Locker  = (*sync.Mutex)(nil)
	_ threadsafe.Locker  = (*sync.RWMutex)(nil)
	_ threadsafe.RLocker = (*sync.RWMutex)(nil)
)

func TestLockerContract(t *testing.T) {
	t.Parallel()

	var (
		mu  sync.RWMutex
		n   int
		got int
	)

	// A *sync.RWMutex drives both the write and the read contract with the
	// same instance; exercise each guarding a real variable.
	func() {
		var lk threadsafe.Locker = &mu

		lk.Lock()
		defer lk.Unlock()

		n++
	}()

	func() {
		var rlk threadsafe.RLocker = &mu

		rlk.RLock()
		defer rlk.RUnlock()

		got = n
	}()

	require.Equal(t, 1, got)
}
