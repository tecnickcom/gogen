package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContains_Evaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     any
		value   any
		want    bool
		wantErr bool
	}{
		{
			name:    "error - no string ref",
			ref:     5,
			value:   nil,
			want:    false,
			wantErr: true,
		},
		{
			name:    "false - nil value",
			ref:     "start",
			value:   nil,
			want:    false,
			wantErr: false,
		},
		{
			name:    "false - non-string value",
			ref:     "start",
			value:   5,
			want:    false,
			wantErr: false,
		},
		{
			name:    "true - contains sub-string",
			ref:     "oniss",
			value:   "buonissimo",
			want:    true,
			wantErr: false,
		},
		{
			name:    "false - do not contains sub-string",
			ref:     "three",
			value:   "bravissimo",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eval, err := newContains(tt.ref)

			require.Equal(t, tt.wantErr, err != nil)

			if !tt.wantErr {
				res := eval.Evaluate(tt.value)

				require.NoError(t, err)
				require.Equal(t, tt.want, res)
			}
		})
	}
}
