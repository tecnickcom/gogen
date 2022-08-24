package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasSuffix_Evaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     interface{}
		value   interface{}
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
			name:    "true - matching suffix",
			ref:     "issimo",
			value:   "buonissimo",
			want:    true,
			wantErr: false,
		},
		{
			name:    "false - not matching suffix",
			ref:     "err",
			value:   "bravissimo",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eval, err := newHasSuffix(tt.ref)

			require.Equal(t, tt.wantErr, err != nil)

			if !tt.wantErr {
				res := eval.Evaluate(tt.value)

				require.NoError(t, err)
				require.Equal(t, tt.want, res)
			}
		})
	}
}
