package logsrv

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

func TestSlogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  logutil.LogLevel
		value zerolog.Level
	}{
		{
			name:  "unknown",
			want:  logutil.LevelInfo,
			value: -32,
		},
		{
			name:  "emergency",
			want:  logutil.LevelEmergency,
			value: zerolog.PanicLevel,
		},
		{
			name:  "alert",
			want:  logutil.LevelAlert,
			value: zerolog.FatalLevel,
		},
		{
			name:  "error",
			want:  logutil.LevelError,
			value: zerolog.ErrorLevel,
		},
		{
			name:  "warning",
			want:  logutil.LevelWarning,
			value: zerolog.WarnLevel,
		},
		{
			name:  "info",
			want:  logutil.LevelInfo,
			value: zerolog.InfoLevel,
		},
		{
			name:  "debug",
			want:  logutil.LevelDebug,
			value: zerolog.DebugLevel,
		},
		{
			name:  "trace",
			want:  logutil.LevelTrace,
			value: zerolog.TraceLevel,
		},
		{
			name:  "nolevel",
			want:  logutil.LevelInfo,
			value: zerolog.NoLevel,
		},
		{
			name:  "disabled",
			want:  logutil.LevelEmergency + 1,
			value: zerolog.Disabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SlogLevel(tt.value)

			require.Equal(t, tt.want, got)
		})
	}
}
