package logutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	require.Equal(t, cfg.Out, os.Stderr)
	require.Equal(t, FormatJSON, cfg.Format)
	require.Equal(t, LevelInfo, cfg.Level)
	require.Empty(t, cfg.CommonAttr)
	require.Nil(t, cfg.HookFn)
}

func TestNewConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "fail with invalid option",
			opts:    []Option{WithFormatStr("invalid")},
			wantErr: true,
		},
		{
			name:    "succeed with valid options",
			opts:    []Option{WithFormat(FormatJSON), WithLevel(LevelInfo)},
			wantErr: false,
		},
		{
			name:    "succeed with empty options",
			opts:    []Option{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := NewConfig(tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, cfg)
			}
		})
	}
}
