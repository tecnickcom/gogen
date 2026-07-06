package enumcache

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New_Set_ID_Name(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	id, err := ec.ID("alpha")
	require.Error(t, err)
	require.Empty(t, id)

	name, err := ec.Name(1)
	require.Error(t, err)
	require.Empty(t, name)

	ec.Set(1, "alpha")

	id, err = ec.ID("alpha")
	require.NoError(t, err)
	require.Equal(t, 1, id)

	name, err = ec.Name(1)
	require.NoError(t, err)
	require.Equal(t, "alpha", name)

	ec.Set(2, "bravo")

	id, err = ec.ID("bravo")
	require.NoError(t, err)
	require.Equal(t, 2, id)

	name, err = ec.Name(2)
	require.NoError(t, err)
	require.Equal(t, "bravo", name)
}

func Test_Set_overwriteSameID_dropsStaleName(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	ec.Set(1, "a")
	ec.Set(1, "b")

	// The reverse map must no longer resolve the stale name.
	id, err := ec.ID("a")
	require.Error(t, err)
	require.Empty(t, id)

	id, err = ec.ID("b")
	require.NoError(t, err)
	require.Equal(t, 1, id)

	name, err := ec.Name(1)
	require.NoError(t, err)
	require.Equal(t, "b", name)

	require.Equal(t, []string{"b"}, ec.SortNames())
	require.Equal(t, []int{1}, ec.SortIDs())
}

func Test_Set_overwriteSameName_dropsStaleID(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	ec.Set(1, "a")
	ec.Set(2, "a")

	// The forward map must no longer resolve the stale id.
	name, err := ec.Name(1)
	require.Error(t, err)
	require.Empty(t, name)

	name, err = ec.Name(2)
	require.NoError(t, err)
	require.Equal(t, "a", name)

	id, err := ec.ID("a")
	require.NoError(t, err)
	require.Equal(t, 2, id)

	require.Equal(t, []string{"a"}, ec.SortNames())
	require.Equal(t, []int{2}, ec.SortIDs())
}

func Test_SetAll_overwrite_keepsMapsConsistent(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	ec.Set(1, "a")
	ec.SetAllNameByID(NameByID{1: "b"})
	ec.SetAllIDByName(IDByName{"b": 2})

	// After re-keying, no stale forward or reverse entry must remain.
	require.Equal(t, []string{"b"}, ec.SortNames())
	require.Equal(t, []int{2}, ec.SortIDs())

	id, err := ec.ID("a")
	require.Error(t, err)
	require.Empty(t, id)

	id, err = ec.ID("b")
	require.NoError(t, err)
	require.Equal(t, 2, id)

	name, err := ec.Name(1)
	require.Error(t, err)
	require.Empty(t, name)

	name, err = ec.Name(2)
	require.NoError(t, err)
	require.Equal(t, "b", name)
}

func Test_SetAllIDByName(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	e := IDByName{
		"first":  11,
		"second": 23,
		"third":  31,
	}

	ec.SetAllIDByName(e)

	id, err := ec.ID("second")
	require.NoError(t, err)
	require.Equal(t, 23, id)

	name, err := ec.Name(23)
	require.NoError(t, err)
	require.Equal(t, "second", name)
}

func Test_SetAllNameByID(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	e := NameByID{
		11: "first",
		23: "second",
		31: "third",
	}

	ec.SetAllNameByID(e)

	id, err := ec.ID("second")
	require.NoError(t, err)
	require.Equal(t, 23, id)

	name, err := ec.Name(23)
	require.NoError(t, err)
	require.Equal(t, "second", name)
}

func Test_SortNames(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	e := NameByID{
		1:  "delta",
		2:  "charlie",
		4:  "bravo",
		8:  "foxtrot",
		16: "echo",
		32: "alpha",
	}

	ec.SetAllNameByID(e)

	sorted := ec.SortNames()
	expected := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"}

	require.Equal(t, expected, sorted)
}

func Test_SortIDs(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	e := NameByID{
		55: "delta",
		33: "charlie",
		22: "bravo",
		66: "foxtrot",
		44: "echo",
		11: "alpha",
	}

	ec.SetAllNameByID(e)

	sorted := ec.SortIDs()
	expected := []int{11, 22, 33, 44, 55, 66}

	require.Equal(t, expected, sorted)
}

func Test_DecodeBinaryMap(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	ec.Set(1, "alpha")
	ec.Set(8, "bravo")

	s, err := ec.DecodeBinaryMap(11)
	require.Error(t, err)
	require.Equal(t, []string{"alpha", "bravo"}, s)
}

func Test_EncodeBinaryMap(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	ec.Set(1, "alpha")
	ec.Set(8, "bravo")

	v, err := ec.EncodeBinaryMap([]string{"alpha", "bravo", "charlie"})
	require.Error(t, err)
	require.Equal(t, 9, v)
}

func Test_ID_Name_sentinelErrors(t *testing.T) {
	t.Parallel()

	ec := New()

	_, err := ec.ID("missing")
	require.ErrorIs(t, err, ErrNameNotFound)

	_, err = ec.Name(99)
	require.ErrorIs(t, err, ErrIDNotFound)
}

func Test_Set_crossRekey(t *testing.T) {
	t.Parallel()

	ec := New()

	ec.Set(1, "a")
	ec.Set(2, "b")
	// Re-keying id 1 to name "b" must drop both the stale name "a" and the stale
	// id 2 in a single set call.
	ec.Set(1, "b")

	require.Equal(t, []string{"b"}, ec.SortNames())
	require.Equal(t, []int{1}, ec.SortIDs())

	_, err := ec.ID("a")
	require.ErrorIs(t, err, ErrNameNotFound)

	_, err = ec.Name(2)
	require.ErrorIs(t, err, ErrIDNotFound)

	id, err := ec.ID("b")
	require.NoError(t, err)
	require.Equal(t, 1, id)
}

func Test_Len_Has_HasID(t *testing.T) {
	t.Parallel()

	ec := New()
	require.Equal(t, 0, ec.Len())
	require.False(t, ec.Has("alpha"))
	require.False(t, ec.HasID(1))

	ec.Set(1, "alpha")
	ec.Set(2, "bravo")

	require.Equal(t, 2, ec.Len())
	require.True(t, ec.Has("alpha"))
	require.True(t, ec.HasID(2))
	require.False(t, ec.Has("charlie"))
	require.False(t, ec.HasID(3))
}

func Test_Delete(t *testing.T) {
	t.Parallel()

	ec := New()
	ec.Set(1, "alpha")
	ec.Set(2, "bravo")

	// Deleting a missing id is a no-op.
	ec.Delete(99)
	require.Equal(t, 2, ec.Len())

	ec.Delete(1)

	require.Equal(t, 1, ec.Len())
	require.False(t, ec.Has("alpha"))
	require.False(t, ec.HasID(1))

	_, err := ec.ID("alpha")
	require.ErrorIs(t, err, ErrNameNotFound)

	_, err = ec.Name(1)
	require.ErrorIs(t, err, ErrIDNotFound)

	// The unrelated entry must be untouched.
	require.True(t, ec.Has("bravo"))
}

// Test_Concurrent_RaceSafety hammers the cache from many goroutines mixing
// writers (Set, SetAllIDByName, SetAllNameByID) with readers (ID, Name,
// DecodeBinaryMap, EncodeBinaryMap, SortNames, SortIDs). It must run clean under
// -race, proving the RWMutex guards every map access. assert (not require) is
// used inside the goroutines per the testifylint go-require rule.
//
// The result correctness of reads is non-deterministic under concurrent writes,
// so the goroutines assert only the absence of races and panics.
func Test_Concurrent_RaceSafety(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	// Seed deterministic bit-flag entries so reads can both hit and miss.
	for n := range 8 {
		ec.Set(1<<n, "flag"+strconv.Itoa(n))
	}

	const (
		workers    = 16
		iterations = 200
	)

	// spawn launches the same worker body across many goroutines on a single
	// WaitGroup, keeping the test deterministic (no sleeps) and flat.
	var wg sync.WaitGroup

	spawn := func(body func(iter int)) {
		for range workers {
			wg.Go(func() {
				for i := range iterations {
					body(i)
				}
			})
		}
	}

	flag := func(i int) string { return "flag" + strconv.Itoa(i%8) }
	bit := func(i int) int { return 1 << (i % 8) }

	spawn(func(i int) { ec.Set(bit(i), flag(i)) })
	spawn(func(i int) { ec.SetAllIDByName(IDByName{flag(i): bit(i)}) })
	spawn(func(i int) { ec.SetAllNameByID(NameByID{bit(i): flag(i)}) })
	spawn(func(i int) { ec.Delete(bit(i)) })

	spawn(func(i int) {
		_, _ = ec.ID(flag(i))
		_, _ = ec.Name(bit(i))
		_ = ec.Has(flag(i))
		_ = ec.HasID(bit(i))
		_ = ec.Len()
	})
	spawn(func(i int) {
		_, _ = ec.DecodeBinaryMap(bit(i))
		_, _ = ec.EncodeBinaryMap([]string{flag(i)})
	})
	spawn(func(int) {
		assert.NotNil(t, ec.SortNames())
		assert.NotNil(t, ec.SortIDs())
	})

	wg.Wait()
}
