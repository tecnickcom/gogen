package numtrie_test

import (
	"testing"

	"github.com/tecnickcom/gogen/pkg/numtrie"
)

func BenchmarkAdd(b *testing.B) {
	root := numtrie.New[int]()
	val := 42

	b.ReportAllocs()

	for b.Loop() {
		root.Add("1-212-555-0100", &val)
	}
}

func BenchmarkGet(b *testing.B) {
	root := numtrie.New[int]()

	v1, v2, v3 := 1, 2, 3
	root.Add("1", &v1)
	root.Add("1212", &v2)
	root.Add("1212555", &v3)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = root.Get("+1-212-555-0100")
	}
}

func BenchmarkGetExact(b *testing.B) {
	root := numtrie.New[int]()

	v := 7
	root.Add("1212555", &v)

	b.ReportAllocs()

	for b.Loop() {
		_ = root.GetExact("1-212-555")
	}
}
