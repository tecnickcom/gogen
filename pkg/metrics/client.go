package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Client represents the state type of this client.
type Client struct {
	Registry              *prometheus.Registry
	handlerOpts           promhttp.HandlerOpts
	Collector             map[string]prometheus.Collector
	CollectorGauge        map[string]prometheus.Gauge
	CollectorCounter      map[string]prometheus.Counter
	CollectorSummary      map[string]prometheus.Summary
	CollectorHistogram    map[string]prometheus.Histogram
	CollectorGaugeVec     map[string]*prometheus.GaugeVec
	CollectorCounterVec   map[string]*prometheus.CounterVec
	CollectorSummaryVec   map[string]*prometheus.SummaryVec
	CollectorHistogramVec map[string]*prometheus.HistogramVec
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
		Registry:              prometheus.NewRegistry(),
		handlerOpts:           promhttp.HandlerOpts{},
		Collector:             make(map[string]prometheus.Collector),
		CollectorGauge:        make(map[string]prometheus.Gauge),
		CollectorCounter:      make(map[string]prometheus.Counter),
		CollectorSummary:      make(map[string]prometheus.Summary),
		CollectorHistogram:    make(map[string]prometheus.Histogram),
		CollectorGaugeVec:     make(map[string]*prometheus.GaugeVec),
		CollectorCounterVec:   make(map[string]*prometheus.CounterVec),
		CollectorSummaryVec:   make(map[string]*prometheus.SummaryVec),
		CollectorHistogramVec: make(map[string]*prometheus.HistogramVec),
	}
}

// InstrumentHandler wraps an http.Handler to collect Prometheus metrics.
// Requires DefaultCollectors.
func (c *Client) InstrumentHandler(path string, handler http.HandlerFunc) http.Handler {
	var h http.Handler
	h = handler
	metricResponseSize, ok := c.CollectorHistogramVec[MetricResponseSize]
	if ok {
		h = promhttp.InstrumentHandlerResponseSize(metricResponseSize, h)
	}
	metricAPIRequests, ok := c.CollectorCounterVec[MetricAPIRequests]
	if ok {
		h = promhttp.InstrumentHandlerCounter(metricAPIRequests, h)
	}
	metricRequestDuration, ok := c.CollectorHistogramVec[MetricRequestDuration]
	if ok {
		h = promhttp.InstrumentHandlerDuration(metricRequestDuration.MustCurryWith(prometheus.Labels{"handler": path}), h)
	}
	metricInFlightRequest, ok := c.CollectorGauge[MetricInFlightRequest]
	if ok {
		h = promhttp.InstrumentHandlerInFlight(metricInFlightRequest, h)
	}
	return h
}

// MetricsHandlerFunc returns an http handler function to serve the metrics endpoint.
func (c *Client) MetricsHandlerFunc() http.HandlerFunc {
	h := promhttp.HandlerFor(c.Registry, c.handlerOpts)
	return promhttp.InstrumentMetricHandler(c.Registry, h).ServeHTTP
}

// DefaultMetricsHandlerFunc returns a default http handler function to serve the metrics endpoint.
func DefaultMetricsHandlerFunc() http.HandlerFunc {
	return promhttp.Handler().ServeHTTP
}

// IncLogLevelCounter counts the number of errors for each syslog level.
// Requires DefaultCollectors.
func (c *Client) IncLogLevelCounter(level string) {
	m, ok := c.CollectorCounterVec[MetricErrorLevel]
	if ok {
		m.With(prometheus.Labels{"level": level}).Inc()
	}
}
