// +build unit

package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/gosrvlibexample/gosrvlibexample/internal/cli"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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

func TestMainCliError(t *testing.T) {
	oldLogFatal := logging.LogFatal
	defer func() { logging.LogFatal = oldLogFatal }()
	logging.LogFatal = zap.L().Panic
	os.Args = []string{cli.AppName, "--INVALID"}
	require.Panics(t, main, "Expected to fail because of invalid argument name")
}

func TestMainCliExecuteError(t *testing.T) {
	oldLogFatal := logging.LogFatal
	defer func() { logging.LogFatal = oldLogFatal }()
	logging.LogFatal = zap.L().Panic
	os.Args = []string{cli.AppName, "--logLevel=INVALID"}
	require.Panics(t, main, "Expected to fail because of invalid argument value")
}
