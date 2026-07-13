package logutil

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithOutWriter(t *testing.T) {
	t.Parallel()

	cfg := &Config{}

	err := WithOutWriter(os.Stdout)(cfg)
	require.NoError(t, err)
	require.Equal(t, os.Stdout, cfg.Out)
}

// TestWithOutWriter_nil pins that an unusable writer is rejected at construction rather than left to
// panic on the first log line. A typed nil is the case a plain `w == nil` check misses: the interface
// holding a nil *os.File is not itself nil, but writing to it panics all the same.
func TestWithOutWriter_nil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  io.Writer
	}{
		{name: "untyped nil", out: nil},
		{name: "typed nil pointer", out: (*os.File)(nil)},
		{name: "typed nil pointer to a buffer", out: (*bytes.Buffer)(nil)},
		{name: "typed nil map", out: nilMapWriter(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}

			err := WithOutWriter(tt.out)(cfg)
			require.Error(t, err, "an unusable writer must be rejected")
			require.Nil(t, cfg.Out, "and must not be stored")
		})
	}
}

// nilMapWriter is a writer whose underlying type is a map, so a nil one is a typed nil of a kind other
// than a pointer.
type nilMapWriter map[string]string

func (nilMapWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestWithFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		testValue LogFormat
		wantErr   bool
	}{
		{
			name:      "invalid",
			testValue: -16,
			wantErr:   true,
		},
		{
			name:      "valid",
			testValue: FormatConsole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}

			err := WithFormat(tt.testValue)(cfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithFormatStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		testValue string
		wantErr   bool
	}{
		{
			name:      "invalid",
			testValue: "invalid",
			wantErr:   true,
		},
		{
			name:      "valid",
			testValue: "console",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}

			err := WithFormatStr(tt.testValue)(cfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		testValue LogLevel
		wantErr   bool
	}{
		{
			name:      "invalid",
			testValue: -16,
			wantErr:   true,
		},
		{
			name:      "valid",
			testValue: LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}

			err := WithLevel(tt.testValue)(cfg)
			if tt.wantErr {
				require.Error(t, err)
				require.EqualError(t, err, "invalid log level")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithLevelStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		testValue string
		wantErr   bool
	}{
		{
			name:      "invalid",
			testValue: "invalid",
			wantErr:   true,
		},
		{
			name:      "valid",
			testValue: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}

			err := WithLevelStr(tt.testValue)(cfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithCommonAttr(t *testing.T) {
	t.Parallel()

	v := []Attr{slog.String("a", "a"), slog.Int("b", 1)}
	cfg := &Config{}
	err := WithCommonAttr(v...)(cfg)
	require.NoError(t, err)
	require.Len(t, v, len(cfg.CommonAttr))
	require.Equal(t, v, cfg.CommonAttr)
}

func TestWithHookFn(t *testing.T) {
	t.Parallel()

	v := func(_ LogLevel, _ string) {
		// void
	}

	cfg := &Config{}
	err := WithHookFn(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.HookFn).Pointer())
}

func TestWithSource(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	err := WithSource(true)(cfg)
	require.NoError(t, err)
	require.True(t, cfg.Source)
}

func TestWithTraceIDFn(t *testing.T) {
	t.Parallel()

	v := func() string {
		return "test-123"
	}

	cfg := &Config{}
	err := WithTraceIDFn(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.TraceIDFn).Pointer())
}
