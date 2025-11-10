package logutil

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyslogToSlogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value   string
		want    slog.Level
		wantErr bool
	}{
		{
			value:   "invalid",
			want:    LevelDebug,
			wantErr: true,
		},
		{
			value: "0",
			want:  LevelEmergency,
		},
		{
			value: "emerg",
			want:  LevelEmergency,
		},
		{
			value: "emergency",
			want:  LevelEmergency,
		},
		{
			value: "EMERGENCY",
			want:  LevelEmergency,
		},
		{
			value: "1",
			want:  LevelAlert,
		},
		{
			value: "alert",
			want:  LevelAlert,
		},
		{
			value: "2",
			want:  LevelCritical,
		},
		{
			value: "crit",
			want:  LevelCritical,
		},
		{
			value: "critical",
			want:  LevelCritical,
		},
		{
			value: "3",
			want:  LevelError,
		},
		{
			value: "err",
			want:  LevelError,
		},
		{
			value: "error",
			want:  LevelError,
		},
		{
			value: "4",
			want:  LevelWarning,
		},
		{
			value: "warn",
			want:  LevelWarning,
		},
		{
			value: "warning",
			want:  LevelWarning,
		},
		{
			value: "5",
			want:  LevelNotice,
		},
		{
			value: "notice",
			want:  LevelNotice,
		},
		{
			value: "6",
			want:  LevelInfo,
		},
		{
			value: "info",
			want:  LevelInfo,
		},
		{
			value: "INFO",
			want:  LevelInfo,
		},
		{
			value: "7",
			want:  LevelDebug,
		},
		{
			value: "debug",
			want:  LevelDebug,
		},
		{
			value: "DEBUG",
			want:  LevelDebug,
		},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			got, err := LevelStrToSlog(tt.value)
			if tt.wantErr {
				require.Error(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}
