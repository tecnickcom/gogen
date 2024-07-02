package stringmetric

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDLDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sa   string
		sb   string
		dist int
	}{
		{
			name: "empty strings",
			sa:   "",
			sb:   "",
			dist: 0,
		},
		{
			name: "empty first string",
			sa:   "",
			sb:   "second β",
			dist: 8,
		},
		{
			name: "empty second string",
			sa:   "first α",
			sb:   "",
			dist: 7,
		},
		{
			name: "equal strings",
			sa:   "test.γ!equal",
			sb:   "test.γ!equal",
			dist: 0,
		},
		{
			name: "one substitution - last char",
			sa:   "testA",
			sb:   "testB",
			dist: 1,
		},
		{
			name: "one substitution - first char",
			sa:   "Atest",
			sb:   "Btest",
			dist: 1,
		},
		{
			name: "insertion",
			sa:   "test",
			sb:   "teBst",
			dist: 1,
		},
		{
			name: "deletion",
			sa:   "teAst",
			sb:   "test",
			dist: 1,
		},
		{
			name: "one transposition",
			sa:   "AB",
			sb:   "BA",
			dist: 1,
		},
		{
			name: "transposition + insertion",
			sa:   "a cat",
			sb:   "a abct",
			dist: 2, // "a cat" -> "a act" -> "a abct"
		},
		{
			name: "INTENTION/EXECUTION",
			sa:   "INTENTION",
			sb:   "EXECUTION",
			dist: 5,
		},
		{
			name: "unicode",
			sa:   "αβγδ",
			sb:   "αδ",
			dist: 2,
		},
		{
			name: "symbols",
			sa:   "!#$%&()*+,-./:;<=>?@[]^_{|}~",
			sb:   "\"'\\`",
			dist: 28,
		},
		{
			name: "symbols reversed",
			sa:   "\"'\\`",
			sb:   "!#$%&()*+,-./:;<=>?@[]^_{|}~",
			dist: 28,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dist := DLDistance(tt.sa, tt.sb)

			require.Equal(t, tt.dist, dist)
		})
	}
}
