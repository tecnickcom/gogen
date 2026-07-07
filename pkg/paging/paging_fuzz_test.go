package paging

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzNew(f *testing.F) {
	seeds := []struct {
		currentPage uint
		pageSize    uint
		totalItems  uint
	}{
		{0, 0, 0},
		{1, 1, 1},
		{3, 5, 17},
		{17, 5, 3},
		{2, 10, 20},
		{1, 10, math.MaxUint},
		{math.MaxUint, 10, math.MaxUint},
	}
	for _, s := range seeds {
		f.Add(s.currentPage, s.pageSize, s.totalItems)
	}

	f.Fuzz(func(t *testing.T, currentPage, pageSize, totalItems uint) {
		p := New(currentPage, pageSize, totalItems)

		// Clamping invariants.
		require.GreaterOrEqual(t, p.PageSize, uint(1), "page size must be >= 1")
		require.GreaterOrEqual(t, p.TotalPages, uint(1), "total pages must be >= 1")
		require.GreaterOrEqual(t, p.CurrentPage, uint(1), "current page must be >= 1")
		require.LessOrEqual(t, p.CurrentPage, p.TotalPages, "current page must be <= total pages")
		require.Equal(t, totalItems, p.TotalItems, "total items must pass through unchanged")

		// Navigation invariants.
		require.GreaterOrEqual(t, p.PreviousPage, uint(1), "previous page must be >= 1")
		require.LessOrEqual(t, p.PreviousPage, p.CurrentPage, "previous page must be <= current page")
		require.GreaterOrEqual(t, p.NextPage, p.CurrentPage, "next page must be >= current page")
		require.LessOrEqual(t, p.NextPage, p.TotalPages, "next page must be <= total pages")

		// Navigation flags must agree with the boundaries and page numbers.
		require.Equal(t, p.CurrentPage > 1, p.HasPreviousPage, "HasPreviousPage must match CurrentPage > 1")
		require.Equal(t, p.CurrentPage < p.TotalPages, p.HasNextPage, "HasNextPage must match CurrentPage < TotalPages")
		require.Equal(t, p.HasPreviousPage, p.PreviousPage < p.CurrentPage, "PreviousPage moves only when HasPreviousPage")
		require.Equal(t, p.HasNextPage, p.NextPage > p.CurrentPage, "NextPage moves only when HasNextPage")

		// In New, currentPage is clamped to the last page, so the offset never
		// overflows and always addresses a real row inside the data.
		require.Equal(t, p.PageSize*(p.CurrentPage-1), p.Offset, "offset must equal pageSize*(page-1)")

		if totalItems > 0 {
			require.Less(t, p.Offset, totalItems, "offset must point inside the data")
		}
	})
}

func FuzzComputeOffsetAndLimit(f *testing.F) {
	seeds := []struct {
		currentPage uint
		pageSize    uint
	}{
		{0, 0},
		{1, 1},
		{3, 5},
		{5, 3},
		{190000000000000000, 100},
		{math.MaxUint, 2},
	}
	for _, s := range seeds {
		f.Add(s.currentPage, s.pageSize)
	}

	f.Fuzz(func(t *testing.T, currentPage, pageSize uint) {
		offset, limit := ComputeOffsetAndLimit(currentPage, pageSize)

		require.GreaterOrEqual(t, limit, uint(1), "limit must be >= 1")

		page := max(currentPage, 1)
		if offset != math.MaxUint {
			require.Equal(t, limit*(page-1), offset, "offset must equal limit*(page-1) unless clamped")
		}
	})
}
