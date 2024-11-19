package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceDateTime(t *testing.T) {
	t.Parallel()

	testSrc := `{"dt":"2012-03-19T07:22:45Z"}`
	testOut := ReplaceDateTime(testSrc, "1970-01-01T00:00:00")
	require.JSONEq(t, `{"dt":"1970-01-01T00:00:00"}`, testOut)
}

func TestReplaceUnixTimestamp(t *testing.T) {
	t.Parallel()

	testSrc := `{"dt":1599486799784652724}`
	testOut := ReplaceUnixTimestamp(testSrc, "0")
	require.JSONEq(t, `{"dt":0}`, testOut)
}
