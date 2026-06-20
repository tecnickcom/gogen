package httpclient

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/tecnickcom/gogen/pkg/random"
	"github.com/tecnickcom/gogen/pkg/redact"
	"github.com/tecnickcom/gogen/pkg/traceid"
)

// Client wraps http.Client with trace ID propagation, structured request/response logging, and optional debug payload dumps.
type Client struct {
	client            *http.Client
	component         string
	logPrefix         string
	traceIDHeaderName string
	redactFn          RedactFn
	logger            *slog.Logger
	rnd               *random.Rnd
}

// defaultTransport returns a private transport for a new client.
// It clones http.DefaultTransport so that per-client options (e.g. WithDialContext)
// never mutate the process-wide http.DefaultTransport, which would race across
// concurrent New calls and change the dialer for every other consumer.
func defaultTransport() http.RoundTripper {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// http.DefaultTransport is always an *http.Transport in the standard
		// library; this guards against a future change or test override.
		return http.DefaultTransport
	}

	return t.Clone()
}

// defaultClient() returns a default client.
func defaultClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout:   1 * time.Minute,
			Transport: defaultTransport(),
		},
		traceIDHeaderName: traceid.DefaultHeader,
		component:         "-",
		redactFn:          redact.HTTPDataString,
		logger:            slog.Default(),
		rnd:               random.New(nil),
	}
}

// New constructs an HTTP client with 1-minute timeout, trace ID propagation, and request/response logging from provided options.
func New(opts ...Option) *Client {
	c := defaultClient()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	return c
}

// cancelReadCloser wraps a response body so that closing it also releases the
// per-request timeout context (canceling its timer promptly) while still
// allowing the timeout to interrupt long reads that happen before Close.
type cancelReadCloser struct {
	io.ReadCloser

	cancel context.CancelFunc
}

// Close closes the underlying body and cancels the per-request context.
func (b *cancelReadCloser) Close() error {
	defer b.cancel()

	return b.ReadCloser.Close() //nolint:wrapcheck
}

// Do executes the request with trace ID attachment, structured logging, and optional debug payload dumps; returns error from underlying client.
//
// The returned response body is wrapped so that the per-request timeout context
// is canceled when the body is closed. The caller is therefore responsible for
// closing resp.Body (e.g. defer resp.Body.Close()) on the success path to avoid
// leaking the timeout timer until the timeout elapses; failing to close the body
// also leaks the underlying connection as with the standard net/http client.
//
//nolint:gocognit
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	reqTime := time.Now().UTC()

	// The cancel func is released either when the response body is closed
	// (success path) or immediately when no response is returned (error path);
	// calling it interrupts in-flight body reads with a context-canceled error.
	ctx, cancel := context.WithTimeout(r.Context(), c.client.Timeout)

	l := c.logger.With(c.logPrefix+"component", c.component)
	debug := l.Enabled(ctx, slog.LevelDebug)

	var err error

	defer func() {
		resTime := time.Now().UTC()
		l = l.With(
			slog.Time(c.logPrefix+"response_time", resTime),
			slog.Duration(c.logPrefix+"response_duration", resTime.Sub(reqTime)),
		)

		if err != nil {
			l.With(slog.Any(c.logPrefix+"error", err)).Error(c.logPrefix + "outbound")
			return
		}

		if debug {
			l.Debug(c.logPrefix + "outbound")
			return
		}

		l.Info(c.logPrefix + "outbound")
	}()

	reqID := traceid.FromContext(ctx, c.rnd.UUIDv7().String())
	ctx = traceid.NewContext(ctx, reqID)
	r.Header.Set(c.traceIDHeaderName, reqID)
	r = r.WithContext(ctx)

	l = l.With(
		slog.String(c.logPrefix+traceid.DefaultLogKey, reqID),
		slog.Time(c.logPrefix+"request_time", reqTime),
		slog.String(c.logPrefix+"request_method", r.Method),
		slog.String(c.logPrefix+"request_path", r.URL.Path),
		slog.String(c.logPrefix+"request_query", r.URL.RawQuery),
		slog.String(c.logPrefix+"request_uri", r.RequestURI),
	)

	if debug {
		reqDump, errd := httputil.DumpRequestOut(r, true)
		if errd == nil {
			l = l.With(slog.String(c.logPrefix+"request", c.redactFn(reqDump)))
		}
	}

	var resp *http.Response

	resp, err = c.client.Do(r)

	if debug && resp != nil {
		respDump, errd := httputil.DumpResponse(resp, true)
		if errd == nil {
			l = l.With(slog.String(c.logPrefix+"response", c.redactFn(respDump)))
		}
	}

	if resp == nil {
		// No body to close: release the timeout timer immediately so it does
		// not leak until the full timeout elapses.
		cancel()

		return resp, err //nolint:wrapcheck
	}

	// Tie the lifetime of the timeout context to the response body: closing the
	// body cancels the context (releasing the timer), while leaving it open
	// preserves the timeout's ability to interrupt long reads.
	resp.Body = &cancelReadCloser{ReadCloser: resp.Body, cancel: cancel}

	return resp, err //nolint:wrapcheck
}
