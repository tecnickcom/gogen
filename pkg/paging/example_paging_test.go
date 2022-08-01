package paging_test

import (
	"fmt"

	"github.com/nexmoinc/gosrvlib/pkg/paging"
)

func ExampleNew() {
	var (
		currentPage uint = 3
		pageSize    uint = 5
		totalItems  uint = 17
	)

	// calculate new paging parameters
	p := paging.New(currentPage, pageSize, totalItems)

	fmt.Println(p)

	// Output:
	// {3 5 17 4 2 4 10}
}

func ExampleComputeOffsetAndLimit() {
	var (
		currentPage uint = 3
		pageSize    uint = 5
	)

	offset, limit := paging.ComputeOffsetAndLimit(currentPage, pageSize)

	fmt.Println(offset)
	fmt.Println(limit)

	// Output:
	// 10
	// 5
}
