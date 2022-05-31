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
