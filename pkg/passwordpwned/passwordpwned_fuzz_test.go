package passwordpwned

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzMatchCount pins the no-panic and sane-output properties of the range
// response parser for arbitrary (hostile) bodies and suffixes.
func FuzzMatchCount(f *testing.F) {
	f.Add([]byte("46FA9F7FDDB8AB4A5BB8295A47E3929171E:2\r\n"), "46FA9F7FDDB8AB4A5BB8295A47E3929171E")
	f.Add([]byte(""), "")
	f.Add([]byte("46FA9F7FDDB8AB4A5BB8295A47E3929171E"), "46FA9F7FDDB8AB4A5BB8295A47E3929171E")
	f.Add([]byte("46FA9F7FDDB8AB4A5BB8295A47E3929171E;2\r\n"), "46FA9F7FDDB8AB4A5BB8295A47E3929171E")
	f.Add([]byte(":0123456789"), ":")

	f.Fuzz(func(t *testing.T, data []byte, suffix string) {
		count, err := matchCount(data, suffix) // must never panic
		if err != nil {
			require.Equal(t, 0, count, "count must be zero on error")
		}

		require.GreaterOrEqual(t, count, 0, "count must never be negative")
	})
}

// FuzzReadRangeBody pins the no-panic property of the read/validate/normalize
// path and its size-cap and shape invariants for arbitrary bodies and limits.
func FuzzReadRangeBody(f *testing.F) {
	f.Add([]byte("46FA9F7FDDB8AB4A5BB8295A47E3929171E:2\r\n"), int64(1024))
	f.Add([]byte("46fa9f7fddb8ab4a5bb8295a47e3929171e:2\r\n"), int64(1024))
	f.Add([]byte(""), int64(1024))
	f.Add([]byte("<html>error</html>"), int64(1024))
	f.Add([]byte("46FA9F7FDDB8AB4A5BB8295A47E3929171E:2\r\n"), int64(4))
	f.Add([]byte("A"), int64(-1))

	f.Fuzz(func(t *testing.T, body []byte, limit int64) {
		data, err := readRangeBody(bytes.NewReader(body), limit) // must never panic
		if err != nil {
			require.Nil(t, data, "data must be nil on error")

			return
		}

		require.True(t, validRangeStart(data), "successful reads must be structurally valid range data")
		require.False(t, bytes.ContainsAny(data, "abcdef"), "successful reads must be uppercase-normalized")

		if limit >= 0 {
			require.LessOrEqual(t, int64(len(data)), limit, "successful reads must respect the size limit")
		}
	})
}
