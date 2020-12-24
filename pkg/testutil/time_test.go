// +build unit

package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceDateTime(t *testing.T) {
	testSrc := `{"dt":"2012-03-19T07:22:45Z"}`
	testOut := ReplaceDateTime(testSrc, "<REPLACED>")
	require.Equal(t, `{"dt":"<REPLACED>"}`, testOut)
}

func TestReplaceUnixTimestamp(t *testing.T) {
	testSrc := `{"dt":1599486799784652724}`
	testOut := ReplaceUnixTimestamp(testSrc, "<REPLACED>")
	require.Equal(t, `{"dt":<REPLACED>}`, testOut)
}
