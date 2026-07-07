package testutil

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest
func TestCaptureOutput(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	origLogWriter := log.Writer()

	output := CaptureOutput(t, func() {
		fmt.Fprintln(os.Stdout, "to stdout")
		fmt.Fprintln(os.Stderr, "to stderr")
		log.Printf("to logger")
	})

	// All three streams are captured.
	require.Contains(t, output, "to stdout")
	require.Contains(t, output, "to stderr")
	require.Regexp(t, `[0-9]{4}(/[0-9]{2}){2}\s([0-9]{2}:){2}[0-9]{2}\sto logger`, output)

	// Global state is restored after capture.
	require.Same(t, origStdout, os.Stdout)
	require.Same(t, origStderr, os.Stderr)
	require.Equal(t, origLogWriter, log.Writer())
}
