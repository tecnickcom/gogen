package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// MetricInFlightRequest is a default metric label.
	MetricInFlightRequest = "in_flight_requests"

	// MetricAPIRequests is a default metric label.
	MetricAPIRequests = "api_requests_total"

	// MetricRequestDuration is a default metric label.
	MetricRequestDuration = "request_duration_seconds"

	// MetricResponseSize is a default metric label.
	MetricResponseSize = "response_size_bytes"

	// MetricErrorLevel is a default metric label.
	MetricErrorLevel = "error_level_total"

	// MetricGoRuntime is a default metric label.
	MetricGoRuntime = "go_runtime"

	// MetricGoProcess is a default metric label.
	MetricGoProcess = "go_process"
)

var (
	// DefaultCollectors contains the list of default collectors
	DefaultCollectors = []Option{
		WithCollector(
			MetricGoRuntime,
			prometheus.NewGoCollector(),
		),
		WithCollector(
			MetricGoProcess,
			prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		),
		WithCollectorGauge(
			MetricInFlightRequest,
			prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: MetricInFlightRequest,
					Help: "A gauge of requests being served by the wrapped handler.",
				},
			),
		),
		WithCollectorCounterVec(
			MetricAPIRequests,
			prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: MetricAPIRequests,
					Help: "A counter for requests to the wrapped handler.",
				},
				[]string{"code", "method"},
			),
		),
		WithCollectorHistogramVec(
			MetricRequestDuration,
			prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    MetricRequestDuration,
					Help:    "A histogram of latencies for requests.",
					Buckets: prometheus.ExponentialBuckets(0.001, 10, 6),
				},
				[]string{"handler", "method"},
			),
		),
		WithCollectorHistogramVec(
			MetricResponseSize,
			prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    MetricResponseSize,
					Help:    "A histogram of response sizes for requests.",
					Buckets: prometheus.ExponentialBuckets(100, 2, 6),
				},
				[]string{},
			),
		),
		WithCollectorCounterVec(
			MetricErrorLevel,
			prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: MetricErrorLevel,
					Help: "Number of errors by levels.",
				},
				[]string{"level"},
			),
		),
	}
)

// Option is the interface that allows to set client options.
type Option func(c *Client) error

// WithHandlerOpts sets the options how to serve metrics via an http.Handler.
// The zero value of HandlerOpts is a reasonable default.
func WithHandlerOpts(opts promhttp.HandlerOpts) Option {
	return func(c *Client) error {
		c.handlerOpts = opts
		return nil
	}
}

// WithCollector register a new generic collector.
func WithCollector(name string, m prometheus.Collector) Option {
	return func(c *Client) error {
		c.Collector[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorGauge register a new Gauge collector.
func WithCollectorGauge(name string, m prometheus.Gauge) Option {
	return func(c *Client) error {
		c.CollectorGauge[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorCounter register a new Counter collector.
func WithCollectorCounter(name string, m prometheus.Counter) Option {
	return func(c *Client) error {
		c.CollectorCounter[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorSummary register a new Summary collector.
func WithCollectorSummary(name string, m prometheus.Summary) Option {
	return func(c *Client) error {
		c.CollectorSummary[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorHistogram register a new Histogram collector.
func WithCollectorHistogram(name string, m prometheus.Histogram) Option {
	return func(c *Client) error {
		c.CollectorHistogram[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorGaugeVec register a new GaugeVec collector.
func WithCollectorGaugeVec(name string, m *prometheus.GaugeVec) Option {
	return func(c *Client) error {
		c.CollectorGaugeVec[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorCounterVec register a new CounterVec collector.
func WithCollectorCounterVec(name string, m *prometheus.CounterVec) Option {
	return func(c *Client) error {
		c.CollectorCounterVec[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorSummaryVec register a new SummaryVec collector.
func WithCollectorSummaryVec(name string, m *prometheus.SummaryVec) Option {
	return func(c *Client) error {
		c.CollectorSummaryVec[name] = m
		return c.Registry.Register(m)
	}
}

// WithCollectorHistogramVec register a new HistogramVec collector.
func WithCollectorHistogramVec(name string, m *prometheus.HistogramVec) Option {
	return func(c *Client) error {
		c.CollectorHistogramVec[name] = m
		return c.Registry.Register(m)
	}
}
