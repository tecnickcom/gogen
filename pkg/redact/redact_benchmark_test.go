package redact

import "testing"

// Benchmarks run against the shared Default instance, the configuration every
// caller gets when no redact function is wired.

func BenchmarkString(b *testing.B) {
	for range b.N {
		_ = Default().String(benchmarkHTTPDataInput)
	}
}

func BenchmarkBytes(b *testing.B) {
	input := []byte(benchmarkHTTPDataInput)

	for range b.N {
		_ = Default().Bytes(input)
	}
}

func BenchmarkAppendTo(b *testing.B) {
	input := []byte(benchmarkHTTPDataInput)
	dst := make([]byte, 0, len(input))

	for range b.N {
		dst = Default().AppendTo(dst, input)
	}
}

func BenchmarkPooled(b *testing.B) {
	input := []byte(benchmarkHTTPDataInput)

	for range b.N {
		Default().Pooled(input, func(out []byte) {
			_ = out
		})
	}
}

// benchmarkDigitHeavyInput mirrors the identifier-dense shape of production
// log lines (trace ids, UUIDs, ports, durations), which digit-run handling
// dominates.
var benchmarkDigitHeavyInput = []byte(`level=info trace=a1b2c3d4e5f67890a1b2c3d4e5f67890 span=0123abcd4567ef89 msg="GET /api/v1/users/123e4567-e89b-12d3-a456-426614174000" host=10.0.0.5:8080 dur=12.345ms code=200`) //nolint:gochecknoglobals

func BenchmarkAppendToDigitHeavy(b *testing.B) {
	dst := make([]byte, 0, len(benchmarkDigitHeavyInput))

	for range b.N {
		dst = Default().AppendTo(dst, benchmarkDigitHeavyInput)
	}
}
