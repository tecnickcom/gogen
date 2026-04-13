package httpclient

import (
	"context"
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

// defaultClient() returns a default client.
func defaultClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout:   1 * time.Minute,
			Transport: http.DefaultTransport,
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

// Do executes the request with trace ID attachment, structured logging, and optional debug payload dumps; returns error from underlying client.
//
//nolint:gocognit
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	reqTime := time.Now().UTC()

	//nolint:govet,gosec // calling cancel() causes long body reads to return context canceled errors.
	ctx, _ := context.WithTimeout(r.Context(), c.client.Timeout)

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

	resp, err = c.client.Do(r) //nolint:gosec

	if debug && resp != nil {
		respDump, errd := httputil.DumpResponse(resp, true)
		if errd == nil {
			l = l.With(slog.String(c.logPrefix+"response", c.redactFn(respDump)))
		}
	}

	return resp, err //nolint:wrapcheck
}
