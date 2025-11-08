package errutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errGlobalVarTest = Trace(errors.New("ERROR GLOBAL VAR"))

func errorTest() error {
	return Trace(errors.New("ERROR FUNC"))
}

func TestTrace(t *testing.T) {
	t.Parallel()

	err := errorTest()
	want := "/pkg/errutil/trace_test.go, line: 13, function: github.com/tecnickcom/gogen/pkg/errutil.errorTest, error: ERROR FUNC"
	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)

	testErr := Trace(errors.New("ERROR VAR"))

	want = "/pkg/errutil/trace_test.go, line: 23, function: github.com/tecnickcom/gogen/pkg/errutil.TestTrace, error: ERROR VAR"
	require.Contains(t, testErr.Error(), want, "unexpected output %v, want %v", testErr, want)

	want = "/pkg/errutil/trace_test.go, line: 10, function: github.com/tecnickcom/gogen/pkg/errutil.init, error: ERROR GLOBAL VAR"
	require.Contains(t, errGlobalVarTest.Error(), want, "unexpected output %v, want %v", errGlobalVarTest, want)

	err = func() error {
		return Trace(errors.New("ERROR LAMBDA FUNC"))
	}()

	want = "/pkg/errutil/trace_test.go, line: 32, function: github.com/tecnickcom/gogen/pkg/errutil.TestTrace.func1, error: ERROR LAMBDA FUNC"
	require.Contains(t, err.Error(), want, "unexpected output %v, want %v", err, want)

	require.NoError(t, Trace(nil))
}
