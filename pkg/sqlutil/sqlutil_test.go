package sqlutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildInClauseInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []int
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_1",
			values: []int{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_2",
			values: []int{99},
			want:   "`test_2` IN (99)",
		},
		{
			name:   "expect multiple values",
			field:  "test_3",
			values: []int{11, 13, 17},
			want:   "`test_3` IN (11,13,17)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := BuildInClauseInt(tt.field, tt.values)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBuildInClauseString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []string
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_1",
			values: []string{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_2",
			values: []string{"A"},
			want:   "`test_2` IN ('A')",
		},
		{
			name:   "expect multiple values",
			field:  "test_3",
			values: []string{"B", "C"},
			want:   "`test_3` IN ('B','C')",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := BuildInClauseString(tt.field, tt.values)
			require.Equal(t, tt.want, got)
		})
	}
}
