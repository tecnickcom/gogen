package enumbitmap_test

import (
	"strconv"
	"testing"

	"github.com/tecnickcom/gogen/pkg/enumbitmap"
)

// benchEnums builds the two full 32-entry lookup maps used by the benchmarks,
// mapping each single-bit mask (1<<0 .. 1<<31) to its binary string name.
func benchEnums() (map[string]int, map[int]string) {
	esi := make(map[string]int, 32)
	eis := make(map[int]string, 32)

	for bit := range 32 {
		mask := 1 << bit
		name := strconv.FormatInt(int64(mask), 2)
		esi[name] = mask
		eis[mask] = name
	}

	return esi, eis
}

func BenchmarkBitMapToStrings_sparse(b *testing.B) {
	_, eis := benchEnums()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = enumbitmap.BitMapToStrings(eis, 0b1001)
	}
}

func BenchmarkBitMapToStrings_full(b *testing.B) {
	_, eis := benchEnums()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = enumbitmap.BitMapToStrings(eis, -1)
	}
}

func BenchmarkStringsToBitMap(b *testing.B) {
	esi, _ := benchEnums()
	names := []string{"1", "1000", "10000000", "10000000000000000000000000000000"}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = enumbitmap.StringsToBitMap(esi, names)
	}
}
