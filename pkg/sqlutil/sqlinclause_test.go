package sqlutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// nolint:dupl
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
			field:  "test_in_1",
			values: []string{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_in_2",
			values: []string{"A"},
			want:   "`test_in_2` IN ('A')",
		},
		{
			name:   "expect multiple values",
			field:  "test_in_3",
			values: []string{"B", "C"},
			want:   "`test_in_3` IN ('B','C')",
		},
		{
			name:   "composed field name",
			field:  "schema_in_4.table_in_4",
			values: []string{"D", "E", "F"},
			want:   "`schema_in_4`.`table_in_4` IN ('D','E','F')",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := defaultSQLUtil()
			got := c.BuildInClauseString(tt.field, tt.values)

			require.Equal(t, tt.want, got)
		})
	}
}

// nolint:dupl
func TestBuildNotInClauseString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []string
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_notin_1",
			values: []string{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_notin_2",
			values: []string{"AA"},
			want:   "`test_notin_2` NOT IN ('AA')",
		},
		{
			name:   "expect multiple values",
			field:  "test_notin_3",
			values: []string{"BB", "CC"},
			want:   "`test_notin_3` NOT IN ('BB','CC')",
		},
		{
			name:   "composed field name",
			field:  "schema_notin_4.table_notin_4",
			values: []string{"DD", "EE", "FF"},
			want:   "`schema_notin_4`.`table_notin_4` NOT IN ('DD','EE','FF')",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := defaultSQLUtil()
			got := c.BuildNotInClauseString(tt.field, tt.values)

			require.Equal(t, tt.want, got)
		})
	}
}

// nolint:dupl
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
			field:  "test_in_1",
			values: []int{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_in_2",
			values: []int{199},
			want:   "`test_in_2` IN (199)",
		},
		{
			name:   "expect multiple values",
			field:  "test_in_3",
			values: []int{111, -113},
			want:   "`test_in_3` IN (111,-113)",
		},
		{
			name:   "composed field name",
			field:  "schema_in_4.table_in_4",
			values: []int{111, -113, 117},
			want:   "`schema_in_4`.`table_in_4` IN (111,-113,117)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := defaultSQLUtil()
			got := c.BuildInClauseInt(tt.field, tt.values)

			require.Equal(t, tt.want, got)
		})
	}
}

// nolint:dupl
func TestBuildNotInClauseInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []int
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_notin_1",
			values: []int{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_notin_2",
			values: []int{299},
			want:   "`test_notin_2` NOT IN (299)",
		},
		{
			name:   "expect multiple values",
			field:  "test_notin_3",
			values: []int{211, -213},
			want:   "`test_notin_3` NOT IN (211,-213)",
		},
		{
			name:   "composed field name",
			field:  "schema_notin_4.table_notin_4",
			values: []int{211, -213, 217},
			want:   "`schema_notin_4`.`table_notin_4` NOT IN (211,-213,217)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := defaultSQLUtil()
			got := c.BuildNotInClauseInt(tt.field, tt.values)

			require.Equal(t, tt.want, got)
		})
	}
}

// nolint:dupl
func TestBuildInClauseUint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []uint64
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_in_1",
			values: []uint64{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_in_2",
			values: []uint64{399},
			want:   "`test_in_2` IN (399)",
		},
		{
			name:   "expect multiple values",
			field:  "test_in_3",
			values: []uint64{311, 313},
			want:   "`test_in_3` IN (311,313)",
		},
		{
			name:   "composed field name",
			field:  "schema_in_4.table_in_4",
			values: []uint64{311, 313, 317},
			want:   "`schema_in_4`.`table_in_4` IN (311,313,317)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := defaultSQLUtil()
			got := c.BuildInClauseUint(tt.field, tt.values)

			require.Equal(t, tt.want, got)
		})
	}
}

// nolint:dupl
func TestBuildNotInClauseUint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []uint64
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_notin_1",
			values: []uint64{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_notin_2",
			values: []uint64{499},
			want:   "`test_notin_2` NOT IN (499)",
		},
		{
			name:   "expect multiple values",
			field:  "test_notin_3",
			values: []uint64{411, 413},
			want:   "`test_notin_3` NOT IN (411,413)",
		},
		{
			name:   "composed field name",
			field:  "schema_notin_4.table_notin_4",
			values: []uint64{411, 413, 417},
			want:   "`schema_notin_4`.`table_notin_4` NOT IN (411,413,417)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := defaultSQLUtil()
			got := c.BuildNotInClauseUint(tt.field, tt.values)

			require.Equal(t, tt.want, got)
		})
	}
}
