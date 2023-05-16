package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNot_Evaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		internal Evaluator
		ref      any
		value    any
		want     bool
	}{
		{
			name:     "true",
			internal: newEqual(1),
			ref:      1,
			value:    2,
			want:     true,
		},
		{
			name:     "false",
			internal: newEqual(1),
			ref:      1,
			value:    1,
			want:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := newNot(tt.internal).Evaluate(tt.value)

			require.Equal(t, tt.want, res)
		})
	}
}
