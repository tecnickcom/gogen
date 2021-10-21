package prometheus

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// NameAPIRequests is the name of the collector that counts the total inbound http requests.
	NameAPIRequests = "api_requests_total"

	// NameInFlightRequests is the name of the collector that counts in-flight inbound http requests.
	NameInFlightRequests = "in_flight_requests"

	// NameRequestDuration is the name of the collector that measures the inbound http request duration in seconds.
	NameRequestDuration = "request_duration_seconds"

	// NameRequestSize is the name of the collector that measures the http request size in bytes.
	NameRequestSize = "request_size_bytes"

	// NameResponseSize is the name of the collector that measures the http response size in bytes.
	NameResponseSize = "response_size_bytes"

	// NameOutboundRequests is the name of the collector that measures the number of outbound requests.
	NameOutboundRequests = "outbound_requests_total"

	// NameOutboundRequestsDuration is the name of the collector that measures the outbound requests duration in seconds.
	NameOutboundRequestsDuration = "outbound_request_duration_seconds"

	// NameOutboundInFlightRequests is the name of the collector that counts in-flight outbound http requests.
	NameOutboundInFlightRequests = "outbound_in_flight_requests"

	// NameErrorLevel is the name of the collector that counts the number of errors for each log severity level.
	NameErrorLevel = "error_level_total"

	// NameErrorCode is the name of the collector that counts the number of errors by task, operation and error code.
	NameErrorCode = "error_code_total"

	labelCode      = "code"
	labelHandler   = "handler"
	labelLevel     = "level"
	labelMethod    = "method"
	labelOperation = "operation"
	labelTask      = "task"
)

// Client represents the state type of this client.
type Client struct {
	registry                          *prometheus.Registry
	handlerOpts                       promhttp.HandlerOpts
	inboundRequestSizeBuckets         []float64
	inboundResponseSizeBuckets        []float64
	inboundRequestDurationBuckets     []float64
	outboundRequestDurationBuckets    []float64
	collectorInFlightRequests         prometheus.Gauge
	collectorAPIRequests              *prometheus.CounterVec
	collectorRequestDuration          *prometheus.HistogramVec
	collectorResponseSize             *prometheus.HistogramVec
	collectorRequestSize              *prometheus.HistogramVec
	collectorOutboundRequests         *prometheus.CounterVec
	collectorOutboundRequestsDuration *prometheus.HistogramVec
	collectorOutboundInFlightRequests prometheus.Gauge
	collectorErrorLevel               *prometheus.CounterVec
	collectorErrorCode                *prometheus.CounterVec
}

// New creates a new metrics instance with default collectors.
func New(opts ...Option) (*Client, error) {
	c := initClient()

	for _, applyOpt := range opts {
		if err := applyOpt(c); err != nil {
			return nil, err
		}
	}

	if err := c.defaultCollectors(); err != nil {
		return nil, err
	}

	return c, nil
}

func initClient() *Client {
	return &Client{
		registry:                       prometheus.NewRegistry(),
		handlerOpts:                    promhttp.HandlerOpts{},
		inboundRequestSizeBuckets:      prometheus.ExponentialBuckets(100, 10, 6),
		inboundResponseSizeBuckets:     prometheus.ExponentialBuckets(100, 10, 6),
		inboundRequestDurationBuckets:  prometheus.ExponentialBuckets(0.001, 10, 6),
		outboundRequestDurationBuckets: prometheus.ExponentialBuckets(0.001, 10, 6),
	}
}

func (c *Client) defaultCollectors() error {
	c.collectorInFlightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: NameInFlightRequests,
			Help: "Number of In-flight http requests.",
		},
	)

	c.collectorAPIRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameAPIRequests,
			Help: "Total number of http requests.",
		},
		[]string{labelCode, labelMethod},
	)

	c.collectorRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameRequestDuration,
			Help:    "Requests duration in seconds.",
			Buckets: c.inboundRequestDurationBuckets,
		},
		[]string{labelHandler, labelMethod},
	)

	c.collectorResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameResponseSize,
			Help:    "Response size in bytes.",
			Buckets: c.inboundResponseSizeBuckets,
		},
		[]string{},
	)

	c.collectorRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameRequestSize,
			Help:    "Requests size in bytes.",
			Buckets: c.inboundRequestSizeBuckets,
		},
		[]string{},
	)

	c.collectorOutboundRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameOutboundRequests,
			Help: "Total number of outbound http requests.",
		},
		[]string{labelCode, labelMethod},
	)

	c.collectorOutboundRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameOutboundRequestsDuration,
			Help:    "Outbound requests duration in seconds.",
			Buckets: c.outboundRequestDurationBuckets,
		},
		[]string{labelCode, labelMethod},
	)

	c.collectorOutboundInFlightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: NameOutboundInFlightRequests,
			Help: "Number of outbound In-flight http requests.",
		},
	)

	c.collectorErrorLevel = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameErrorLevel,
			Help: "Number of errors by severity level.",
		},
		[]string{labelLevel},
	)

	c.collectorErrorCode = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameErrorCode,
			Help: "Number of errors by task, operation and error code.",
		},
		[]string{labelTask, labelOperation, labelCode},
	)

	colls := []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		c.collectorInFlightRequests,
		c.collectorAPIRequests,
		c.collectorRequestDuration,
		c.collectorResponseSize,
		c.collectorRequestSize,
		c.collectorOutboundRequests,
		c.collectorOutboundRequestsDuration,
		c.collectorOutboundInFlightRequests,
		c.collectorErrorLevel,
		c.collectorErrorCode,
	}

	for _, m := range colls {
		if err := c.registry.Register(m); err != nil {
			return fmt.Errorf("failed registering collector: %w", err)
		}
	}

	return nil
}

// InstrumentDB wraps a sql.DB to collect metrics.
func (c *Client) InstrumentDB(dbName string, db *sql.DB) {
	coll := sqlstats.NewStatsCollector(dbName, db)
	c.registry.MustRegister(coll)
}

// InstrumentHandler wraps an http.Handler to collect Prometheus metrics.
func (c *Client) InstrumentHandler(path string, handler http.HandlerFunc) http.Handler {
	var h http.Handler
	h = promhttp.InstrumentHandlerRequestSize(c.collectorRequestSize, handler)
	h = promhttp.InstrumentHandlerResponseSize(c.collectorResponseSize, h)
	h = promhttp.InstrumentHandlerCounter(c.collectorAPIRequests, h)
	h = promhttp.InstrumentHandlerDuration(c.collectorRequestDuration.MustCurryWith(prometheus.Labels{labelHandler: path}), h)
	h = promhttp.InstrumentHandlerInFlight(c.collectorInFlightRequests, h)

	return h
}

// InstrumentRoundTripper is a middleware that wraps the provided http.RoundTripper to observe the request result with default metrics.
func (c *Client) InstrumentRoundTripper(next http.RoundTripper) http.RoundTripper {
	next = promhttp.InstrumentRoundTripperCounter(c.collectorOutboundRequests, next)
	next = promhttp.InstrumentRoundTripperDuration(c.collectorOutboundRequestsDuration, next)
	next = promhttp.InstrumentRoundTripperInFlight(c.collectorOutboundInFlightRequests, next)

	return next
}

// MetricsHandlerFunc returns an http handler function to serve the metrics endpoint.
func (c *Client) MetricsHandlerFunc() http.HandlerFunc {
	h := promhttp.HandlerFor(c.registry, c.handlerOpts)
	return promhttp.InstrumentMetricHandler(c.registry, h).ServeHTTP
}

// IncLogLevelCounter counts the number of errors for each log severity level.
func (c *Client) IncLogLevelCounter(level string) {
	c.collectorErrorLevel.With(prometheus.Labels{labelLevel: level}).Inc()
}

// IncErrorCounter increments the number of errors by task, operation and error code.
func (c *Client) IncErrorCounter(task, operation, code string) {
	c.collectorErrorCode.With(prometheus.Labels{labelTask: task, labelOperation: operation, labelCode: code}).Inc()
}

// Close method.
func (c *Client) Close() error {
	return nil
}
