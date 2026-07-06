package phonekeypad_test

import (
	"testing"

	"github.com/tecnickcom/gogen/pkg/phonekeypad"
)

// benchInput mixes digits, letters, and separators to exercise every path.
const benchInput = "1-800-FLOWERS (555) 123-GOGEN"

func BenchmarkKeypadDigit(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_, _ = phonekeypad.KeypadDigit('S')
	}
}

func BenchmarkKeypadNumber(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = phonekeypad.KeypadNumber(benchInput)
	}
}

func BenchmarkKeypadNumberString(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = phonekeypad.KeypadNumberString(benchInput)
	}
}
