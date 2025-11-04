package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLTE_Evaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     any
		value   any
		want    bool
		wantErr bool
	}{
		{
			name:    "false - nil value",
			ref:     5,
			value:   nil,
			want:    false,
			wantErr: false,
		},
		{
			name:    "true - smaller int",
			ref:     5,
			value:   4,
			want:    true,
			wantErr: false,
		},
		{
			name:    "true - equal int",
			ref:     5,
			value:   5,
			want:    true,
			wantErr: false,
		},
		{
			name:    "false - greater int",
			ref:     5,
			value:   6,
			want:    false,
			wantErr: false,
		},
		{
			name:    "true - smaller string",
			ref:     5,
			value:   "ciao",
			want:    true,
			wantErr: false,
		},
		{
			name:    "true - equal string",
			ref:     4,
			value:   "ciao",
			want:    true,
			wantErr: false,
		},
		{
			name:    "false - greater string",
			ref:     3,
			value:   "ciao",
			want:    false,
			wantErr: false,
		},
		{
			name:    "true - smaller string with string alias",
			ref:     5,
			value:   stringAlias("ciao"),
			want:    true,
			wantErr: false,
		},
		{
			name:    "true - equal string with string alias",
			ref:     4,
			value:   stringAlias("ciao"),
			want:    true,
			wantErr: false,
		},
		{
			name:    "false - greater string with string alias",
			ref:     3,
			value:   stringAlias("ciao"),
			want:    false,
			wantErr: false,
		},
		{
			name:    "true - smaller slice",
			ref:     5,
			value:   []int{1, 2, 3, 4},
			want:    true,
			wantErr: false,
		},
		{
			name:    "true - equal slice",
			ref:     5,
			value:   []int{1, 2, 3, 4, 5},
			want:    true,
			wantErr: false,
		},
		{
			name:    "true - greater slice",
			ref:     5,
			value:   []int{1, 2, 3, 4, 5, 6},
			want:    false,
			wantErr: false,
		},
		{
			name:    "false - unsupported type",
			ref:     5,
			value:   struct{ s string }{s: "hello"},
			want:    false,
			wantErr: false,
		},
		{
			name:    "error - invalid ref type",
			ref:     "hello",
			value:   "ciao",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eval, err := newLTE(tt.ref)

			require.Equal(t, tt.wantErr, err != nil)

			if !tt.wantErr {
				res := eval.Evaluate(tt.value)

				require.NoError(t, err)
				require.Equal(t, tt.want, res)
			}
		})
	}
}
