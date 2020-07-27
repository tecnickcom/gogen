// +build unit

package logging

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nexmoinc/gosrvlib/pkg/internal/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLogger(t *testing.T) {
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
			name:    "succeed with console format",
			opts:    []Option{WithFormat(ConsoleFormat), WithLevel(zap.InfoLevel)},
			wantErr: false,
		},
		{
			name:    "succeed with JSON format",
			opts:    []Option{WithFormat(JSONFormat), WithLevel(zap.InfoLevel)},
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

	mockSyncer := mocks.NewMockSyncer(mockCtrl)
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

	// do not override with other logger
	ctx2 := WithLogger(ctx, zap.NewNop())
	require.Equal(t, ctx, ctx2)
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
