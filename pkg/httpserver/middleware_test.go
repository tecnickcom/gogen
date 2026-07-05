package httpserver

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/random"
	"github.com/tecnickcom/gogen/pkg/redact"
	"github.com/tecnickcom/gogen/pkg/traceid"
)

func TestRequestInjectHandler_debug(t *testing.T) {
	t.Parallel()

	reader, writer, perr := os.Pipe()
	require.NoError(t, perr, "Unexpected error (os.Pipe)")

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug}))

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
	rnd := random.New(nil)

	handler := RequestInjectHandler(logger, traceid.DefaultHeader, redact.HTTPDataString, rnd, nextHandler)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
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

	rnd := random.New(nil)

	handler := RequestInjectHandler(logger, traceid.DefaultHeader, redact.HTTPDataString, rnd, nextHandler)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	handler.ServeHTTP(nil, req)

	cerr := writer.Close()
	require.NoError(t, cerr, "Unexpected error (writer.Close)")

	outlog := <-out
	require.NotEmpty(t, outlog, "captured log output")

	require.NotContains(t, outlog, "request_dump")
}

func TestRequestInjectHandler_redactsQuery(t *testing.T) {
	t.Parallel()

	reader, writer, perr := os.Pipe()
	require.NoError(t, perr, "Unexpected error (os.Pipe)")

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	nextHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	rnd := random.New(nil)

	handler := RequestInjectHandler(logger, traceid.DefaultHeader, redact.HTTPDataString, rnd, nextHandler)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/p?token=SUPERSECRET&x=1", nil)
	handler.ServeHTTP(nil, req)

	cerr := writer.Close()
	require.NoError(t, cerr, "Unexpected error (writer.Close)")

	outlog := <-out
	require.NotEmpty(t, outlog, "captured log output")

	// The secret must not leak through either the query or the request-URI field.
	require.NotContains(t, outlog, "SUPERSECRET", "query secret must be redacted in the inbound log")
	require.Contains(t, outlog, redact.RedactionMarker, "redaction marker must replace the secret")
	require.Contains(t, outlog, "request_query=")
	require.Contains(t, outlog, "request_uri=")
}

func TestRedactRequestURI(t *testing.T) {
	t.Parallel()

	redactFn := redact.HTTPDataString

	// No query: returned unchanged (the redact function is not invoked).
	require.Equal(t, "/path", redactRequestURI("/path", redactFn))

	// The query portion is redacted while the path is preserved.
	got := redactRequestURI("/path?token=SUPERSECRET&x=1", redactFn)
	require.True(t, strings.HasPrefix(got, "/path?"))
	require.NotContains(t, got, "SUPERSECRET")
	require.Contains(t, got, redact.RedactionMarker)
}
