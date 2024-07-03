package httpclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	body := make([]byte, 0)

	for range 100 {
		body = append(body, []byte(bodyStr+`\n`)...)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	t.Cleanup(
		func() {
			server.Close()
		},
	)

	tests := []struct {
		name          string
		logLevel      string
		logSinkScheme string
		requestAddr   string
		opts          []Option
		wantErr       bool
	}{
		{
			name:          "no options, info level",
			logLevel:      "info",
			logSinkScheme: "memdiffa",
			requestAddr:   server.URL,
		},
		{
			name:          "no options, debug level",
			logLevel:      "debug",
			logSinkScheme: "memdiffb",
			requestAddr:   server.URL,
		},
		{
			name:          "prefix, debug level",
			logLevel:      "debug",
			logSinkScheme: "memdiffc",
			requestAddr:   server.URL,
			opts:          []Option{WithLogPrefix("testprefix_")},
		},
		{
			name:          "no options, error",
			logLevel:      "debug",
			logSinkScheme: "memdiffd",
			requestAddr:   "/error",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := New(tt.opts...)
			ctx := context.Background()

			// Create a sink instance, and register it with zap for the "memory" protocol.
			sink := &MemorySink{new(bytes.Buffer)}
			err := zap.RegisterSink(tt.logSinkScheme, func(*url.URL) (zap.Sink, error) {
				return sink, nil
			})
			require.NoError(t, err)

			l, err := logging.NewLogger(
				logging.WithFields(
					zap.String("program", "test_log"),
					zap.String("version", "1.2.3"),
					zap.String("release", "4"),
				),
				logging.WithFormatStr("json"),
				logging.WithLevelStr(tt.logLevel),
				logging.WithOutputPaths([]string{tt.logSinkScheme + "://"}),      // Redirect all messages to the MemorySink.
				logging.WithErrorOutputPaths([]string{tt.logSinkScheme + "://"}), // Redirect all errors to the MemorySink.
			)
			require.NoError(t, err)
			require.NotNil(t, l)

			ctx = logging.WithLogger(ctx, l)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, tt.requestAddr, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)

			t.Cleanup(
				func() {
					if resp != nil {
						err := resp.Body.Close()
						require.NoError(t, err, "error closing resp.Body")
					}
				},
			)

			// check logs
			out := sink.String()
			require.NotEmpty(t, out, "captured log output")
			require.Contains(t, out, `"`+client.logPrefix+`traceid"`)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, out, `"`+client.logPrefix+`error"`)

				return
			}

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err)
			}()

			require.NoError(t, err)
			require.NotNil(t, resp)
			responseBody, errb := io.ReadAll(resp.Body)
			require.NoError(t, errb)
			require.Equal(t, body, responseBody)
			require.Contains(t, out, `"`+client.logPrefix+`outbound"`)

			if tt.logLevel == "debug" {
				require.Contains(t, out, `"`+client.logPrefix+`request":"GET / HTTP/1.1`)
				require.Contains(t, out, `"`+client.logPrefix+`response":"HTTP/1.1 200 OK`)
				require.Contains(t, out, bodyStr)
			} else {
				require.NotContains(t, out, `"`+client.logPrefix+`request":`)
				require.NotContains(t, out, `"`+client.logPrefix+`response":`)
				require.NotContains(t, out, bodyStr)
			}
		})
	}
}

// MemorySink implements zap.Sink by writing all messages to a buffer.
type MemorySink struct {
	*bytes.Buffer
}

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

func (s *MemorySink) Close() error { return nil }
func (s *MemorySink) Sync() error  { return nil }
