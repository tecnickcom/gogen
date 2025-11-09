package httpserver

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/redact"
	"github.com/tecnickcom/gogen/pkg/traceid"
)

func TestRequestInjectHandler_debug(t *testing.T) {
	t.Parallel()

	reader, writer, perr := os.Pipe()
	require.NoError(t, perr, "Unexpected error (os.Pipe)")

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		var buf bytes.Buffer

		wg.Done()

		_, err := io.Copy(&buf, reader)
		if err == nil {
			out <- buf.String()
		}
	}()

	wg.Wait()

	nextHandler := http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			// check if the request_time can be retrieved.
			reqTime, ok := httputil.GetRequestTime(r)
			assert.True(t, ok)
			assert.NotEmpty(t, reqTime)
		},
	)

	handler := RequestInjectHandler(logger, traceid.DefaultHeader, redact.HTTPData, nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(nil, req)

	cerr := writer.Close()
	require.NoError(t, cerr, "Unexpected error (writer.Close)")

	outlog := <-out
	require.NotEmpty(t, outlog, "captured log output")

	require.Contains(t, outlog, "request_dump")
}

func TestRequestInjectHandler_info(t *testing.T) {
	t.Parallel()

	reader, writer, perr := os.Pipe()
	require.NoError(t, perr, "Unexpected error (os.Pipe)")

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		var buf bytes.Buffer

		wg.Done()

		_, err := io.Copy(&buf, reader)
		if err == nil {
			out <- buf.String()
		}
	}()

	wg.Wait()

	nextHandler := http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			// check if the request_time can be retrieved.
			reqTime, ok := httputil.GetRequestTime(r)
			assert.True(t, ok)
			assert.NotEmpty(t, reqTime)
		},
	)

	handler := RequestInjectHandler(logger, traceid.DefaultHeader, redact.HTTPData, nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(nil, req)

	cerr := writer.Close()
	require.NoError(t, cerr, "Unexpected error (writer.Close)")

	outlog := <-out
	require.NotEmpty(t, outlog, "captured log output")

	require.NotContains(t, outlog, "request_dump")
}
