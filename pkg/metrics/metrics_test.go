package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestInstrumentDB(t *testing.T) {
	t.Parallel()

	c := &Default{}

	db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	err = c.InstrumentDB("db_test", db)
	require.NoError(t, err)
}

func TestInstrumentHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	c := &Default{}

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

	c := &Default{}

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
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
}

func TestIncLogLevelCounter(t *testing.T) {
	t.Parallel()

	c := &Default{}

	c.IncLogLevelCounter("debug")
}

func TestIncErrorCounter(t *testing.T) {
	t.Parallel()

	c := &Default{}

	c.IncErrorCounter("test_task", "test_operation", "3791")
}

func TestClose(t *testing.T) {
	t.Parallel()

	c := &Default{}

	err := c.Close()
	require.NoError(t, err)
}
