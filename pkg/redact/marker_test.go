package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// expectedRedaction rewrites the `***` placeholders used in the test tables into
// whatever RedactionMarker actually is, so the expectations stay correct if the
// default marker ever changes.
func expectedRedaction(s string) string {
	return strings.ReplaceAll(s, "***", RedactionMarker)
}

func TestRedactionMarkerValue(t *testing.T) {
	t.Parallel()

	require.Equal(t, "***", RedactionMarker)
}

func TestMarkerAt(t *testing.T) {
	t.Parallel()

	re := Default()

	require.True(t, re.markerAt([]byte("***tail"), 0))
	require.True(t, re.markerAt([]byte("head***tail"), 4))
	require.False(t, re.markerAt([]byte("head***"), 0))

	// A marker that would run past the end of the input is not a match.
	require.False(t, re.markerAt([]byte("**"), 0))
}

func TestMarkerEndsAt(t *testing.T) {
	t.Parallel()

	re := Default()

	require.True(t, re.markerEndsAt([]byte("***"), 2))
	require.True(t, re.markerEndsAt([]byte("head***"), 6))
	require.False(t, re.markerEndsAt([]byte("head***"), 5))

	// A marker that would start before the input does not match.
	require.False(t, re.markerEndsAt([]byte("**"), 1))
}
