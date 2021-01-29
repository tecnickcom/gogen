package cli

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gosrvlibexample/gosrvlibexample/internal/metrics"
	"github.com/nexmoinc/gosrvlib/pkg/bootstrap"
	"github.com/nexmoinc/gosrvlib/pkg/httputil/jsendx"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// nolint:gocognit
func Test_bind(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *appConfig
		preBindAddr    string
		pingAddr       string
		wantErr        bool
		wantTimeoutErr bool
	}{
		{
			name: "fails with monitor port already bound",
			cfg: &appConfig{
				Enabled:           false,
				MonitoringAddress: ":30040",
				PublicAddress:     ":30041",
			},
			preBindAddr:    ":30040",
			wantErr:        true,
			wantTimeoutErr: false,
		},
		{
			name: "fails with bad ipify address",
			cfg: &appConfig{
				Enabled:           true,
				MonitoringAddress: ":30040",
				PublicAddress:     ":30041",
				Ipify: ipifyConfig{
					Address: "test.ipify.url.invalid\u007F",
					Timeout: 1,
				},
			},
			wantErr:        true,
			wantTimeoutErr: false,
		},
		{
			name: "fails with service port already bound",
			cfg: &appConfig{
				Enabled:           false,
				MonitoringAddress: ":30040",
				PublicAddress:     ":30041",
			},
			preBindAddr:    ":30041",
			wantErr:        true,
			wantTimeoutErr: false,
		},
		{
			name: "succeed with separate ports",
			cfg: &appConfig{
				Enabled:           false,
				MonitoringAddress: ":30040",
				PublicAddress:     ":30041",
			},
			wantErr: false,
		},
		{
			name: "succeed with same ports",
			cfg: &appConfig{
				Enabled:           false,
				MonitoringAddress: ":30040",
				PublicAddress:     ":30040",
			},
			wantErr: false,
		},
		{
			name: "succeed with enabled flag set",
			cfg: &appConfig{
				Enabled:           true,
				MonitoringAddress: ":30040",
				PublicAddress:     ":30040",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preBindAddr != "" {
				l, err := net.Listen("tcp", tt.preBindAddr)
				require.NoError(t, err)
				defer func() { _ = l.Close() }()
			}

			mtr := metrics.New()

			testBindFn := bind(
				tt.cfg,
				&jsendx.AppInfo{
					ProgramName:    "test",
					ProgramVersion: "0.0.0",
					ProgramRelease: "0",
				},
				mtr,
			)

			testCtx, cancel := context.WithTimeout(testutil.Context(), 1*time.Second)
			defer cancel()

			testBootstrapOpts := []bootstrap.Option{
				bootstrap.WithContext(testCtx),
				bootstrap.WithLogger(logging.FromContext(testCtx)),
			}
			err := bootstrap.Bootstrap(testBindFn, testBootstrapOpts...)
			if tt.wantErr {
				require.Error(t, err, "bind() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantTimeoutErr {
				require.True(t, errors.Is(err, context.DeadlineExceeded),
					"bind() error = %v, wantErr %v", err, context.DeadlineExceeded)
			} else {
				require.False(t, errors.Is(err, context.DeadlineExceeded), "bind() unexpected timeout error")
			}
		})
	}
}
