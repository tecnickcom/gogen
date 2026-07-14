package redact

import "sync"

// redactionBufferPool backs [Redactor.Pooled] (and, through it,
// [Redactor.BytesToString]) with reusable output buffers, so the hot logging
// path does not allocate a destination per call.
var redactionBufferPool = sync.Pool{New: newRedactionBuffer} //nolint:gochecknoglobals

// newRedactionBuffer is the sync.Pool factory for reusable output buffers.
func newRedactionBuffer() any {
	b := make([]byte, 0, 1024)

	return &b
}

func getPooledRedactionBuffer(minCap int) []byte {
	bp, _ := redactionBufferPool.Get().(*[]byte)
	if bp == nil {
		b := make([]byte, 0, minCap)

		return b
	}

	b := *bp
	if cap(b) < minCap {
		return make([]byte, 0, minCap)
	}

	return b[:0]
}

func putPooledRedactionBuffer(b []byte) {
	// Avoid keeping very large buffers indefinitely in the pool.
	const maxPooledCap = 1 << 20
	if cap(b) > maxPooledCap {
		return
	}

	b = b[:0]
	redactionBufferPool.Put(&b)
}
