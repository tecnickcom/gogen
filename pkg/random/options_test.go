package random

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithByteToCharMap(t *testing.T) {
	t.Parallel()

	want := []byte(chrMapDefault)
	c := &Rnd{}

	WithByteToCharMap(want)(c)
	require.Equal(t, want, c.chrMap)

	WithByteToCharMap(nil)(c)
	require.Equal(t, want, c.chrMap)

	WithByteToCharMap([]byte{})(c)
	require.Equal(t, want, c.chrMap)

	WithByteToCharMap([]byte("0123456789abcdefx"))(c)
	require.Len(t, c.chrMap, 17)

	longMap := make([]byte, chrMapMaxLen+1)

	WithByteToCharMap(longMap)(c)
	require.Len(t, c.chrMap, chrMapMaxLen)
}
