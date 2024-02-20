//go:generate mockgen -package logging -destination ./mock_test.go . Syncer

package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLogger(t *testing.T) {
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
			name:    "fail with invalid format",
			opts:    []Option{WithFormat(-1), WithLevel(zap.InfoLevel)},
			wantErr: true,
		},
		{
			name:    "fail with no format",
			opts:    []Option{WithFormat(noFormat), WithLevel(zap.InfoLevel)},
			wantErr: true,
		},
		{
			name:    "succeed with console format",
			opts:    []Option{WithFormat(ConsoleFormat), WithLevel(zap.InfoLevel)},
			wantErr: false,
		},
		{
			name:    "succeed with JSON format",
			opts:    []Option{WithFormat(JSONFormat), WithLevel(zap.InfoLevel)},
			wantErr: false,
		},
		{
			name:    "succeed with empty options",
			opts:    []Option{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := tt.opts

			var loggedMetricLevel string

			opts = append(opts, WithIncrementLogMetricsFunc(func(ll string) {
				loggedMetricLevel = ll
			}))

			l, err := NewLogger(opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if l != nil {
				l.Info("test")
				require.Equal(t, "info", loggedMetricLevel)
			}
		})
	}
}

func TestNopLogger(t *testing.T) {
	t.Parallel()
	require.Equal(t, zap.NewNop(), NopLogger())
}

func TestSync(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSyncer := NewMockSyncer(mockCtrl)
	mockSyncer.EXPECT().Sync().Times(1)

	Sync(mockSyncer)
}

func TestWithComponent(t *testing.T) {
	t.Parallel()

	ctx, logs := testLogContext(zap.InfoLevel)
	l := WithComponent(ctx, "test_c")

	l.Info("test w/ component")

	logEntry := logs.All()[0]
	logContextMap := logEntry.ContextMap()

	cValue, cExists := logContextMap["component"]
	require.True(t, cExists, "component field missing")
	require.Equal(t, "test_c", cValue)

	require.Equal(t, "test w/ component", logEntry.Message)
}

func TestWithComponentAndMethod(t *testing.T) {
	t.Parallel()

	ctx, logs := testLogContext(zap.InfoLevel)
	l := WithComponentAndMethod(ctx, "test_c", "test_m")

	l.Info("test w/ component and method")

	logEntry := logs.All()[0]
	logContextMap := logEntry.ContextMap()

	cValue, cExists := logContextMap["component"]
	require.True(t, cExists, "component field missing")
	require.Equal(t, "test_c", cValue)

	mValue, mExists := logContextMap["method"]
	require.True(t, mExists, "method field missing")
	require.Equal(t, "test_m", mValue)

	require.Equal(t, "test w/ component and method", logEntry.Message)
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	l1 := zap.NewNop()
	ctx := WithLogger(context.Background(), l1)

	el1 := FromContext(ctx)
	require.Equal(t, el1, l1)

	// do not override with same logger
	ctx1 := WithLogger(ctx, l1)
	require.Equal(t, ctx, ctx1)

	// override with another logger
	l2 := zap.NewNop()
	ctx2 := WithLogger(ctx, l2)
	require.NotEqual(t, ctx, ctx2)

	// test with real logger
	l3, err := NewLogger()
	require.NoError(t, err)

	ctx3 := WithLogger(ctx, l3)
	require.NotEqual(t, ctx, ctx3)

	// override with logger change
	l4 := l3.With(zap.String("A", "B"))
	ctx4 := WithLogger(ctx3, l4)
	require.NotEqual(t, ctx3, ctx4)
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	// Context without logger
	l1 := FromContext(context.Background())
	require.NotNil(t, l1)

	// Context with logger
	ctx := WithLogger(context.Background(), zap.NewNop())
	l2 := FromContext(ctx)
	require.NotNil(t, l2)
}

func TestNewDefaultLogger(t *testing.T) {
	t.Parallel()

	l, err := NewDefaultLogger("test", "0.0.0", "1", "json", "info")
	require.NoError(t, err)
	require.NotNil(t, l)

	// invalid format
	l2, err := NewDefaultLogger("test", "0.0.0", "1", "unicorn", "info")
	require.Error(t, err)
	require.Nil(t, l2)
}

func testLogContext(level zapcore.Level) (context.Context, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	l := zap.New(core)

	return WithLogger(context.Background(), l), logs
}

// MemorySink implements zap.Sink by writing all messages to a buffer.
type MemorySink struct {
	*bytes.Buffer
}

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

func (s *MemorySink) Close() error { return nil }
func (s *MemorySink) Sync() error  { return nil }

func TestLogDifferences(t *testing.T) {
	t.Parallel()

	// Create a sink instance, and register it with zap for the "memory" protocol.
	sink := &MemorySink{new(bytes.Buffer)}
	err := zap.RegisterSink("memdiff", func(*url.URL) (zap.Sink, error) {
		return sink, nil
	})
	require.NoError(t, err)

	l, err := NewLogger(
		WithFields(
			zap.String("program", "test_log_diff"),
			zap.String("version", "2.3.5"),
			zap.String("release", "7"),
		),
		WithFormatStr("json"),
		WithLevelStr("info"),
		WithOutputPaths([]string{"memdiff://"}),      // Redirect all messages to the MemorySink.
		WithErrorOutputPaths([]string{"memdiff://"}), // Redirect all errors to the MemorySink.
	)
	require.NoError(t, err)
	require.NotNil(t, l)

	l.Info("A")
	time.Sleep(time.Second)
	l.Info("B")

	err = l.Sync()
	require.NoError(t, err)

	out := sink.String()
	require.NotEmpty(t, out, "captured log output")

	logs := strings.SplitN(out, "\n", 2)
	require.Len(t, logs, 2, "there should be 2 logs")

	type LogData struct {
		Level     string `json:"level"`
		Timestamp int64  `json:"timestamp"`
		Msg       string `json:"msg"`
		Hostname  string `json:"hostname"`
		Program   string `json:"program"`
		Version   string `json:"version"`
		Release   string `json:"release"`
	}

	var log1 LogData
	err = json.Unmarshal([]byte(logs[0]), &log1)
	require.NoError(t, err)
	require.NotEmpty(t, log1.Level, "first log level should not be empty")
	require.NotEmpty(t, log1.Timestamp, "first log timestamp should not be empty")
	require.NotEmpty(t, log1.Msg, "first log msg should not be empty")
	require.NotEmpty(t, log1.Program, "first log program should not be empty")
	require.NotEmpty(t, log1.Version, "first log version should not be empty")
	require.NotEmpty(t, log1.Release, "first log release should not be empty")

	var log2 LogData
	err = json.Unmarshal([]byte(logs[1]), &log2)
	require.NoError(t, err)
	require.NotEmpty(t, log2.Level, "second log level should not be empty")
	require.NotEmpty(t, log2.Timestamp, "second log timestamp should not be empty")
	require.NotEmpty(t, log2.Msg, "second log msg should not be empty")
	require.NotEmpty(t, log2.Program, "second log program should not be empty")
	require.NotEmpty(t, log2.Version, "second log version should not be empty")
	require.NotEmpty(t, log2.Release, "second log release should not be empty")

	require.Equal(t, log1.Level, log2.Level, "Logs should have the same level")
	require.NotEqual(t, log1.Timestamp, log2.Timestamp, "Logs should have different timestamp")
	require.NotEqual(t, log1.Msg, log2.Msg, "Logs should have different msg")
	require.Equal(t, log1.Hostname, log2.Hostname, "Logs should have the same hostname")
	require.Equal(t, log1.Program, log2.Program, "Logs should have the same program")
	require.Equal(t, log1.Version, log2.Version, "Logs should have the same version")
	require.Equal(t, log1.Release, log2.Release, "Logs should have the same release")
}

type testCloseError struct{}

func (c *testCloseError) Close() error {
	return errors.New("close error")
}

type testCloseOK struct{}

func (c *testCloseOK) Close() error {
	return nil
}

func TestClose(t *testing.T) {
	t.Parallel()

	// Create a sink instance, and register it with zap for the "memory" protocol.
	sink := &MemorySink{new(bytes.Buffer)}
	err := zap.RegisterSink("memclose", func(*url.URL) (zap.Sink, error) {
		return sink, nil
	})
	require.NoError(t, err)

	l, err := NewLogger(
		WithFormatStr("json"),
		WithLevelStr("debug"),
		WithOutputPaths([]string{"memclose://"}),      // Redirect all messages to the MemorySink.
		WithErrorOutputPaths([]string{"memclose://"}), // Redirect all errors to the MemorySink.
	)
	require.NoError(t, err)
	require.NotNil(t, l)

	ctx := WithLogger(context.Background(), l)

	objOK := &testCloseOK{}

	Close(ctx, objOK, "test error OK")

	err = l.Sync()
	require.NoError(t, err)

	out := sink.String()
	require.Empty(t, out, "expecting empty log")

	objErr := &testCloseError{}
	Close(ctx, objErr, "test error ERROR")

	err = l.Sync()
	require.NoError(t, err)

	out = sink.String()
	require.NotEmpty(t, out, "expecting non-empty log")
	require.Contains(t, out, "\"msg\":\"test error ERROR\"")
	require.Contains(t, out, "\"error\":\"close error\"}\n")
}
