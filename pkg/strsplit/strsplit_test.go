package strsplit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChunk(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    string
		size int
		n    int
		want []string
	}{
		{
			name: "size zero",
			s:    "hello",
			size: 0,
			n:    -1,
			want: nil,
		},
		{
			name: "n zero",
			s:    "hello",
			size: 10,
			n:    0,
			want: nil,
		},
		{
			name: "empty string",
			s:    "",
			size: 5,
			n:    -1,
			want: []string{},
		},
		{
			name: "newline separator",
			s:    "hello\nworld\nverylongword",
			size: 10,
			n:    -1,
			want: []string{"hello", "world", "verylongwo", "rd"},
		},
		{
			name: "n smaller",
			s:    "hello\nworld\nbella\nciao",
			size: 7,
			n:    2,
			want: []string{"hello", "world"},
		},
		{
			name: "n exact",
			s:    "hello\nworld\nbella\nciao",
			size: 7,
			n:    4,
			want: []string{"hello", "world", "bella", "ciao"},
		},
		{
			name: "n bigger",
			s:    "hello\nworld\nbella\nciao",
			size: 7,
			n:    5,
			want: []string{"hello", "world", "bella", "ciao"},
		},
		{
			name: "n large",
			s:    "hello\nworld\nbella\nciao",
			size: 7,
			n:    10,
			want: []string{"hello", "world", "bella", "ciao"},
		},
		{
			name: "n with reminder",
			s:    "helloworld\nbellaciao",
			size: 5,
			n:    3,
			want: []string{"hello", "world", "bella"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Chunk(tt.s, tt.size, tt.n)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestChunkLine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    string
		size int
		n    int
		want []string
	}{
		{
			name: "size zero",
			s:    "hello",
			size: 0,
			n:    -1,
			want: nil,
		},
		{
			name: "n zero",
			s:    "hello",
			size: 10,
			n:    0,
			want: nil,
		},
		{
			name: "empty string",
			s:    "",
			size: 5,
			n:    -1,
			want: []string{},
		},
		{
			name: "exact division",
			s:    "helloworld",
			size: 5,
			n:    -1,
			want: []string{"hello", "world"},
		},
		{
			name: "remainder chunk",
			s:    "helloworld!",
			size: 5,
			n:    -1,
			want: []string{"hello", "world", "!"},
		},
		{
			name: "single character chunks",
			s:    "abc",
			size: 1,
			n:    -1,
			want: []string{"a", "b", "c"},
		},
		{
			name: "size larger than string",
			s:    "hi",
			size: 10,
			n:    -1,
			want: []string{"hi"},
		},
		{
			name: "split on space separator",
			s:    "hello world test",
			size: 10,
			n:    -1,
			want: []string{"hello", "world test"},
		},
		{
			name: "split on tab separator",
			s:    "hello\tworld",
			size: 10,
			n:    -1,
			want: []string{"hello", "world"},
		},
		{
			name: "split on newline separator",
			s:    "hello\nworld",
			size: 10,
			n:    -1,
			want: []string{"hello", "world"},
		},
		{
			name: "split on punctuation",
			s:    "hello,world",
			size: 10,
			n:    -1,
			want: []string{"hello,", "world"},
		},
		{
			name: "whitespace trimming",
			s:    "  hello  ",
			size: 10,
			n:    -1,
			want: []string{"hello"},
		},
		{
			name: "size one with spaces",
			s:    "a b c d",
			size: 1,
			n:    -1,
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "unicode split",
			s:    "Hello 世界 Hello 世界 Hello 世界", //nolint:gosmopolitan
			size: 7,
			n:    -1,
			want: []string{"Hello", "世界", "Hello", "世界", "Hello", "世界"}, //nolint:gosmopolitan
		},
		{
			name: "n smaller",
			s:    "Hello 世界 Hello 世界 Hello 世界", //nolint:gosmopolitan
			size: 7,
			n:    3,
			want: []string{"Hello", "世界", "Hello"}, //nolint:gosmopolitan
		},
		{
			name: "n exact",
			s:    "Hello 世界 Hello 世界 Hello 世界", //nolint:gosmopolitan
			size: 7,
			n:    6,
			want: []string{"Hello", "世界", "Hello", "世界", "Hello", "世界"}, //nolint:gosmopolitan
		},
		{
			name: "n bigger",
			s:    "hello\nworld\nbella\nciao",
			size: 7,
			n:    5,
			want: []string{"hello", "world", "bella", "ciao"},
		},
		{
			name: "n smaller",
			s:    "helloworldbellaciao",
			size: 5,
			n:    2,
			want: []string{"hello", "world"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ChunkLine(tt.s, tt.size, tt.n)
			require.Equal(t, tt.want, got)
		})
	}
}
