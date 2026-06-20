package statsd

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	libhttputil "github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/statsd"
)

const (
	// defaultStatsPrefix is the StatsD client's string prefix that will be used in every bucket name.
	defaultStatsPrefix = ""

	// defaultStatsNetwork is the network type used by the StatsD client (i.e. udp or tcp).
	defaultStatsNetwork = "udp"

	// defaultStatsAddress is the network address of the StatsD daemon (ip:port) or just (:port).
	defaultStatsAddress = ":8125"

	// defaultStatsFlushPeriod sets how often the StatsD client's buffer is flushed.
	// When 0 the buffer is only flushed when it is full.
	defaultStatsFlushPeriod = 100 * time.Millisecond

	labelCount        = "count"
	labelError        = "error"
	labelIn           = "in"
	labelInbound      = "inbound"
	labelLevel        = "level"
	labelLog          = "log"
	labelOut          = "out"
	labelOutbound     = "outbound"
	labelRequestSize  = "request_size"
	labelResponseSize = "response_size"
	labelSeparator    = "."
	labelTime         = "time"
)

// Client is a StatsD-backed implementation of the shared metrics interface.
//
// Construct it with [New].
type Client struct {
	statsd      *statsd.Client
	prefix      string        // StatsD client's string prefix that will be used in every bucket name.
	network     string        // Network type used by the StatsD client (i.e. udp or tcp).
	address     string        // Network address of the StatsD daemon (ip:port) or just (:port).
	flushPeriod time.Duration // How often the StatsD client's buffer is flushed.
}

// New creates a StatsD metrics client with defaults, then applies options.
//
// Defaults:
//   - network: udp
//   - address: :8125
//   - prefix:  ""
//   - flush:   100 ms
func New(opts ...Option) (*Client, error) {
	c := defaultClient()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	var err error

	c.statsd, err = statsd.New(
		statsd.Prefix(c.prefix),
		statsd.Network(c.network),
		statsd.Address(c.address),
		statsd.FlushPeriod(c.flushPeriod),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the StatsD client: %w", err)
	}

	return c, nil
}

// defaultClient returns a Client instance with default values.
func defaultClient() *Client {
	return &Client{
		prefix:      defaultStatsPrefix,
		network:     defaultStatsNetwork,
		address:     defaultStatsAddress,
		flushPeriod: defaultStatsFlushPeriod,
	}
}

// SqlOpen delegates to sql.Open.
// StatsD does not instrument database/sql at the driver level in this package.
func (c *Client) SqlOpen(driverName, dsn string) (*sql.DB, error) {
	return sql.Open(driverName, dsn) //nolint:wrapcheck
}

// InstrumentDB is currently a no-op for the StatsD backend.
func (c *Client) InstrumentDB(_ string, _ *sql.DB) error {
	return nil
}

// InstrumentHandler wraps an http.Handler to collect StatsD metrics.
func (c *Client) InstrumentHandler(path string, handler http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := c.statsd.NewTiming()
		labelInboundPath := labelInbound + labelSeparator + path + labelSeparator + r.Method + labelSeparator

		c.statsd.Increment(labelInboundPath + labelIn)
		defer c.statsd.Increment(labelInboundPath + labelOut)

		reqSize := requestSize(r)
		rw := libhttputil.NewResponseWriterWrapper(w)

		defer func() {
			status := rw.Status()
			if status == 0 {
				status = http.StatusOK
			}

			labelStatus := labelInboundPath + strconv.Itoa(status) + labelSeparator
			c.statsd.Increment(labelStatus + labelCount)
			c.statsd.Gauge(labelStatus+labelRequestSize, reqSize)
			c.statsd.Gauge(labelStatus+labelResponseSize, rw.Size())
			t.Send(labelStatus + labelTime)
		}()

		handler.ServeHTTP(rw, r)
	})
}

// requestSize approximates the size of an inbound HTTP request in bytes without
// buffering its body.
//
// It sums an approximation of the request-line and header bytes with the body
// length taken from r.ContentLength when it is known (>= 0). A negative
// ContentLength means the body length is unknown (e.g. chunked transfer), in
// which case only the metadata size is reported rather than draining the body.
func requestSize(r *http.Request) int {
	size := len(r.Method) + len(r.URL.RequestURI()) + len(r.Proto) + len("  \r\n")

	if r.Host != "" {
		size += len("Host: ") + len(r.Host) + len("\r\n")
	}

	for name, values := range r.Header {
		for _, value := range values {
			size += len(name) + len(": ") + len(value) + len("\r\n")
		}
	}

	if r.ContentLength > 0 {
		size += int(r.ContentLength)
	}

	return size
}

// InstrumentRoundTripper wraps next to emit outbound HTTP metrics.
//
// For successful requests it records in/out counters, status counts, and
// request duration timing grouped by method and status code.
func (c *Client) InstrumentRoundTripper(next http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		t := c.statsd.NewTiming()
		labelOutboundPath := labelOutbound + labelSeparator + r.Method + labelSeparator

		c.statsd.Increment(labelOutboundPath + labelIn)
		defer c.statsd.Increment(labelOutboundPath + labelOut)

		resp, err := next.RoundTrip(r)
		if err == nil {
			labelStatus := labelOutboundPath + strconv.Itoa(resp.StatusCode) + labelSeparator

			c.statsd.Increment(labelStatus + labelCount)
			defer t.Send(labelStatus + labelTime)
		}

		return resp, err //nolint:wrapcheck
	})
}

// MetricsHandlerFunc returns an HTTP handler for a metrics endpoint.
//
// StatsD is push-based in this implementation, so the handler always responds
// with HTTP 501 Not Implemented.
func (c *Client) MetricsHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		status := http.StatusNotImplemented
		http.Error(w, http.StatusText(status), status)
	}
}

// IncLogLevelCounter counts the number of errors for each log severity level.
func (c *Client) IncLogLevelCounter(level string) {
	c.statsd.Increment(labelLog + labelSeparator + labelLevel + labelSeparator + level)
}

// IncErrorCounter increments the number of errors by task, operation and error code.
func (c *Client) IncErrorCounter(task, operation, code string) {
	c.statsd.Increment(labelError + labelSeparator + task + labelSeparator + operation + labelSeparator + code)
}

// Close flushes and closes the underlying StatsD client.
func (c *Client) Close() error {
	c.statsd.Close()
	return nil
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (rt roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}
