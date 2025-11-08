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
