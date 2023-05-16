package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegexp_Evaluate(t *testing.T) {
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
			name:    "error - invalid regexp",
			ref:     "[",
			value:   nil,
			want:    false,
			wantErr: true,
		},
		{
			name:    "true - matching regexp",
			ref:     "[a-d]+",
			value:   "abcdaabbccdd",
			want:    true,
			wantErr: false,
		},
		{
			name:    "false - not matching regexp",
			ref:     "^[a-d]+$",
			value:   "abcdaxabbccdd",
			want:    false,
			wantErr: false,
		},
		{
			name:    "true - matching regexp with string alias",
			ref:     "[a-d]+",
			value:   stringAlias("abcdaabbccdd"),
			want:    true,
			wantErr: false,
		},
		{
			name:    "false - not matching regexp with string alias",
			ref:     "^[a-d]+$",
			value:   stringAlias("abcdaxabbccdd"),
			want:    false,
			wantErr: false,
		},
		{
			name:    "false - struct input",
			ref:     ".*",
			value:   []struct{}{},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eval, err := newRegexp(tt.ref)

			require.Equal(t, tt.wantErr, err != nil)

			if !tt.wantErr {
				res := eval.Evaluate(tt.value)

				require.NoError(t, err)
				require.Equal(t, tt.want, res)
			}
		})
	}
}
