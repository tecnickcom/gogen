package opentel

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

//nolint:gocognit,paralleltest
func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		opts         []Option
		setEnvFn     func()
		wantErr      bool
		wantCloseErr bool
	}{
		{
			name:    "succeeds with empty options",
			wantErr: false,
		},
		{
			name: "succeeds with default STDOUT options",
			opts: []Option{
				WithTracerProviderFn(DefaultTracerProviderStdout),
				WithMeterProviderFn(DefaultMeterProviderStdout),
			},
			wantErr: false,
		},
		{
			name: "succeeds with default OTLP options",
			opts: []Option{
				WithTracerProviderFn(DefaultTracerProviderOTLP),
				WithMeterProviderFn(DefaultMeterProviderOTLP),
			},
			wantErr:      false,
			wantCloseErr: true,
		},
		{
			name: "succeeds with default providers and no env vars (STDOUT)",
			opts: []Option{
				WithTracerProviderFn(DefaultTracerProvider),
				WithMeterProviderFn(DefaultMeterProvider),
			},
			wantErr:      false,
			wantCloseErr: false,
		},
		{
			name: "succeeds with default providers and first opt env vars (OTLP)",
			opts: []Option{
				WithTracerProviderFn(DefaultTracerProvider),
				WithMeterProviderFn(DefaultMeterProvider),
			},
			setEnvFn: func() {
				t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "localhost:64000")
				t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "localhost:64000")
			},
			wantErr:      false,
			wantCloseErr: false,
		},
		{
			name: "succeeds with default providers and OTEL_EXPORTER_OTLP_ENDPOINT env var (OTLP)",
			opts: []Option{
				WithTracerProviderFn(DefaultTracerProvider),
				WithMeterProviderFn(DefaultMeterProvider),
			},
			setEnvFn: func() {
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:64000")
				t.Setenv("OTEL_DEPLOYMENT_ENVIRONMENT_NAME", "test")
			},
			wantErr:      false,
			wantCloseErr: false,
		},
		{
			name: "succeeds with in-memory exporter",
			opts: []Option{
				WithTracerProviderFn(func(ctx context.Context, _ *sdkresource.Resource) *sdktrace.TracerProvider {
					return DefaultTracerProviderWithExporter(DefaultSDKResource(ctx, "gogen-test", "0.0.0-1"), tracetest.NewInMemoryExporter())
				}),
				WithMeterProviderFn(func(ctx context.Context, _ *sdkresource.Resource) *sdkmetric.MeterProvider {
					return sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))
				}),
			},
			wantErr: false,
		},
		{
			name:    "fails with invalid option",
			opts:    []Option{func(_ *Client) error { return errors.New("Error") }},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnvFn != nil {
				tt.setEnvFn()
			}

			c, err := New(t.Context(), "gogen-test", "0.0.0-1", tt.opts...)
			if err == nil {
				defer func() {
					err := c.Close()
					if tt.wantErr {
						require.Error(t, err)
					}
				}()
			}

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestInstrumentHandler(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	c, err := New(ctx, "gogen-test", "0.0.0-1")
	require.NoError(t, err)

	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	rr := httptest.NewRecorder()

	handler := c.InstrumentHandler("/test", c.MetricsHandlerFunc())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	require.NoError(t, err, "failed creating http request: %s", err)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestInstrumentRoundTripper(t *testing.T) {
	t.Parallel()

	c, err := New(t.Context(), "gogen-test", "0.0.0-1")
	require.NoError(t, err)

	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`OK`))
			},
		),
	)
	defer server.Close()

	client := server.Client()
	client.Timeout = 1 * time.Second
	client.Transport = c.InstrumentRoundTripper(client.Transport)

	//nolint:noctx
	resp, err := client.Get(server.URL)
	require.NoError(t, err, "client.Get() unexpected error = %v", err)
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	client.Transport = c.InstrumentRoundTripper(nil)

	//nolint:noctx
	respd, err := client.Get(server.URL)
	require.NoError(t, err, "client.Get() unexpected error = %v", err)
	require.NotNil(t, resp)

	defer func() {
		err := respd.Body.Close()
		require.NoError(t, err, "error closing respd.Body")
	}()
}

func TestIncLogLevelCounter(t *testing.T) {
	t.Parallel()

	c, err := New(t.Context(), "gogen-test", "0.0.0-1")
	require.NoError(t, err)

	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	c.IncLogLevelCounter("debug")
}

func TestIncErrorCounter(t *testing.T) {
	t.Parallel()

	c, err := New(t.Context(), "gogen-test", "0.0.0-1")
	require.NoError(t, err)

	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	c.IncErrorCounter("test_task", "test_operation", "3791")
}

func TestClose(t *testing.T) {
	t.Parallel()

	c, err := New(t.Context(), "gogen-test", "0.0.0-1")
	require.NoError(t, err)

	err = c.Close()
	require.NoError(t, err, "Close() unexpected error = %v", err)
}

func TestInstrumentDB(t *testing.T) {
	t.Parallel()

	c, err := New(t.Context(), "gogen-test", "0.0.0-1")
	require.NoError(t, err)

	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	err = c.InstrumentDB("db_test", db)
	require.NoError(t, err)

	err = c.InstrumentDB("db_nil", nil)
	require.Error(t, err)
}

func Test_setInt64CounterError(t *testing.T) {
	t.Parallel()

	c, err := New(t.Context(), "gogen-test", "0.0.0-1")
	require.NoError(t, err)

	testErrMeter := &errMeter{}

	_, err = c.setInt64Counter(t.Context(), testErrMeter, "test")
	require.Error(t, err)
}

type errMeter struct {
	metric.Meter
}

func (m *errMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errors.New("test-error")
}

func TestTraceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      context.Context //nolint:containedctx
		expected string
	}{
		{
			name:     "context without trace ID",
			ctx:      t.Context(),
			expected: "",
		},
		{
			name: "context with valid trace ID",
			ctx: ContextWithSpanContext(
				t.Context(),
				[16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				[8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			),
			expected: "0102030405060708090a0b0c0d0e0f10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TraceID(tt.ctx)
			if got != tt.expected {
				t.Errorf("TraceID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestContextWithSpanContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		traceID trace.TraceID
		spanID  trace.SpanID
	}{
		{
			name:    "inject valid trace and span IDs",
			traceID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			spanID:  [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:    "inject zero trace and span IDs",
			traceID: [16]byte{},
			spanID:  [8]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			newCtx := ContextWithSpanContext(ctx, tt.traceID, tt.spanID)

			spanCtx := trace.SpanContextFromContext(newCtx)

			if spanCtx.TraceID() != tt.traceID {
				t.Errorf("TraceID mismatch: got %v, want %v", spanCtx.TraceID(), tt.traceID)
			}

			if spanCtx.SpanID() != tt.spanID {
				t.Errorf("SpanID mismatch: got %v, want %v", spanCtx.SpanID(), tt.spanID)
			}

			if !spanCtx.IsSampled() {
				t.Errorf("TraceFlags should be sampled")
			}
		})
	}
}

func TestTraceIDRoundtrip(t *testing.T) {
	t.Parallel()

	originalTraceID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	originalSpanID := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

	ctx := ContextWithSpanContext(context.Background(), originalTraceID, originalSpanID)
	retrievedTraceID := TraceID(ctx)

	expectedTraceID := trace.TraceID(originalTraceID).String()
	if retrievedTraceID != expectedTraceID {
		t.Errorf("Roundtrip TraceID failed: got %q, want %q", retrievedTraceID, expectedTraceID)
	}
}

//nolint:paralleltest
func Test_getOTELAttr(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		envname  string
		attrname string
		ora      map[string]string
		setEnvFn func()
		want     string
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name: "val",
			val:  "testval",
			want: "testval",
		},
		{
			name:    "env",
			envname: "TEST_ENV",
			setEnvFn: func() {
				t.Setenv("TEST_ENV", "testenv")
			},
			want: "testenv",
		},
		{
			name:     "attr",
			attrname: "testattrname",
			ora:      map[string]string{"testattrname": "testattr"},
			want:     "testattr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnvFn != nil {
				tt.setEnvFn()
			}

			got := getOTELAttr(tt.val, tt.envname, tt.attrname, tt.ora)

			require.Equal(t, tt.want, got)
		})
	}
}

func Test_parseOTELResourceAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want map[string]string
	}{
		{
			name: "empty",
			want: map[string]string{},
		},
		{
			name: "two",
			val:  "key1=val1,key2=val2",
			want: map[string]string{"key1": "val1", "key2": "val2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseOTELResourceAttributes(tt.val)

			require.Equal(t, tt.want, got)
		})
	}
}
