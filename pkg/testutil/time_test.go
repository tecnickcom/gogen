package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceDateTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		repl string
		want string
	}{
		{
			name: "quoted RFC3339 UTC",
			src:  `{"dt":"2012-03-19T07:22:45Z"}`,
			repl: "1970-01-01T00:00:00",
			want: `{"dt":"1970-01-01T00:00:00"}`,
		},
		{
			name: "fractional seconds and numeric offset",
			src:  `{"dt":"2012-03-19T07:22:45.123456+02:00"}`,
			repl: "X",
			want: `{"dt":"X"}`,
		},
		{
			name: "multiple occurrences",
			src:  `{"a":"2012-03-19T07:22:45Z","b":"2020-01-02T03:04:05Z"}`,
			repl: "X",
			want: `{"a":"X","b":"X"}`,
		},
		{
			name: "does not cross newlines or depend on a following quote",
			src:  "start 2012-03-19T07:22:45Z line1\nline2 keepme\nend",
			repl: "X",
			want: "start X line1\nline2 keepme\nend",
		},
		{
			name: "space-separated datetime is not matched",
			src:  "2012-03-19 07:22:45",
			repl: "X",
			want: "2012-03-19 07:22:45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ReplaceDateTime(tt.src, tt.repl))
		})
	}
}

func TestReplaceUnixTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		repl string
		want string
	}{
		{
			name: "19-digit timestamp",
			src:  `{"dt":1599486799784652724}`,
			repl: "0",
			want: `{"dt":0}`,
		},
		{
			name: "multiple 19-digit timestamps",
			src:  `{"a":1599486799784652724,"b":1599486799784652725}`,
			repl: "0",
			want: `{"a":0,"b":0}`,
		},
		{
			name: "20-digit number is left untouched",
			src:  "id:12345678901234567890",
			repl: "0",
			want: "id:12345678901234567890",
		},
		{
			name: "18-digit number is left untouched",
			src:  "ts:123456789012345678",
			repl: "0",
			want: "ts:123456789012345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ReplaceUnixTimestamp(tt.src, tt.repl))
		})
	}
}
