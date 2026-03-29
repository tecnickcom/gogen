/*
Package paging computes all pagination metadata — current page, total pages,
previous/next page numbers, and SQL OFFSET/LIMIT values — from three inputs:
current page number, page size, and total item count.

# Problem

Paginating API responses and database queries requires the same arithmetic
repeatedly: clamping out-of-range page numbers, computing total pages with
correct ceiling division, deriving OFFSET for SQL, and determining safe
previous/next page links that never go below 1 or above the last page.
Getting any one of these details wrong produces broken navigation links,
off-by-one errors in SQL queries, or panics on empty result sets.

# Solution

[New] accepts the three caller-supplied values and returns a fully populated
[Paging] struct in a single call. Out-of-range inputs are clamped automatically
so callers never need to guard against zero page sizes or page numbers beyond
the last page:

	p := paging.New(currentPage, pageSize, totalItems)
	// p.Offset and p.PageSize are ready for use in a SQL LIMIT/OFFSET clause.
	// p.PreviousPage and p.NextPage are safe to embed in a JSON response.

For cases where only SQL values are needed:

	offset, limit := paging.ComputeOffsetAndLimit(currentPage, pageSize)

# Features

  - Single-call API: [New] computes every pagination field at once — no manual
    arithmetic required in application code.
  - Safe input clamping: page numbers below 1 are raised to 1; page numbers
    beyond the last page are clamped to [Paging.TotalPages]; page sizes below
    1 are raised to 1. Callers can pass raw query-string values directly.
  - Correct ceiling division: total-page calculation uses integer ceiling
    division so the last partial page is never lost.
  - Boundary-safe navigation: [Paging.PreviousPage] is always >= 1 and
    [Paging.NextPage] is always <= [Paging.TotalPages], making them safe to
    embed in API responses and hypermedia links without further validation.
  - SQL-ready offset: [Paging.Offset] is the zero-based row offset for use
    directly in a SQL OFFSET clause.
  - JSON-serialisable: all [Paging] fields carry json struct tags, so the
    struct can be returned as part of an API envelope without a separate DTO.
  - Lightweight SQL helper: [ComputeOffsetAndLimit] provides the two SQL
    values without constructing a full [Paging] struct.
  - Zero dependencies: uses only the Go standard library.

# Example

For 17 items displayed 5 per page, navigating to page 3:

	p := paging.New(3, 5, 17)
	// p.CurrentPage  == 3
	// p.PageSize     == 5
	// p.TotalItems   == 17
	// p.TotalPages   == 4
	// p.PreviousPage == 2
	// p.NextPage     == 4
	// p.Offset       == 10  (used as SQL OFFSET)

# Benefits

This package eliminates repetitive, error-prone pagination arithmetic from
application and data-access code, providing safe, consistent pagination
metadata in a single import.
*/
package paging

// Paging contains all pagination metadata computed from current page, page size, and total item count; all fields are JSON-serializable for API responses.
type Paging struct {
	// CurrentPage is the current page number starting from 1.
	CurrentPage uint `json:"page"`

	// PageSize is the maximum number of items that can be contained in a page. It is also the LIMIT in SQL queries.
	PageSize uint `json:"page_size"`

	// TotalItems is the total number of items to be paginated.
	TotalItems uint `json:"total_items"`

	// TotalPages is the total number of pages required to contain all the items.
	TotalPages uint `json:"total_pages"`

	// PreviousPage is the previous page. It is equal to 1 if we are on the first page (CurrentPage == 1).
	PreviousPage uint `json:"previous_page"`

	// NextPage is the next page. It is equal to TotalPages if we are on the last page (CurrentPage == TotalPages).
	NextPage uint `json:"next_page"`

	// Offset is the zero-based number of items before the current page.
	Offset uint `json:"offset"`
}

// New computes all pagination metadata, clamping inputs to safe ranges: pageSize and currentPage default to 1 if less, and currentPage is clamped to totalPages.
func New(currentPage, pageSize, totalItems uint) Paging {
	pageSize = minPageSize(pageSize)
	totalPages := computeTotalPages(totalItems, pageSize)
	currentPage = maxCurrentPage(minCurrentPage(currentPage), totalPages)

	return Paging{
		CurrentPage:  currentPage,
		PageSize:     pageSize,
		TotalItems:   totalItems,
		TotalPages:   totalPages,
		PreviousPage: computePreviousPage(currentPage),
		NextPage:     computeNextPage(currentPage, totalPages),
		Offset:       computeOffset(currentPage, pageSize),
	}
}

// ComputeOffsetAndLimit returns the zero-based SQL OFFSET and LIMIT (page size) for the given currentPage and pageSize, auto-clamping both to minimum values of 1.
func ComputeOffsetAndLimit(currentPage, pageSize uint) (uint, uint) {
	currentPage = minCurrentPage(currentPage)
	pageSize = minPageSize(pageSize)

	return computeOffset(currentPage, pageSize), pageSize
}

// minCurrentPage ensures that the current page is at least 1.
func minCurrentPage(currentPage uint) uint {
	if currentPage < 1 {
		return 1
	}

	return currentPage
}

// maxCurrentPage ensures that the current page does not exceed the total number of pages.
func maxCurrentPage(currentPage, totalPages uint) uint {
	if currentPage > totalPages {
		return totalPages
	}

	return currentPage
}

// minPageSize ensures that the page size is at least 1.
func minPageSize(pageSize uint) uint {
	if pageSize < 1 {
		return 1
	}

	return pageSize
}

// computeOffset computes the zero-based offset for the given current page and page size.
func computeOffset(currentPage, pageSize uint) uint {
	return pageSize * (currentPage - 1)
}

// computeTotalPages computes the total number of pages required to contain all items.
func computeTotalPages(totalItems, pageSize uint) uint {
	if totalItems <= pageSize {
		return 1
	}

	return (totalItems + pageSize - 1) / pageSize
}

// computePreviousPage computes the previous page number.
func computePreviousPage(currentPage uint) uint {
	if currentPage <= 1 {
		return 1
	}

	return currentPage - 1
}

// computeNextPage computes the next page number.
func computeNextPage(currentPage, totalPages uint) uint {
	if currentPage >= totalPages {
		return totalPages
	}

	return currentPage + 1
}
