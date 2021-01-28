package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Client represents the state type of this client.
type Client struct {
	Registry    *prometheus.Registry
	handlerOpts promhttp.HandlerOpts
}

// New creates a new metrics instance.
func New(opts ...Option) (*Client, error) {
	c := initClient()
	err := c.Configure(opts...)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Configure allows to specify more options.
func (c *Client) Configure(opts ...Option) error {
	for _, applyOpt := range opts {
		if err := applyOpt(c); err != nil {
			return err
		}
	}
	return nil
}

func initClient() *Client {
	return &Client{
		Registry:    prometheus.NewRegistry(),
		handlerOpts: promhttp.HandlerOpts{},
	}
}

// InstrumentHandler wraps an http.Handler to collect Prometheus metrics.
func (c *Client) InstrumentHandler(path string, handler http.HandlerFunc) http.Handler {
	var h http.Handler
	h = handler
	h = promhttp.InstrumentHandlerRequestSize(collectorRequestSize, h)
	h = promhttp.InstrumentHandlerResponseSize(collectorResponseSize, h)
	h = promhttp.InstrumentHandlerCounter(collectorAPIRequests, h)
	h = promhttp.InstrumentHandlerDuration(collectorRequestDuration.MustCurryWith(prometheus.Labels{labelHandler: path}), h)
	h = promhttp.InstrumentHandlerInFlight(collectorInFlightRequests, h)
	return h
}

// InstrumentRoundTripper is a middleware that wraps the provided http.RoundTripper to observe the request result with default metrics.
func (c *Client) InstrumentRoundTripper(next http.RoundTripper) http.RoundTripper {
	next = promhttp.InstrumentRoundTripperCounter(collectorOutboundRequests, next)
	next = promhttp.InstrumentRoundTripperDuration(collectorOutboundRequestsDuration, next)
	next = promhttp.InstrumentRoundTripperInFlight(collectorOutboundInFlightRequests, next)
	return next
}

// MetricsHandlerFunc returns an http handler function to serve the metrics endpoint.
func (c *Client) MetricsHandlerFunc() http.HandlerFunc {
	h := promhttp.HandlerFor(c.Registry, c.handlerOpts)
	return promhttp.InstrumentMetricHandler(c.Registry, h).ServeHTTP
}

// IncLogLevelCounter counts the number of errors for each log severity level.
func (c *Client) IncLogLevelCounter(level string) {
	collectorErrorLevel.With(prometheus.Labels{labelLevel: level}).Inc()
}

// IncErrorCounter increments the number of errors by task, operation and error code.
func (c *Client) IncErrorCounter(task, operation, code string) {
	collectorErrorCode.With(prometheus.Labels{labelTask: task, labelOperation: operation, labelCode: code}).Inc()
}
