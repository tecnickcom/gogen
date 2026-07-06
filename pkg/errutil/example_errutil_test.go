package errutil_test

import (
	"errors"
	"fmt"

	"github.com/tecnickcom/gogen/pkg/errutil"
)

//nolint:testableexamples
func ExampleTrace() {
	err := errors.New("example error")
	testErr := errutil.Trace(err)

	fmt.Println(testErr)
}

func ExampleJoinFnError() {
	err := errors.New("original error")

	fn := func() error {
		return errors.New("function error")
	}

	errutil.JoinFnError(&err, fn)

	fmt.Println(err)

	// Output:
	// original error
	// function error
}

func ExampleJoinFnError_defer() {
	// JoinFnError is designed for defer/cleanup: err must be a named return
	// value so the deferred call can amend it after the function body runs.
	process := func() (err error) { //nolint:nonamedreturns // JoinFnError amends the return via its address
		closer := func() error {
			return errors.New("close failed")
		}
		defer errutil.JoinFnError(&err, closer)

		return errors.New("primary failure")
	}

	fmt.Println(process())

	// Output:
	// primary failure
	// close failed
}
