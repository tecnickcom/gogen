package redact

import "testing"

func BenchmarkHTTPData(b *testing.B) {
	for range b.N {
		_ = HTTPData(benchmarkHTTPDataInput)
	}
}

func BenchmarkHTTPDataBytes(b *testing.B) {
	for range b.N {
		_ = HTTPDataBytes([]byte(benchmarkHTTPDataInput))
	}
}

func BenchmarkHTTPDataBytesInto(b *testing.B) {
	input := []byte(benchmarkHTTPDataInput)
	dst := make([]byte, 0, len(input))

	for range b.N {
		dst = HTTPDataBytesInto(dst, input)
	}
}

func BenchmarkHTTPDataBytesPooled(b *testing.B) {
	input := []byte(benchmarkHTTPDataInput)

	for range b.N {
		HTTPDataBytesPooled(input, func(out []byte) {
			_ = out
		})
	}
}
