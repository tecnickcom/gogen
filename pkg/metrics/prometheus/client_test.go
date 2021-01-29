package prometheus

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
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
			name: "succeeds with custom options",
			opts: []Option{WithCollector(
				prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "test",
						Help: "Test collector.",
					},
				),
			),
			},
			wantErr: false,
		},
		{
			name:    "fails with invalid option",
			opts:    []Option{func(c *Client) error { return fmt.Errorf("Error") }},
			wantErr: true,
		},
		{
			name: "fails with duplicate collector",
			opts: []Option{WithCollector(
				prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: NameInFlightRequests,
						Help: "Test collector.",
					},
				),
			),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.opts...)
			if tt.wantErr {
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func TestInstrumentHandler(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "New() unexpected error = %v", err)

	rr := httptest.NewRecorder()

	handler := c.InstrumentHandler("/test", c.MetricsHandlerFunc())

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err, "failed creating http request: %s", err)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	rt, err := testutil.GatherAndCount(c.registry, NameAPIRequests)
	require.NoError(t, err, "failed to gather metrics: %s", err)
	require.Equal(t, 1, rt, "failed to assert right metrics: got %v want %v", rt, 1)
}

func TestInstrumentRoundTripper(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "New() unexpected error = %v", err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`OK`))
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 1 * time.Second
	client.Transport = c.InstrumentRoundTripper(client.Transport)

	_, err = client.Get(server.URL)
	require.NoError(t, err, "client.Do() unexpected error = %v", err)

	rt, err := testutil.GatherAndCount(c.registry, NameOutboundRequests)
	require.NoError(t, err, "failed to gather metrics: %s", err)
	require.Equal(t, 1, rt, "failed to assert right metrics: got %v want %v", rt, 1)
}

func TestIncLogLevelCounter(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "unexpected error = %v", err)

	c.IncLogLevelCounter("debug")

	i, err := testutil.GatherAndCount(c.registry, NameErrorLevel)
	require.NoError(t, err, "failed to gather metrics: %s", err)

	if i != 1 {
		t.Errorf("failed to assert right metrics: got %v want %v", i, 1)
	}
}

func TestIncErrorCounter(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "unexpected error = %v", err)

	c.IncErrorCounter("test_task", "test_operation", "3791")

	i, err := testutil.GatherAndCount(c.registry, NameErrorCode)
	require.NoError(t, err, "failed to gather metrics: %s", err)

	if i != 1 {
		t.Errorf("failed to assert right metrics: got %v want %v", i, 1)
	}
}
