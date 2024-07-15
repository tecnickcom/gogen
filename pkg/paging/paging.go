/*
Package paging provides utilities to handle pagination.

The Paging struct represents the pagination information and is automatically
populated by the New function.
*/
package paging

// Paging contains the paging information.
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

// New returns a new paging information instance.
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

// ComputeOffsetAndLimit computes the OFFSET (zero based) and LIMIT values to be used with SQL queries.
func ComputeOffsetAndLimit(currentPage, pageSize uint) (uint, uint) {
	currentPage = minCurrentPage(currentPage)
	pageSize = minPageSize(pageSize)

	return computeOffset(currentPage, pageSize), pageSize
}

func minCurrentPage(currentPage uint) uint {
	if currentPage < 1 {
		return 1
	}

	return currentPage
}

func maxCurrentPage(currentPage, totalPages uint) uint {
	if currentPage > totalPages {
		return totalPages
	}

	return currentPage
}

func minPageSize(pageSize uint) uint {
	if pageSize < 1 {
		return 1
	}

	return pageSize
}

func computeOffset(currentPage, pageSize uint) uint {
	return pageSize * (currentPage - 1)
}

func computeTotalPages(totalItems, pageSize uint) uint {
	if totalItems <= pageSize {
		return 1
	}

	return (totalItems + pageSize - 1) / pageSize
}

func computePreviousPage(currentPage uint) uint {
	if currentPage <= 1 {
		return 1
	}

	return currentPage - 1
}

func computeNextPage(currentPage, totalPages uint) uint {
	if currentPage >= totalPages {
		return totalPages
	}

	return currentPage + 1
}
