package enumcache_test

import (
	"strconv"
	"testing"

	"github.com/tecnickcom/gogen/pkg/enumcache"
)

// benchCache returns a cache seeded with the 32 single-bit flags flag0..flag31.
func benchCache() *enumcache.EnumCache {
	ec := enumcache.New()

	for n := range 32 {
		ec.Set(1<<n, "flag"+strconv.Itoa(n))
	}

	return ec
}

func BenchmarkSet_overwrite(b *testing.B) {
	ec := benchCache()

	b.ReportAllocs()

	for b.Loop() {
		ec.Set(1, "flag0")
	}
}

func BenchmarkID(b *testing.B) {
	ec := benchCache()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = ec.ID("flag5")
	}
}

func BenchmarkName(b *testing.B) {
	ec := benchCache()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = ec.Name(1 << 5)
	}
}

func BenchmarkDecodeBinaryMap(b *testing.B) {
	ec := benchCache()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = ec.DecodeBinaryMap(0b10101010)
	}
}

func BenchmarkEncodeBinaryMap(b *testing.B) {
	ec := benchCache()
	names := []string{"flag1", "flag3", "flag5", "flag7"}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = ec.EncodeBinaryMap(names)
	}
}
