package logutil

import (
	"bytes"
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

func TestSlogHandler_DefaultFormatHonorsOut(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(WithOutWriter(&buf), WithLevel(LevelDebug))
	require.NoError(t, err)

	cfg.Format = -16 // force the default branch with an invalid format

	logger := slog.New(cfg.SlogHandler())
	logger.Debug("default branch")

	require.Contains(t, buf.String(), "default branch", "default branch must honor the configured writer")
	require.Contains(t, buf.String(), `"level":"debug"`, "default branch must honor the configured level (syslog-style name)")
}

func TestSlogHandler_NilOutFallsBackToStderr(t *testing.T) {
	t.Parallel()

	cfg, err := NewConfig()
	require.NoError(t, err)

	cfg.Out = nil // hand-cleared writer must not panic when building the handler

	require.NotPanics(t, func() {
		h := cfg.SlogHandler()
		require.NotNil(t, h)
	})
}

func TestSlogHandler_FormatNoneNoHookIsDiscard(t *testing.T) {
	t.Parallel()

	cfg, err := NewConfig(WithFormat(FormatNone))
	require.NoError(t, err)
	require.Nil(t, cfg.HookFn)

	h := cfg.SlogHandler()
	require.False(t, h.Enabled(t.Context(), LevelError), "FormatNone without a hook must be a zero-cost discard handler")
}

func TestSlogHandler_Source(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(WithOutWriter(&buf), WithFormat(FormatJSON), WithSource(true))
	require.NoError(t, err)

	cfg.SlogLogger().Info("with source")

	require.Contains(t, buf.String(), `"source":`, "source location must be present when enabled")
}

func TestSlogHandler_UnknownLevelKeepsSlogFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(WithOutWriter(&buf), WithFormat(FormatJSON), WithLevel(LevelTrace))
	require.NoError(t, err)

	cfg.SlogLogger().Log(t.Context(), LevelWarning+1, "odd") // 5: not a defined level

	require.Contains(t, buf.String(), `"level":"WARN+1"`, "unrecognized levels keep slog's banded name, not a bare number")
}

func TestSlogHandler_HookFiresUnderFormatNone(t *testing.T) {
	t.Parallel()

	var fired int

	cfg, err := NewConfig(
		WithFormat(FormatNone),
		WithLevel(LevelDebug),
		WithHookFn(func(_ LogLevel, _ string) { fired++ }),
	)
	require.NoError(t, err)

	l := cfg.SlogLogger()
	l.Error("x")
	l.Info("y")

	require.Equal(t, 2, fired, "hooks must fire under FormatNone (output discarded, side effects preserved)")
}

func TestSlogHandler_SyslogLevelNames(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(WithOutWriter(&buf), WithFormat(FormatJSON), WithLevel(LevelTrace))
	require.NoError(t, err)

	l := cfg.SlogLogger()
	l.Log(t.Context(), LevelCritical, "c")
	l.Log(t.Context(), LevelNotice, "n")
	l.Log(t.Context(), LevelEmergency, "e")

	out := buf.String()
	require.Contains(t, out, `"level":"critical"`, "extended levels must render as syslog names, not ERROR+8")
	require.Contains(t, out, `"level":"notice"`)
	require.Contains(t, out, `"level":"emergency"`)
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
