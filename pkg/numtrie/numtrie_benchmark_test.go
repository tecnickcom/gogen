package numtrie_test

import (
	"fmt"
	"testing"

	"github.com/tecnickcom/nurago/pkg/numtrie"
)

func BenchmarkAdd(b *testing.B) {
	// Distinct keys inserted into a fresh trie per op, so each Add allocates
	// real nodes. Re-adding a single key would report a misleading 0 allocs/op
	// because every node already exists after the first iteration.
	const numKeys = 1000

	keys := make([]string, numKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("%07d", i)
	}

	val := 42

	b.ReportAllocs()

	for b.Loop() {
		root := numtrie.New[int]()
		for _, k := range keys {
			root.Add(k, &val)
		}
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
