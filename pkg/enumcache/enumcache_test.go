package enumcache

import (
	"testing"

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

func Test_SortNames(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	ec.Set(1, "delta")
	ec.Set(2, "charlie")
	ec.Set(4, "bravo")
	ec.Set(8, "foxtrot")
	ec.Set(16, "echo")
	ec.Set(32, "alpha")

	sorted := ec.SortNames()
	expected := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"}

	require.Equal(t, expected, sorted)
}

func Test_SortIDs(t *testing.T) {
	t.Parallel()

	ec := New()
	require.NotNil(t, ec)

	ec.Set(55, "delta")
	ec.Set(33, "charlie")
	ec.Set(22, "bravo")
	ec.Set(66, "foxtrot")
	ec.Set(44, "echo")
	ec.Set(11, "alpha")

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
