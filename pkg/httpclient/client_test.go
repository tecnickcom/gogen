package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/traceid"
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

	// The identity round-tripper returns its argument unchanged, so the
	// transport must be the client's own cloned *http.Transport (not the
	// process-wide http.DefaultTransport).
	require.IsType(t, &http.Transport{}, got.client.Transport)
	require.NotSame(t, http.DefaultTransport, got.client.Transport)
}

func TestNew_TransportIsolation(t *testing.T) {
	t.Parallel()

	// Each client must own a private transport, so two clients never share a
	// transport with each other nor with http.DefaultTransport.
	c1 := New()
	c2 := New()

	require.NotSame(t, c1.client.Transport, c2.client.Transport)
	require.NotSame(t, http.DefaultTransport, c1.client.Transport)
	require.NotSame(t, http.DefaultTransport, c2.client.Transport)
}

//nolint:paralleltest // swaps the process-wide http.DefaultTransport.
func TestDefaultTransport_NotHTTPTransportFallback(t *testing.T) {
	orig := http.DefaultTransport

	t.Cleanup(func() { http.DefaultTransport = orig })

	rt := &stubRoundTripper{}
	http.DefaultTransport = rt

	require.Same(t, rt, defaultTransport())
}

// stubRoundTripper is an http.RoundTripper that is not an *http.Transport,
// exercising the defaultTransport fallback path. Being a pointer type it is
// comparable by identity.
type stubRoundTripper struct{}

func (*stubRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("TEST")
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

// TestClient_Do_ZeroTimeout verifies that WithTimeout(0) means "no timeout"
// (the net/http convention) instead of an already-expired context that would
// fail every request immediately.
func TestClient_Do_ZeroTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	t.Cleanup(server.Close)

	client := New(WithTimeout(0))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("OK"), body)
	require.NoError(t, resp.Body.Close())
}

// TestClient_Do_InvalidTraceIDReplaced verifies that an invalid trace ID
// stored in the context (e.g. containing control characters) never reaches the
// outbound header: it is replaced with a freshly generated valid ID.
func TestClient_Do_InvalidTraceIDReplaced(t *testing.T) {
	t.Parallel()

	const invalidID = "bad\x00trace\nid"

	var gotHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(traceid.DefaultHeader)
		_, _ = w.Write([]byte("OK"))
	}))
	t.Cleanup(server.Close)

	client := New()

	ctx := traceid.NewContext(t.Context(), invalidID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, resp.Body.Close())

	require.NotEmpty(t, gotHeader)
	require.NotEqual(t, invalidID, gotHeader, "the invalid trace ID must not be propagated")
	require.Regexp(t, `^[0-9A-Za-z\-\_\.]{1,64}$`, gotHeader)
}

func TestClient_Do_CloseBodyCancelsContext(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	t.Cleanup(server.Close)

	// Capture the per-request context handed to the transport so we can assert
	// it is canceled once the body is closed.
	got := make(chan context.Context, 1)
	wrap := func(next http.RoundTripper) http.RoundTripper {
		return &contextCapturingRoundTripper{next: next, ch: got}
	}

	client := New(WithTimeout(time.Minute), WithRoundTripper(wrap))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// The body is wrapped so closing it cancels the per-request context.
	_, ok := resp.Body.(*cancelReadCloser)
	require.True(t, ok, "resp.Body should be wrapped in *cancelReadCloser")

	reqCtx := <-got

	// Before Close the context must still be live (not yet canceled).
	require.NoError(t, reqCtx.Err(), "context must be live before closing the body")

	require.NoError(t, resp.Body.Close())

	// After Close the per-request context must be done (canceled), proving the
	// timeout timer was released instead of leaking until the timeout.
	select {
	case <-reqCtx.Done():
		require.ErrorIs(t, reqCtx.Err(), context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("context was not canceled after closing the body")
	}
}

// contextCapturingRoundTripper records the per-request context and delegates.
type contextCapturingRoundTripper struct {
	next http.RoundTripper
	ch   chan context.Context
}

func (t *contextCapturingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.ch <- r.Context()

	return t.next.RoundTrip(r) //nolint:wrapcheck
}

func TestClient_Do_ErrorPathCancelsContext(t *testing.T) {
	t.Parallel()

	// A round-tripper that fails without returning a response forces the
	// no-response (error) branch, which must cancel immediately.
	canceled := make(chan context.Context, 1)
	fail := func(_ http.RoundTripper) http.RoundTripper {
		return &capturingRoundTripper{ch: canceled}
	}

	client := New(WithRoundTripper(fail))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.invalid", nil)
	require.NoError(t, err)

	//nolint:bodyclose // resp is nil on this error path; nothing to close.
	resp, err := client.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)

	gotCtx := <-canceled

	select {
	case <-gotCtx.Done():
		require.ErrorIs(t, gotCtx.Err(), context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("context was not canceled on the error path")
	}
}

// capturingRoundTripper captures the request context and returns an error
// without producing a response.
type capturingRoundTripper struct {
	ch chan context.Context
}

func (t *capturingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.ch <- r.Context()

	return nil, errors.New("TEST round-trip failure")
}

func TestWithDialContext_Isolation(t *testing.T) {
	t.Parallel()

	// Two clients with different dialers must not interfere with each other,
	// and the process-wide http.DefaultTransport must stay untouched.
	defaultDialer := http.DefaultTransport.(*http.Transport).DialContext //nolint:forcetypeassert

	var calls1, calls2 atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	t.Cleanup(server.Close)

	dial := func(counter *atomic.Int64) DialContextFunc {
		return func(ctx context.Context, network, address string) (net.Conn, error) {
			counter.Add(1)

			conn, derr := (&net.Dialer{}).DialContext(ctx, network, address)
			assert.NoError(t, derr)

			return conn, derr //nolint:wrapcheck
		}
	}

	c1 := New(WithDialContext(dial(&calls1)))
	c2 := New(WithDialContext(dial(&calls2)))

	do := func(c *Client) {
		req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, rerr)

		resp, rerr := c.Do(req)
		require.NoError(t, rerr)

		_, rerr = io.Copy(io.Discard, resp.Body)
		require.NoError(t, rerr)
		require.NoError(t, resp.Body.Close())
	}

	do(c1)

	require.Equal(t, int64(1), calls1.Load(), "client 1 dialer should be used")
	require.Equal(t, int64(0), calls2.Load(), "client 2 dialer must not be used by client 1")

	do(c2)

	require.Equal(t, int64(1), calls1.Load(), "client 1 dialer must not be used by client 2")
	require.Equal(t, int64(1), calls2.Load(), "client 2 dialer should be used")

	// The global transport's dialer must be unchanged by any client option.
	require.Equal(t,
		reflect.ValueOf(defaultDialer).Pointer(),
		reflect.ValueOf(http.DefaultTransport.(*http.Transport).DialContext).Pointer(), //nolint:forcetypeassert
		"http.DefaultTransport.DialContext must remain untouched",
	)
}

func TestWithDialContext_RoundTripperOrdering(t *testing.T) {
	t.Parallel()

	// When WithRoundTripper wraps the transport first, the transport is no
	// longer an *http.Transport, so a later WithDialContext is a silent no-op
	// (the cloned transport keeps its default dialer).
	wrap := func(next http.RoundTripper) http.RoundTripper {
		return &passthroughRoundTripper{next: next}
	}
	dialed := func(_ context.Context, _, _ string) (net.Conn, error) {
		return nil, errors.New("custom dialer")
	}
	customPtr := reflect.ValueOf(DialContextFunc(dialed)).Pointer()

	defaultDialer := http.DefaultTransport.(*http.Transport).DialContext //nolint:forcetypeassert
	defaultPtr := reflect.ValueOf(defaultDialer).Pointer()

	c := New(WithRoundTripper(wrap), WithDialContext(dialed))

	rt, ok := c.client.Transport.(*passthroughRoundTripper)
	require.True(t, ok, "transport should be the round-tripper wrapper")

	inner, ok := rt.next.(*http.Transport)
	require.True(t, ok, "wrapped transport should still be the cloned *http.Transport")
	require.Equal(t, defaultPtr, reflect.ValueOf(inner.DialContext).Pointer(),
		"WithDialContext after WithRoundTripper must be a no-op (dialer unchanged)")

	// Applying WithDialContext before WithRoundTripper does take effect.
	c2 := New(WithDialContext(dialed), WithRoundTripper(wrap))

	rt2, ok := c2.client.Transport.(*passthroughRoundTripper)
	require.True(t, ok)

	inner2, ok := rt2.next.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, customPtr, reflect.ValueOf(inner2.DialContext).Pointer(),
		"WithDialContext before WithRoundTripper must take effect (custom dialer set)")
}

// passthroughRoundTripper wraps another RoundTripper without delegating to an
// *http.Transport at the top level.
type passthroughRoundTripper struct {
	next http.RoundTripper
}

func (t *passthroughRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.next.RoundTrip(r) //nolint:wrapcheck
}
