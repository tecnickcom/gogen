package metrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
			name:    "succeeds with default options",
			opts:    DefaultCollectors,
			wantErr: false,
		},
		{
			name:    "fails with invalid option",
			opts:    []Option{func(c *Client) error { return fmt.Errorf("Error") }},
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

func TestIncLogLevelCounter(t *testing.T) {
	t.Parallel()

	c, err := New(DefaultCollectors...)
	require.NoError(t, err, "unexpected error = %v", err)

	c.IncLogLevelCounter("debug")

	i, err := testutil.GatherAndCount(c.Registry, MetricErrorLevel)
	if err != nil {
		t.Errorf("failed to gather metrics: %s", err)
	}

	if i != 1 {
		t.Errorf("failed to assert right metrics: got %v want %v", i, 1)
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()

	c, err := New(DefaultCollectors...)
	require.NoError(t, err, "unexpected error = %v", err)

	rr := httptest.NewRecorder()

	handler := c.InstrumentHandler("/test", c.MetricsHandlerFunc())

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	rt, err := testutil.GatherAndCount(c.Registry, MetricAPIRequests)
	if err != nil {
		t.Errorf("failed to gather metrics: %s", err)
	}
	require.Equal(t, 1, rt, "failed to assert right metrics: got %v want %v", rt, 1)
}

func TestDefaultMetricsHandlerFunc(t *testing.T) {
	t.Parallel()

	h := DefaultMetricsHandlerFunc()
	require.NotNil(t, h, "unexpected nil")
}
