package httpclient

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/traceid"
	"github.com/nexmoinc/gosrvlib/pkg/uidc"
	"go.uber.org/zap"
)

// Client wraps the default HTTP client functionalities and adds logging and instrumentation capabilities.
type Client struct {
	client            *http.Client
	traceIDHeaderName string
	component         string
}

// New creates a new HTTP client instance.
func New(opts ...Option) *Client {
	c := defaultClient()
	for _, applyOpt := range opts {
		applyOpt(c)
	}
	return c
}

// defaultClient() returns a default client.
func defaultClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 1 * time.Minute,
		},
		traceIDHeaderName: traceid.DefaultHeader,
		component:         "-",
	}
}

// Do performs the HTTP request with added trace ID, logging and metrics.
func (c *Client) Do(r *http.Request) (resp *http.Response, err error) {
	ctx := r.Context()

	l := logging.WithComponent(ctx, c.component)
	debug := l.Check(zap.DebugLevel, "debug") != nil

	defer func() {
		if err != nil {
			l.Error("error", zap.Error(err))
			return
		}
		if debug {
			l.Debug("outbound")
			return
		}
		l.Info("outbound")
	}()

	reqID := traceid.FromContext(ctx, uidc.NewID128())
	ctx = traceid.NewContext(ctx, reqID)
	r.Header.Set(c.traceIDHeaderName, reqID)
	r = r.WithContext(ctx)

	l = l.With(
		zap.String("traceid", reqID),
		zap.String("request_method", r.Method),
		zap.String("request_path", r.URL.Path),
		zap.String("request_query", r.URL.RawQuery),
		zap.String("request_uri", r.RequestURI),
	)

	if debug {
		reqDump, _ := httputil.DumpRequestOut(r, true)
		l = l.With(zap.String("request", string(reqDump)))
	}

	start := time.Now()
	resp, err = c.client.Do(r)
	l = l.With(zap.Duration("duration", time.Since(start)))

	if resp != nil {
		if debug {
			respDump, _ := httputil.DumpResponse(resp, true)
			l = l.With(zap.String("response", string(respDump)))
		}
		_ = resp.Body.Close()
	}

	return resp, err
}
