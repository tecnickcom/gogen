// +build unit

package testutil

import (
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaptureOutput(t *testing.T) {
	testFn := func() {
		log.Printf("test output capture")
	}

	output := CaptureOutput(t, testFn)
	output = strings.TrimSuffix(output, "\n")
	require.Regexp(t, `^[0-9]{4}(/[0-9]{2}){2}\s([0-9]{2}:){2}[0-9]{2}\stest\soutput\scapture$`, output)
}
