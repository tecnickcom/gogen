package httpclient

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	timeout := 17 * time.Second
	traceid := "test-header-123"
	component := "test-component"
	logPrefix := "prefixtest_"
	fn := func(next http.RoundTripper) http.RoundTripper { return next }
	opts := []Option{
		WithTimeout(timeout),
		WithRoundTripper(fn),
		WithTraceIDHeaderName(traceid),
		WithComponent(component),
		WithLogPrefix(logPrefix),
	}
	got := New(opts...)
	require.NotNil(t, got, "New() returned client should not be nil")
	require.Equal(t, traceid, got.traceIDHeaderName)
	require.Equal(t, component, got.component)
	require.Equal(t, timeout, got.client.Timeout)
	require.Equal(t, fn(http.DefaultTransport), got.client.Transport)
}

//nolint:gocognit,tparallel,paralleltest
func TestClient_Do(t *testing.T) {
	bodyStr := `TEST BODY OK`
	body := bytes.Repeat([]byte(bodyStr+`\n`), 10000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	t.Cleanup(
		func() {
			server.Close()
		},
	)

	tests := []struct {
		name        string
		logLevel    slog.Level
		requestAddr string
		opts        []Option
		wantErr     bool
	}{
		{
			name:        "no options, info level",
			logLevel:    slog.LevelInfo,
			requestAddr: server.URL,
		},
		{
			name:        "no options, debug level",
			logLevel:    slog.LevelDebug,
			requestAddr: server.URL,
		},
		{
			name:        "prefix, debug level",
			logLevel:    slog.LevelDebug,
			requestAddr: server.URL,
			opts:        []Option{WithLogPrefix("testprefix_")},
		},
		{
			name:        "no options, error",
			logLevel:    slog.LevelDebug,
			requestAddr: "/error",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader, writer, perr := os.Pipe()
			require.NoError(t, perr, "Unexpected error (os.Pipe)")

			logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: tt.logLevel}))
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

			tt.opts = append(tt.opts, WithLogger(logger))

			client := New(tt.opts...)
			ctx := t.Context()

			req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, tt.requestAddr, nil)
			require.NoError(t, rerr)

			resp, err := client.Do(req)

			t.Cleanup(
				func() {
					if resp != nil {
						cerr := resp.Body.Close()
						require.NoError(t, cerr, "error closing resp.Body")
					}
				},
			)

			cerr := writer.Close()
			require.NoError(t, cerr, "Unexpected error (writer.Close)")

			outlog := <-out
			require.NotEmpty(t, outlog, "captured log output")

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			responseBody, berr := io.ReadAll(resp.Body)
			require.NoError(t, berr)
			require.Equal(t, body, responseBody)

			if tt.logLevel == slog.LevelDebug {
				require.Contains(t, outlog, `request=`)
				require.Contains(t, outlog, `response=`)
				require.Contains(t, outlog, bodyStr)
			} else {
				require.NotContains(t, outlog, `request=`)
				require.NotContains(t, outlog, `response=`)
				require.NotContains(t, outlog, bodyStr)
			}
		})
	}
}
