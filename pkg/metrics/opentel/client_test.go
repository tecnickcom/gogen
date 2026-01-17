package opentel

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "succeeds with empty options",
			wantErr: false,
		},
		{
			name: "succeeds with options",
			opts: []Option{
				WithTracerProvider(trace.NewTracerProvider()),
				WithMeterProvider(sdkmetric.NewMeterProvider()),
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
			t.Parallel()

			c, err := New(t.Context(), "gogen-test", "0.0.0-1", tt.opts...)
			if err == nil {
				defer func() {
					err := c.Close()
					require.NoError(t, err)
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
