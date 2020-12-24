package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestIncLogLevelCounter(t *testing.T) {
	t.Parallel()

	IncLogLevelCounter("debug")

	i, err := testutil.GatherAndCount(prometheus.DefaultGatherer, "error_level_total")
	if err != nil {
		t.Errorf("failed to gather metrics: %s", err)
	}

	if i != 1 {
		t.Errorf("failed to assert right metrics: got %v want %v", i, 1)
	}
}

func TestPrometheusHandler(t *testing.T) {
	rr := httptest.NewRecorder()

	handler := Handler("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	rt, err := testutil.GatherAndCount(prometheus.DefaultGatherer, "api_requests_total")
	if err != nil {
		t.Errorf("failed to gather metrics: %s", err)
	}
	require.Equal(t, 1, rt, "failed to assert right metrics: got %v want %v", rt, 1)
}
