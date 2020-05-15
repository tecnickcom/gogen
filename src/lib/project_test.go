package ~#LIBPACKAGE#~

import (
	"testing"
)

func TestGetDesc(t *testing.T) {
	exp := "test string"
	i := &Info{Desc: exp}
	res := i.GetDesc()
	if res != exp {
		t.Errorf("The strings are different: %s <> %s", res, exp)
	}
}

func BenchmarkGetDesc(b *testing.B) {
	info := &Info{Desc: "test"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info.GetDesc()
	}
}
