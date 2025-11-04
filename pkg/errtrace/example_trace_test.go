package errtrace_test

import (
	"errors"
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/errtrace"
)

//nolint:testableexamples
func ExampleTrace() {
	err := errors.New("example error")
	testErr := errtrace.Trace(err)

	fmt.Println(testErr)
}
