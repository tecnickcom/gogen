// +build unit

package logging

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithFormat(t *testing.T) {
	v := JSONFormat
	cfg := &config{}
	err := WithFormat(v)(cfg)
	require.Nil(t, err)
	require.Equal(t, v, cfg.format)
}

func TestWithLevel(t *testing.T) {
	v := zap.DebugLevel
	cfg := &config{}
	err := WithLevel(v)(cfg)
	require.Nil(t, err)
	require.Equal(t, v, cfg.level)
}

func TestWithFields(t *testing.T) {
	v := []zap.Field{zap.String("a", "a"), zap.String("b", "b")}
	cfg := &config{}
	err := WithFields(v...)(cfg)
	require.Nil(t, err)
	require.Len(t, v, len(cfg.fields))
	require.EqualValues(t, v, cfg.fields)
}

func TestWithIncrementLogMetricsFunc(t *testing.T) {
	v := func(s string) {
		// mock function
	}
	cfg := &config{}
	err := WithIncrementLogMetricsFunc(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.incMetricLogLevel).Pointer())
}

func TestWithFormatStr(t *testing.T) {
	tests := []struct {
		name      string
		testValue string
		wantErr   bool
	}{
		{
			name:      "should pass with console",
			testValue: "console",
		},
		{
			name:      "should error",
			testValue: "unicorn",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config{}
			if err := WithFormatStr(tt.testValue)(cfg); (err != nil) != tt.wantErr {
				t.Errorf("WithFormatStr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWithLevelStr(t *testing.T) {
	tests := []struct {
		name      string
		testValue string
		wantErr   bool
	}{
		{
			name:      "should pass with debug",
			testValue: "debug",
		},
		{
			name:      "should error",
			testValue: "unicorn",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config{}
			if err := WithLevelStr(tt.testValue)(cfg); (err != nil) != tt.wantErr {
				t.Errorf("WithLevelStr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
