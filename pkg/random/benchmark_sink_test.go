package random

// Package-level sinks, assigned by the benchmarks so that the compiler cannot
// discard the work being measured.
//
// Assigning a result to the blank identifier instead lets escape analysis keep it
// on the stack and lets dead-code elimination drop it entirely, so the reported
// numbers understate the real cost: the discarded form of TUID128.Hex measures
// 8 B/op — only the entropy read — although the string it returns is 32 bytes.
// These numbers are what the package's allocation docs are written from, so a
// misleading benchmark becomes a misleading doc.
//
// For the slice-returning Byte helpers the allocation is escape-dependent, so both
// forms are benchmarked: _NoEscape (the result stays local) and _Escaping (the
// result outlives the call). Both numbers are real; the docs cite both.
//
//nolint:gochecknoglobals // package-level by design: a local would be optimized away.
var (
	sinkString string
	sinkBytes  []byte
	sinkByte   byte
	sinkUint32 uint32
	sinkUint64 uint64
	sinkUUID   UUID
	sinkUID64  TUID64
	sinkUID128 TUID128
	errSink    error
)
