package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    zapcore.Level
		wantErr bool
	}{
		{
			name:    "fails and fallback with invalid string",
			value:   "invalid",
			want:    zapcore.DebugLevel,
			wantErr: true,
		},
		{
			name:    "succeed with debug",
			value:   "debug",
			want:    zapcore.DebugLevel,
			wantErr: false,
		},
		{
			name:    "succeed with info",
			value:   "info",
			want:    zapcore.InfoLevel,
			wantErr: false,
		},
		{
			name:    "succeed with 'warn'",
			value:   "warn",
			want:    zapcore.WarnLevel,
			wantErr: false,
		},
		{
			name:    "succeed with warning",
			value:   "warning",
			want:    zapcore.WarnLevel,
			wantErr: false,
		},
		{
			name:    "succeed with notice",
			value:   "notice",
			want:    zapcore.InfoLevel,
			wantErr: false,
		},
		{
			name:    "succeed with err",
			value:   "err",
			want:    zapcore.ErrorLevel,
			wantErr: false,
		},
		{
			name:    "succeed with error",
			value:   "error",
			want:    zapcore.ErrorLevel,
			wantErr: false,
		},
		{
			name:    "succeed with crit",
			value:   "crit",
			want:    zapcore.FatalLevel,
			wantErr: false,
		},
		{
			name:    "succeed with critical",
			value:   "critical",
			want:    zapcore.FatalLevel,
			wantErr: false,
		},
		{
			name:    "succeed with emergency",
			value:   "emergency",
			want:    zapcore.PanicLevel,
			wantErr: false,
		},
		{
			name:    "succeed with alert",
			value:   "alert",
			want:    zapcore.PanicLevel,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseLevel(tt.value)
			if tt.wantErr {
				require.NotNil(t, err, "ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
			require.Equal(t, tt.want, got, "ParseLevel() got = %v, want %v", got, tt.want)
		})
	}
}
