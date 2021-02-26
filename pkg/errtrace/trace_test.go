package errtrace

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// nolint:gochecknoglobals
var globalTestErr = Trace(fmt.Errorf("ERROR GLOBAL VAR"))

func errorTest() error {
	return Trace(fmt.Errorf("ERROR FUNC"))
}

func TestTrace(t *testing.T) {
	t.Parallel()

	err := errorTest()
	want := "/pkg/errtrace/trace_test.go:14 github.com/nexmoinc/gosrvlib/pkg/errtrace.errorTest: ERROR FUNC"
	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)

	var testErr = Trace(fmt.Errorf("ERROR VAR"))

	want = "/pkg/errtrace/trace_test.go:24 github.com/nexmoinc/gosrvlib/pkg/errtrace.TestTrace: ERROR VAR"

	require.Contains(t, testErr.Error(), want, "unexpected output %v, want %v", testErr, want)

	want = "/pkg/errtrace/trace_test.go:11 github.com/nexmoinc/gosrvlib/pkg/errtrace.init: ERROR GLOBAL VAR"
	require.Contains(t, globalTestErr.Error(), want, "unexpected output %v, want %v", globalTestErr, want)

	err = func() error {
		return Trace(fmt.Errorf("ERROR LAMBDA FUNC"))
	}()
	want = "/pkg/errtrace/trace_test.go:34 github.com/nexmoinc/gosrvlib/pkg/errtrace.TestTrace.func1: ERROR LAMBDA FUNC"

	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)
}
