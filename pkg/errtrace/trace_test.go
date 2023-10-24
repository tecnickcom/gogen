package errtrace

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var errGlobalVarTest = Trace(fmt.Errorf("ERROR GLOBAL VAR"))

func errorTest() error {
	return Trace(fmt.Errorf("ERROR FUNC"))
}

func TestTrace(t *testing.T) {
	t.Parallel()

	err := errorTest()
	want := "/pkg/errtrace/trace_test.go, line: 13, function: github.com/Vonage/gosrvlib/pkg/errtrace.errorTest, error: ERROR FUNC"
	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)

	testErr := Trace(fmt.Errorf("ERROR VAR"))

	want = "/pkg/errtrace/trace_test.go, line: 23, function: github.com/Vonage/gosrvlib/pkg/errtrace.TestTrace, error: ERROR VAR"
	require.Contains(t, testErr.Error(), want, "unexpected output %v, want %v", testErr, want)

	want = "/pkg/errtrace/trace_test.go, line: 10, function: github.com/Vonage/gosrvlib/pkg/errtrace.init, error: ERROR GLOBAL VAR"
	require.Contains(t, errGlobalVarTest.Error(), want, "unexpected output %v, want %v", errGlobalVarTest, want)

	err = func() error {
		return Trace(fmt.Errorf("ERROR LAMBDA FUNC"))
	}()

	want = "/pkg/errtrace/trace_test.go, line: 32, function: github.com/Vonage/gosrvlib/pkg/errtrace.TestTrace.func1, error: ERROR LAMBDA FUNC"
	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)

	require.NoError(t, Trace(nil))
}
