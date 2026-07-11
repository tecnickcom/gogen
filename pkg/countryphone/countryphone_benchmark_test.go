package countryphone_test

import (
	"testing"

	"github.com/tecnickcom/nurago/pkg/countryphone"
)

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		_ = countryphone.New(nil)
	}
}

func BenchmarkData_NumberInfo(b *testing.B) {
	data := countryphone.New(nil)

	b.ResetTimer()

	for b.Loop() {
		_, _ = data.NumberInfo("1357123456")
	}
}

func BenchmarkData_NumberInfo_noMatch(b *testing.B) {
	data := countryphone.New(nil)

	b.ResetTimer()

	for b.Loop() {
		_, _ = data.NumberInfo("999999999")
	}
}

func BenchmarkData_NumberType(b *testing.B) {
	data := countryphone.New(nil)

	b.ResetTimer()

	for b.Loop() {
		_, _ = data.NumberType(2)
	}
}
