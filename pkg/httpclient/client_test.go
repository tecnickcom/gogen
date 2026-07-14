package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/redact"
	"github.com/tecnickcom/nurago/pkg/traceid"
)

// roundTripperFunc adapts a function to http.RoundTripper for tests.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// errorReader is an io.Reader that always fails, used to force dump errors.
type errorReader struct {
	err error
}

func (e errorReader) Read([]byte) (int, error) {
	return 0, e.err
}

// newDebugLogger returns a text logger at debug level writing to w.
func newDebugLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

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
	require.Equal(t, timeout, got.timeout)
	// The deadline is applied via the request context, not http.Client.Timeout,
	// so the underlying client's timeout stays zero (a single timer per request).
	require.Zero(t, got.client.Timeout)

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

// TestClient_Do_RedactsQueryLogsHostNoURI verifies the outbound log fields:
// query-string secrets are redacted, the destination host is logged, and neither
// the removed request_url nor the always-empty client-side request_uri field is
// present.
func TestClient_Do_RedactsQueryLogsHostNoURI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	t.Cleanup(server.Close)

	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	client := New(WithLogger(logger))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/p?token=SUPERSECRET&x=1", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	out := buf.String()
	require.NotContains(t, out, "SUPERSECRET", "query secret must be redacted before logging")
	require.Contains(t, out, redact.RedactionMarker, "redaction marker must appear in place of the secret")
	require.Contains(t, out, "request_host=")
	require.Contains(t, out, "request_path=")
	require.NotContains(t, out, "request_url=", "the redundant request_url field must be removed")
	require.NotContains(t, out, "request_uri=", "the client-side request_uri field must be removed")
}

// TestClient_Do_NilRequest verifies Do returns ErrNilRequest instead of panicking.
func TestClient_Do_NilRequest(t *testing.T) {
	t.Parallel()

	//nolint:bodyclose // no response is returned on this error path.
	resp, err := New().Do(nil)
	require.Nil(t, resp)
	require.ErrorIs(t, err, ErrNilRequest)
}

// TestClient_Do_NilRequestURL verifies Do returns ErrNilRequestURL instead of
// panicking on a request with a nil URL.
func TestClient_Do_NilRequestURL(t *testing.T) {
	t.Parallel()

	req := &http.Request{Method: http.MethodGet, Header: make(http.Header)}

	//nolint:bodyclose // no response is returned on this error path.
	resp, err := New().Do(req.WithContext(t.Context()))
	require.Nil(t, resp)
	require.ErrorIs(t, err, ErrNilRequestURL)
}

// TestClient_Do_LogsResponseStatus verifies the outbound log carries the response
// status code and, for a known-length response, the size.
func TestClient_Do_LogsResponseStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("nope"))
	}))
	t.Cleanup(server.Close)

	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	client := New(WithLogger(logger))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	out := buf.String()
	require.Contains(t, out, "response_code=503", "the response status code must be logged")
	require.Contains(t, out, "response_content_length=4", "a known response length must be logged")
	require.Regexp(t, `response_duration=[0-9]`, out, "a non-negative duration must be logged")
}

// TestClient_Do_ValidContextTraceIDReused verifies that a valid trace ID already
// in the context is propagated unchanged (exercising the no-rewrap fast path).
func TestClient_Do_ValidContextTraceIDReused(t *testing.T) {
	t.Parallel()

	const ctxID = "ctx-trace-778899"

	var got string

	capture := func(_ http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Header.Get(traceid.DefaultHeader)

			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
		})
	}

	client := New(WithRoundTripper(capture))

	ctx := traceid.NewContext(t.Context(), ctxID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, ctxID, got, "the context trace ID must be propagated unchanged")
}

// TestClient_Do_CallerHeaderHonored verifies that a valid caller-set trace header
// is used (not clobbered) when the context carries no trace ID.
func TestClient_Do_CallerHeaderHonored(t *testing.T) {
	t.Parallel()

	const headerID = "caller-set-661122"

	var got string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(traceid.DefaultHeader)
		_, _ = w.Write([]byte("OK"))
	}))
	t.Cleanup(server.Close)

	client := New()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	req.Header.Set(traceid.DefaultHeader, headerID)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, headerID, got, "a valid caller-set trace header must be honored")
}

// TestClient_Do_InvalidTraceIDContextMatchesHeader verifies that when the
// inbound context carries an invalid trace ID, the ID forced onto the request
// context matches the freshly generated ID propagated in the header.
func TestClient_Do_InvalidTraceIDContextMatchesHeader(t *testing.T) {
	t.Parallel()

	const invalidID = "bad\x00trace\nid"

	type captured struct {
		header string
		ctxID  string
	}

	got := make(chan captured, 1)
	capture := func(_ http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			got <- captured{
				header: r.Header.Get(traceid.DefaultHeader),
				ctxID:  traceid.FromContext(r.Context(), ""),
			}

			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
		})
	}

	client := New(WithRoundTripper(capture))

	ctx := traceid.NewContext(t.Context(), invalidID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	c := <-got
	require.NotEmpty(t, c.header)
	require.NotEqual(t, invalidID, c.header, "the invalid ID must not be propagated")
	require.Equal(t, c.header, c.ctxID, "request context trace ID must match the header sent downstream")
}

// TestClient_Do_DoesNotMutateCallerRequest verifies Do works on a clone: the
// caller's request keeps its original (empty) trace header after Do returns.
func TestClient_Do_DoesNotMutateCallerRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	t.Cleanup(server.Close)

	client := New()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Empty(t, req.Header.Get(traceid.DefaultHeader), "caller's request headers must not be mutated")
}

// TestClient_Do_RoundTripperPanicCancelsContext verifies that a panic in a
// user-supplied round-tripper still releases the per-request timeout context.
func TestClient_Do_RoundTripperPanicCancelsContext(t *testing.T) {
	t.Parallel()

	got := make(chan context.Context, 1)
	panicRT := func(_ http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			got <- r.Context()

			panic("round-trip panic")
		})
	}

	client := New(WithRoundTripper(panicRT))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.invalid", nil)
	require.NoError(t, err)

	require.Panics(t, func() {
		//nolint:bodyclose // the round-tripper panics; there is no response to close.
		_, _ = client.Do(req)
	})

	ctx := <-got

	select {
	case <-ctx.Done():
		require.ErrorIs(t, ctx.Err(), context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("context was not canceled after a round-tripper panic")
	}
}

// TestClient_dumpRequest covers the body-inclusion decision for request dumps.
func TestClient_dumpRequest(t *testing.T) {
	t.Parallel()

	c := defaultClient()

	newReq := func(contentLength int64, body io.ReadCloser) *http.Request {
		r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example.test", nil)
		require.NoError(t, err)

		r.ContentLength = contentLength
		r.Body = body

		return r
	}

	// nil body: headers only, no error.
	d, err := c.dumpRequest(newReq(0, nil))
	require.NoError(t, err)
	require.Contains(t, string(d), "POST")

	// small known body: included.
	d, err = c.dumpRequest(newReq(5, io.NopCloser(strings.NewReader("hello"))))
	require.NoError(t, err)
	require.Contains(t, string(d), "hello")

	// unknown length (streaming): body omitted, no deadlock.
	d, err = c.dumpRequest(newReq(-1, io.NopCloser(strings.NewReader("streamed"))))
	require.NoError(t, err)
	require.NotContains(t, string(d), "streamed")

	// known but over the cap: body omitted.
	c.maxDumpSize = 4
	d, err = c.dumpRequest(newReq(100, io.NopCloser(strings.NewReader("toolongbody"))))
	require.NoError(t, err)
	require.NotContains(t, string(d), "toolongbody")
}

// TestClient_dumpResponseBody covers the size-cap decision for known-length
// response dumps.
func TestClient_dumpResponseBody(t *testing.T) {
	t.Parallel()

	c := defaultClient() // default cap 1 MiB

	require.True(t, c.dumpResponseBody(0))
	require.True(t, c.dumpResponseBody(100))
	require.False(t, c.dumpResponseBody(2<<20), "over-cap response body is omitted")

	c.maxDumpSize = 0 // cap disabled
	require.True(t, c.dumpResponseBody(2<<20))
}

// newResp builds a minimal response suitable for httputil.DumpResponse.
func newResp(contentLength int64, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          body,
		ContentLength: contentLength,
	}
}

// TestClient_dumpResponse covers the response-dump body-capping decisions,
// including the hard cap applied to unknown-length (chunked) bodies.
func TestClient_dumpResponse(t *testing.T) {
	t.Parallel()

	t.Run("known length within cap includes body", func(t *testing.T) {
		t.Parallel()

		c := defaultClient()
		resp := newResp(5, io.NopCloser(strings.NewReader("hello")))

		dump, err := c.dumpResponse(resp)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Contains(t, string(dump), "hello")
	})

	t.Run("known length over cap omits body", func(t *testing.T) {
		t.Parallel()

		c := defaultClient()
		c.maxDumpSize = 4
		resp := newResp(20, io.NopCloser(strings.NewReader("way-too-long-body")))

		dump, err := c.dumpResponse(resp)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.NotContains(t, string(dump), "way-too-long-body")
	})

	t.Run("unknown length with cap disabled dumps full body", func(t *testing.T) {
		t.Parallel()

		c := defaultClient()
		c.maxDumpSize = 0
		resp := newResp(-1, io.NopCloser(strings.NewReader("chunkeddata")))

		dump, err := c.dumpResponse(resp)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Contains(t, string(dump), "chunkeddata")
	})

	t.Run("unknown length under cap dumps full body and preserves it", func(t *testing.T) {
		t.Parallel()

		c := defaultClient()
		c.maxDumpSize = 100
		resp := newResp(-1, io.NopCloser(strings.NewReader("small")))

		dump, err := c.dumpResponse(resp)
		require.NoError(t, err)
		require.Contains(t, string(dump), "small")
		require.NotContains(t, string(dump), "truncated")

		got, rerr := io.ReadAll(resp.Body)
		require.NoError(t, rerr)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, "small", string(got))
	})

	t.Run("unknown length over cap truncates but preserves full body", func(t *testing.T) {
		t.Parallel()

		c := defaultClient()
		c.maxDumpSize = 4
		resp := newResp(-1, io.NopCloser(strings.NewReader("HELLOWORLD")))

		dump, err := c.dumpResponse(resp)
		require.NoError(t, err)
		require.Contains(t, string(dump), "truncated", "over-cap chunked body must be marked truncated")
		require.NotContains(t, string(dump), "WORLD", "bytes beyond the cap must not appear in the dump")

		// The caller still receives the complete body.
		got, rerr := io.ReadAll(resp.Body)
		require.NoError(t, rerr)
		require.Equal(t, "HELLOWORLD", string(got))
		require.NoError(t, resp.Body.Close())
	})

	t.Run("unknown length read error is returned", func(t *testing.T) {
		t.Parallel()

		c := defaultClient()
		c.maxDumpSize = 100
		resp := newResp(-1, io.NopCloser(errorReader{err: errors.New("stream boom")}))

		_, err := c.dumpResponse(resp)
		require.Error(t, err)
		require.NoError(t, resp.Body.Close())
	})

	t.Run("extreme cap does not overflow", func(t *testing.T) {
		t.Parallel()

		c := defaultClient()
		c.maxDumpSize = math.MaxInt64 // cap+1 would overflow without the clamp
		resp := newResp(-1, io.NopCloser(strings.NewReader("tiny")))

		dump, err := c.dumpResponse(resp)
		require.NoError(t, err)
		require.Contains(t, string(dump), "tiny")
		require.NotContains(t, string(dump), "truncated")
		require.NoError(t, resp.Body.Close())
	})
}

// TestClient_Do_ChunkedResponseTruncated exercises the unknown-length cap end to
// end: the dump is truncated while the caller still reads the full body.
func TestClient_Do_ChunkedResponseTruncated(t *testing.T) {
	t.Parallel()

	const bodyText = "FIRSTPARTsecretSECONDPART"

	var buf bytes.Buffer

	rt := func(_ http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			resp := newResp(-1, io.NopCloser(strings.NewReader(bodyText))) // unknown length
			resp.Request = r

			return resp, nil
		})
	}

	client := New(WithLogger(newDebugLogger(&buf)), WithRoundTripper(rt), WithMaxDumpSize(5))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, bodyText, string(got), "caller must receive the full body despite the dump cap")

	out := buf.String()
	require.Contains(t, out, "truncated", "the dump must be marked truncated")
	require.NotContains(t, out, "SECONDPART", "content past the cap must not be dumped")
}

// TestClient_Do_RequestDumpError verifies a failed request dump is surfaced as a
// log field instead of being silently discarded.
func TestClient_Do_RequestDumpError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	client := New(WithLogger(newDebugLogger(&buf)))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "http://example.invalid", errorReader{err: errors.New("read boom")})
	require.NoError(t, err)

	req.ContentLength = 5 // known, small: dump attempts to drain the failing body

	resp, err := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.Error(t, err)
	require.Contains(t, buf.String(), "request_dump_error")
}

// TestClient_Do_ResponseDumpError verifies a failed response dump is surfaced as
// a log field while the request itself still succeeds.
func TestClient_Do_ResponseDumpError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	rt := func(_ http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				ProtoMajor:    1,
				ProtoMinor:    1,
				Header:        make(http.Header),
				Body:          io.NopCloser(errorReader{err: errors.New("body boom")}),
				ContentLength: 5,
				Request:       r,
			}, nil
		})
	}

	client := New(WithLogger(newDebugLogger(&buf)), WithRoundTripper(rt))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, resp.Body.Close())

	require.Contains(t, buf.String(), "response_dump_error")
}

// TestClient_Do_LargeResponseBodyOmitted verifies that a response whose known
// length exceeds the cap has its headers dumped but its body omitted, leaving
// the real body intact for the caller.
func TestClient_Do_LargeResponseBodyOmitted(t *testing.T) {
	t.Parallel()

	const bodyText = "REALRESPONSEBODY"

	var buf bytes.Buffer

	rt := func(_ http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				ProtoMajor:    1,
				ProtoMinor:    1,
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader(bodyText)),
				ContentLength: 2 << 20, // advertised length over the default cap
				Request:       r,
			}, nil
		})
	}

	client := New(WithLogger(newDebugLogger(&buf)), WithRoundTripper(rt))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	out := buf.String()
	require.Contains(t, out, "response=", "response headers should still be dumped")
	require.NotContains(t, out, bodyText, "an over-cap response body must be omitted from the dump")

	// The real body is preserved for the caller.
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, bodyText, string(got))
}

// TestClient_CloseIdleConnections exercises the pass-through to the transport.
func TestClient_CloseIdleConnections(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		New().CloseIdleConnections()
	})
}

// TestClient_Do_ErrorFieldRedactsQuery verifies that a failed request does not
// leak query-string secrets through the logged error field, while the error
// returned to the caller stays intact.
func TestClient_Do_ErrorFieldRedactsQuery(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	client := New(WithLogger(logger), WithTimeout(2*time.Second))

	// Loopback port 1 refuses immediately, so the request fails at the transport.
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet, "http://127.0.0.1:1/p?api_key=SUPERSECRET&x=1", nil)
	require.NoError(t, err)

	//nolint:bodyclose // the request fails; there is no response body to close.
	resp, derr := client.Do(req)
	require.Error(t, derr)
	require.Nil(t, resp)

	out := buf.String()
	require.NotContains(t, out, "SUPERSECRET", "the error field must not leak the query secret")
	require.Contains(t, out, "error=", "an error field must be logged")

	// The error returned to the caller is unredacted (callers may need the URL).
	require.Contains(t, derr.Error(), "SUPERSECRET", "the returned error must be left unchanged")
}

// TestClient_Do_InvalidCallerHeaderReplaced verifies that an invalid caller-set
// trace header is not propagated: it is replaced with a freshly generated ID.
func TestClient_Do_InvalidCallerHeaderReplaced(t *testing.T) {
	t.Parallel()

	const injected = "bad\x00id\ninjected"

	var got string

	capture := func(_ http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Header.Get(traceid.DefaultHeader)

			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
		})
	}

	client := New(WithRoundTripper(capture))

	// Context has no trace ID and the request carries an invalid one.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)
	req.Header.Set(traceid.DefaultHeader, injected)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.NotEqual(t, injected, got, "an invalid caller header must not be propagated")
	require.Regexp(t, `^[0-9A-Za-z\-\_\.]{1,64}$`, got, "a valid generated ID must be sent instead")
}

// TestClient_redactErrorForLog covers the error-redaction helper.
func TestClient_redactErrorForLog(t *testing.T) {
	t.Parallel()

	c := defaultClient()

	// A non-*url.Error is returned unchanged.
	plain := errors.New("plain failure")
	require.Equal(t, plain, c.redactErrorForLog(plain))

	// A *url.Error has its URL query and userinfo redacted, reason preserved.
	// The userinfo is assembled from parts so the literal is not flagged as a
	// hardcoded credential; it is a test fixture, not a real secret.
	userinfo := "user" + ":" + "pw"
	uerr := &url.Error{
		Op:  "Get",
		URL: "http://" + userinfo + "@example.test/p?token=SUPERSECRET&x=1",
		Err: errors.New("connection refused"),
	}
	msg := c.redactErrorForLog(uerr).Error()
	require.NotContains(t, msg, "SUPERSECRET")
	require.NotContains(t, msg, "user")
	require.NotContains(t, msg, "pw")
	require.Contains(t, msg, redact.RedactionMarker)
	require.Contains(t, msg, "connection refused", "the failure reason must be preserved")

	// The original error is untouched (only a copy is redacted).
	require.Contains(t, uerr.URL, "SUPERSECRET")
}

// TestClient_redactURLForLog covers the URL-redaction branches.
func TestClient_redactURLForLog(t *testing.T) {
	t.Parallel()

	c := defaultClient()

	got := c.redactURLForLog("http://h/p?token=SECRET&x=1")
	require.NotContains(t, got, "SECRET")
	require.Contains(t, got, redact.RedactionMarker)

	// No query and no userinfo: unchanged.
	require.Equal(t, "http://h/p", c.redactURLForLog("http://h/p"))

	// Userinfo (which may carry a token) is dropped.
	got = c.redactURLForLog("http://tok:pw@h/p")
	require.NotContains(t, got, "tok")
	require.NotContains(t, got, "pw")

	// An unparseable URL still has its query redacted via the fallback.
	got = c.redactURLForLog("://bad?token=SECRET")
	require.NotContains(t, got, "SECRET")
	require.Contains(t, got, redact.RedactionMarker)
}

// TestRedactQueryTail covers the split-based fallback used for unparseable URLs.
func TestRedactQueryTail(t *testing.T) {
	t.Parallel()

	// No query: returned unchanged (the redact function is not applied).
	require.Equal(t, "no-query-here", redactQueryTail("no-query-here", redact.Default().BytesToString))

	// With a query: the tail is redacted.
	got := redactQueryTail("p?token=SECRET", redact.Default().BytesToString)
	require.NotContains(t, got, "SECRET")
	require.Contains(t, got, redact.RedactionMarker)
}

// TestNew_TransportPoolTuned verifies the default transport raises the per-host
// idle-connection pool above net/http's default of 2.
func TestNew_TransportPoolTuned(t *testing.T) {
	t.Parallel()

	tr, ok := New().client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, defaultMaxIdleConnsPerHost, tr.MaxIdleConnsPerHost)
	require.Greater(t, tr.MaxIdleConnsPerHost, 2, "per-host idle pool must exceed net/http's default of 2")
}

// TestWithTransport_ThenDialContext verifies the documented ordering: a dialer
// set after WithTransport lands on the transport installed by WithTransport.
func TestWithTransport_ThenDialContext(t *testing.T) {
	t.Parallel()

	dial := func(_ context.Context, _, _ string) (net.Conn, error) { return nil, errors.New("TEST") }

	c := New(WithTransport(&http.Transport{}), WithDialContext(dial))

	tr, ok := c.client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, tr.DialContext, "WithDialContext after WithTransport must set the dialer on the new base")

	_, err := tr.DialContext(t.Context(), "", "")
	require.Error(t, err)
}

// TestClient_Do_TimeoutFires verifies that WithTimeout actually aborts a slow
// request with a deadline error.
func TestClient_Do_TimeoutFires(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until released or the client disconnects (its timeout fires).
		select {
		case <-release:
		case <-r.Context().Done():
		}

		w.WriteHeader(http.StatusOK)
	}))

	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	client := New(WithTimeout(100 * time.Millisecond))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	//nolint:bodyclose // the request times out; there is no response body.
	resp, derr := client.Do(req)
	require.Error(t, derr)
	require.Nil(t, resp)
	require.ErrorIs(t, derr, context.DeadlineExceeded)
}
