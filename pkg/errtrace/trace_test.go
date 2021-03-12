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
	want := "/pkg/errtrace/trace_test.go, line: 14, function: github.com/nexmoinc/gosrvlib/pkg/errtrace.errorTest, error: ERROR FUNC"
	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)

	var testErr = Trace(fmt.Errorf("ERROR VAR"))

	want = "/pkg/errtrace/trace_test.go, line: 24, function: github.com/nexmoinc/gosrvlib/pkg/errtrace.TestTrace, error: ERROR VAR"
	require.Contains(t, testErr.Error(), want, "unexpected output %v, want %v", testErr, want)

	want = "/pkg/errtrace/trace_test.go, line: 11, function: github.com/nexmoinc/gosrvlib/pkg/errtrace.init, error: ERROR GLOBAL VAR"
	require.Contains(t, globalTestErr.Error(), want, "unexpected output %v, want %v", globalTestErr, want)

	err = func() error {
		return Trace(fmt.Errorf("ERROR LAMBDA FUNC"))
	}()

	want = "/pkg/errtrace/trace_test.go, line: 33, function: github.com/nexmoinc/gosrvlib/pkg/errtrace.TestTrace.func1, error: ERROR LAMBDA FUNC"
	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)
}
