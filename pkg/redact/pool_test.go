package redact

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // Swaps the shared pool to validate pooled buffer behavior.
func TestPooledBufferHelpers(t *testing.T) {
	// Intentionally not parallel: this test replaces the shared pool. Restore an
	// equivalent pool afterwards so other tests/benchmarks keep working.
	t.Cleanup(func() { redactionBufferPool = sync.Pool{New: newRedactionBuffer} })

	// Pool whose New returns a too-small buffer, forcing the grow-path fallback.
	redactionBufferPool = sync.Pool{New: func() any {
		b := make([]byte, 0, 1)

		return &b
	}}

	b := getPooledRedactionBuffer(2 << 20)
	require.GreaterOrEqual(t, cap(b), 2<<20)

	// Pool whose New returns an unexpected value type, forcing the nil-assertion
	// fallback in getPooledRedactionBuffer.
	redactionBufferPool = sync.Pool{New: func() any { return &struct{}{} }}

	b = getPooledRedactionBuffer(64)
	require.GreaterOrEqual(t, cap(b), 64)

	// Pool whose New returns a usable buffer, exercising the reuse path.
	redactionBufferPool = sync.Pool{New: func() any {
		b := make([]byte, 0, 256)

		return &b
	}}

	b = getPooledRedactionBuffer(64)
	require.GreaterOrEqual(t, cap(b), 64)

	// Oversized buffers are dropped; right-sized buffers are returned to the pool.
	putPooledRedactionBuffer(make([]byte, 0, 2<<20))
	putPooledRedactionBuffer(make([]byte, 0, 128))
}
