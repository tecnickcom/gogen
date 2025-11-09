package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/gogenexampleowner/gogenexample/internal/cli"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/testutil"
)

//nolint:paralleltest
func TestProgramVersion(t *testing.T) {
	os.Args = []string{cli.AppName, "version"}
	out := testutil.CaptureOutput(t, func() {
		main()
	})

	match, err := regexp.MatchString("^[\\d]+\\.[\\d]+\\.[\\d]+[\\s]*$", out)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !match {
		t.Errorf("The expected version has not been returned")
	}
}

//nolint:paralleltest
func TestMainCliError(t *testing.T) {
	oldExitFn := exitFn

	defer func() { exitFn = oldExitFn }()

	exitFn = func(v int) { panic(v) }

	os.Args = []string{cli.AppName, "--INVALID"}

	require.Panics(t, main, "Expected to fail because of invalid argument name")
}

//nolint:paralleltest
func TestMainCliExecuteError(t *testing.T) {
	oldExitFn := exitFn

	defer func() { exitFn = oldExitFn }()

	exitFn = func(v int) { panic(v) }

	os.Args = []string{cli.AppName, "--logLevel=INVALID"}

	require.Panics(t, main, "Expected to fail because of invalid argument value")
}
