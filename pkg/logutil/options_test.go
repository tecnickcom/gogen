package logutil

import (
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

func TestWithLevelHookFn(t *testing.T) {
	t.Parallel()

	v := func(_ string) {
		// mock function
	}
	cfg := &Config{}
	err := WithLevelHookFn(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.LevelHookFn).Pointer())
}
