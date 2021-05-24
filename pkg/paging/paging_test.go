package paging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	type args struct {
		currentPage uint
		pageSize    uint
		totalItems  uint
	}

	tests := []struct {
		name string
		args args
		want Paging
	}{
		{
			name: "all zero input",
			args: args{
				currentPage: 0,
				pageSize:    0,
				totalItems:  0,
			},
			want: Paging{
				CurrentPage:  1,
				PageSize:     1,
				TotalItems:   0,
				TotalPages:   1,
				PreviousPage: 1,
				NextPage:     1,
				Offset:       0,
			},
		},
		{
			name: "all ones input",
			args: args{
				currentPage: 1,
				pageSize:    1,
				totalItems:  1,
			},
			want: Paging{
				CurrentPage:  1,
				PageSize:     1,
				TotalItems:   1,
				TotalPages:   1,
				PreviousPage: 1,
				NextPage:     1,
				Offset:       0,
			},
		},
		{
			name: "current page too big",
			args: args{
				currentPage: 3,
				pageSize:    1,
				totalItems:  1,
			},
			want: Paging{
				CurrentPage:  1,
				PageSize:     1,
				TotalItems:   1,
				TotalPages:   1,
				PreviousPage: 1,
				NextPage:     1,
				Offset:       0,
			},
		},
		{
			name: "totalItems less than pageSize",
			args: args{
				currentPage: 1,
				pageSize:    5,
				totalItems:  3,
			},
			want: Paging{
				CurrentPage:  1,
				PageSize:     5,
				TotalItems:   3,
				TotalPages:   1,
				PreviousPage: 1,
				NextPage:     1,
				Offset:       0,
			},
		},
		{
			name: "totalItems greater than pageSize",
			args: args{
				currentPage: 1,
				pageSize:    5,
				totalItems:  11,
			},
			want: Paging{
				CurrentPage:  1,
				PageSize:     5,
				TotalItems:   11,
				TotalPages:   3,
				PreviousPage: 1,
				NextPage:     2,
				Offset:       0,
			},
		},
		{
			name: "currentPage < pageSize < totalItems",
			args: args{
				currentPage: 3,
				pageSize:    5,
				totalItems:  17,
			},
			want: Paging{
				CurrentPage:  3,
				PageSize:     5,
				TotalItems:   17,
				TotalPages:   4,
				PreviousPage: 2,
				NextPage:     4,
				Offset:       10,
			},
		},
		{
			name: "currentPage > pageSize > totalItems",
			args: args{
				currentPage: 17,
				pageSize:    5,
				totalItems:  3,
			},
			want: Paging{
				CurrentPage:  1,
				PageSize:     5,
				TotalItems:   3,
				TotalPages:   1,
				PreviousPage: 1,
				NextPage:     1,
				Offset:       0,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := New(tt.args.currentPage, tt.args.pageSize, tt.args.totalItems)

			require.Equal(t, got, tt.want, "New() = %v, want %v", got, tt.want)
		})
	}
}

func TestComputeOffsetAndLimit(t *testing.T) {
	t.Parallel()

	type args struct {
		currentPage uint
		pageSize    uint
	}

	tests := []struct {
		name       string
		args       args
		wantOffset uint
		wantLimit  uint
	}{
		{
			name: "all zero input",
			args: args{
				currentPage: 0,
				pageSize:    0,
			},
			wantOffset: 0,
			wantLimit:  1,
		},
		{
			name: "zero currentPage",
			args: args{
				currentPage: 0,
				pageSize:    1,
			},
			wantOffset: 0,
			wantLimit:  1,
		},
		{
			name: "zero pageSize",
			args: args{
				currentPage: 1,
				pageSize:    0,
			},
			wantOffset: 0,
			wantLimit:  1,
		},
		{
			name: "all one",
			args: args{
				currentPage: 1,
				pageSize:    1,
			},
			wantOffset: 0,
			wantLimit:  1,
		},
		{
			name: "less 1 3",
			args: args{
				currentPage: 1,
				pageSize:    3,
			},
			wantOffset: 0,
			wantLimit:  3,
		},
		{
			name: "greater 3 1",
			args: args{
				currentPage: 3,
				pageSize:    1,
			},
			wantOffset: 2,
			wantLimit:  1,
		},
		{
			name: "less 3 5",
			args: args{
				currentPage: 3,
				pageSize:    5,
			},
			wantOffset: 10,
			wantLimit:  5,
		},
		{
			name: "greater 5 3",
			args: args{
				currentPage: 5,
				pageSize:    3,
			},
			wantOffset: 12,
			wantLimit:  3,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotOffset, gotLimit := ComputeOffsetAndLimit(tt.args.currentPage, tt.args.pageSize)

			require.Equal(t, tt.wantOffset, gotOffset, "OFFSET got = %v, want %v", gotOffset, tt.wantOffset)
			require.Equal(t, tt.wantLimit, gotLimit, "LIMIT got = %v, want %v", gotLimit, tt.wantLimit)
		})
	}
}
