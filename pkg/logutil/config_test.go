package logutil

import (
	"log/slog"
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

func TestSlogHookHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		record  slog.Record
		wantErr bool
	}{
		{
			name:   "test",
			record: slog.Record{Level: slog.LevelError, Message: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hkfn := func(level LogLevel, message string) {
				require.Equal(t, slog.LevelError, level)
				require.Equal(t, "test", message)
			}

			h := slog.DiscardHandler

			h = &SlogHookHandler{
				Handler: h,
				hookFn:  hkfn,
			}

			err := h.Handle(t.Context(), tt.record)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSlogHandler(t *testing.T) {
	t.Parallel()

	hkfn := func(_ LogLevel, _ string) {}

	tests := []struct {
		name string
		opts []Option
	}{
		{
			name: "hook function",
			opts: []Option{WithFormat(FormatNone), WithHookFn(hkfn)},
		},
		{
			name: "json",
			opts: []Option{WithFormat(FormatJSON)},
		},
		{
			name: "console",
			opts: []Option{WithFormat(FormatConsole)},
		},
		{
			name: "default",
			opts: []Option{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := NewConfig(tt.opts...)
			require.NoError(t, err)

			if tt.name == "default" {
				cfg.Format = -16 // force invalid value to trigger the default option
			}

			sh := cfg.SlogHandler()

			require.NotNil(t, sh)
		})
	}
}

func TestConfig_SlogLogger(t *testing.T) {
	t.Parallel()

	cfg, err := NewConfig()
	require.NoError(t, err)

	l := cfg.SlogLogger()

	require.NotNil(t, l)
}

func TestConfig_SlogDefaultLogger(t *testing.T) {
	t.Parallel()

	cfg, err := NewConfig()
	require.NoError(t, err)

	l := cfg.SlogDefaultLogger()

	require.NotNil(t, l)
}

func Test_defaultTraceID(t *testing.T) {
	t.Parallel()

	s := defaultTraceID()

	require.Empty(t, s)
}
