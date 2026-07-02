package logsrv

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	attr := []logutil.Attr{
		slog.String("program", "test"),
		slog.Int("version", 1),
	}

	var hookValue string

	hookFn := func(_ logutil.LogLevel, message string) {
		hookValue = message
	}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(os.Stderr),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithCommonAttr(attr...),
		logutil.WithHookFn(hookFn),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	l := NewLogger(cfg)

	require.NotNil(t, l)

	l.Info("test")

	require.Equal(t, "test", hookValue)
}

// TestNewLogger_nilTraceIDFn verifies that a nil TraceIDFn is treated as
// valid (as logutil does): the logger is created without panicking and the
// trace ID field is simply omitted from the output.
func TestNewLogger_nilTraceIDFn(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithTraceIDFn(nil),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Nil(t, cfg.TraceIDFn)

	require.NotPanics(t, func() {
		l := NewLogger(cfg)

		require.NotNil(t, l)

		l.Info("no trace id")
	})

	require.Contains(t, out.String(), "no trace id")
	require.NotContains(t, out.String(), traceIDName, "the trace ID field must be omitted when TraceIDFn is nil")
}

// TestNewLogger_hookReceivesOriginalLevel verifies the hook is invoked with
// the original slog record level instead of the zerolog-collapsed one
// (zerolog maps Notice->Info and Critical->Error).
func TestNewLogger_hookReceivesOriginalLevel(t *testing.T) {
	t.Parallel()

	type hookCall struct {
		level   logutil.LogLevel
		message string
	}

	var calls []hookCall

	hookFn := func(level logutil.LogLevel, message string) {
		calls = append(calls, hookCall{level: level, message: message})
	}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(io.Discard),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithHookFn(hookFn),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	l := NewLogger(cfg)

	require.NotNil(t, l)

	l.Log(t.Context(), logutil.LevelNotice, "notice message")
	l.Log(t.Context(), logutil.LevelCritical, "critical message")

	require.Equal(t, []hookCall{
		{level: logutil.LevelNotice, message: "notice message"},
		{level: logutil.LevelCritical, message: "critical message"},
	}, calls, "the hook must receive the original slog levels, not the zerolog-collapsed ones")
}

// TestNewLogger_concurrent creates many loggers concurrently while each one is
// actively logging. Before the sync.Once fix, NewLogger reassigned the global
// szlog.LogLevels map on every call while previously created handlers read it
// at log time, which the race detector flags. This test must pass under -race.
func TestNewLogger_concurrent(t *testing.T) {
	t.Parallel()

	const goroutines = 16

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(io.Discard),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			l := NewLogger(cfg)

			assert.NotNil(t, l)

			for range 50 {
				l.Info("concurrent log line")
				l.Debug("concurrent debug line")
			}
		}()
	}

	wg.Wait()
}

func Test_writerByFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format logutil.LogFormat
		out    io.Writer
		want   io.Writer
	}{
		{
			name:   "json",
			format: logutil.FormatJSON,
			out:    os.Stdout,
			want:   os.Stdout,
		},
		{
			name:   "console",
			format: logutil.FormatConsole,
			out:    os.Stdout,
			want:   zerolog.ConsoleWriter{Out: os.Stdout},
		},
		{
			name:   "none",
			format: logutil.FormatNone,
			out:    os.Stdout,
			want:   io.Discard,
		},
		{
			name:   "default",
			format: 56,
			out:    os.Stdout,
			want:   os.Stdout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := writerByFormat(tt.format, tt.out)

			require.Equal(t, tt.want, got)
		})
	}
}
