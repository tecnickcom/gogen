// +build unit

package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/bootstrap"
	"github.com/nexmoinc/gosrvlib/pkg/config"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// nolint:gocognit
func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		osArgs       []string
		boostrapFunc bootstrapFunc
		wantOutput   func(t *testing.T, out string)
		wantErr      bool
	}{
		{
			name:       "call version subcommand",
			osArgs:     []string{AppName, "version"},
			wantErr:    false,
			wantOutput: matchTestVersion,
		},
		{
			name:       "fails with unknown flag",
			osArgs:     []string{AppName, "--unknown"},
			wantErr:    true,
			wantOutput: matchErrorOutput,
		},
		{
			name:       "fails with incomplete log format flag",
			osArgs:     []string{AppName, "--logFormat"},
			wantErr:    true,
			wantOutput: matchErrorOutput,
		},
		{
			name:       "fails with incomplete log level flag",
			osArgs:     []string{AppName, "--logLevel"},
			wantErr:    true,
			wantOutput: matchErrorOutput,
		},
		{
			name:    "fails with invalid flag",
			osArgs:  []string{AppName, "--logLevel", "INVALID"},
			wantErr: true,
		},
		{
			name:       "fails with incomplete config dir (short)",
			osArgs:     []string{AppName, "-c"},
			wantErr:    true,
			wantOutput: matchErrorOutput,
		}, {
			name:       "fails with incomplete config dir (long)",
			osArgs:     []string{AppName, "--configDir"},
			wantErr:    true,
			wantOutput: matchErrorOutput,
		},
		{
			name:    "fails with valid config and invalid override of log format",
			osArgs:  []string{AppName, "-c", "../../resources/test/etc/gosrvlibexample/", "--logFormat", "invalid"},
			wantErr: true,
		},
		{
			name:    "fails with valid config and invalid override of log level",
			osArgs:  []string{AppName, "-c", "../../resources/test/etc/gosrvlibexample/", "--logLevel", "invalid"},
			wantErr: true,
		},
		{
			name:    "attempts bootstrap with invalid configuration",
			osArgs:  []string{AppName, "-c", "../../resources/test/etc/invalid/"},
			wantErr: true,
		},
		{
			name:   "bootstrap with valid configuration",
			osArgs: []string{AppName, "-c", "../../resources/test/etc/gosrvlibexample/"},
			boostrapFunc: func(bindFn bootstrap.BindFunc, opts ...bootstrap.Option) error {
				return nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config.Reset()

			oldOsArgs := os.Args
			defer func() { os.Args = oldOsArgs }()
			os.Args = tt.osArgs

			bootstrapFunc := bootstrap.Bootstrap
			if tt.boostrapFunc != nil {
				bootstrapFunc = tt.boostrapFunc
			}

			// execute the main function
			var out string
			cmd, err := New("0.0.0-test", "0", bootstrapFunc)
			if err == nil {
				require.NotNil(t, cmd)

				out = testutil.CaptureOutput(t, func() {
					err = cmd.Execute()
				})

				if tt.wantOutput != nil {
					tt.wantOutput(t, out)
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func matchErrorOutput(t *testing.T, out string) {
	if strings.HasPrefix(out, "Error:") {
		return
	}
	t.Errorf("An error message was expected")
}

func matchTestVersion(t *testing.T, out string) {
	if strings.HasPrefix(out, "0.0.0-test") {
		return
	}
	t.Errorf("A version number was expected")
}
