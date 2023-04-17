package errtrace_test

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/errtrace"
)

//nolint:testableexamples
func ExampleTrace() {
	err := fmt.Errorf("example error")
	testErr := errtrace.Trace(err)

	fmt.Println(testErr)
}
