package enumcache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnumCache(t *testing.T) {
	t.Parallel()

	ec := MakeEnumCache()
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
