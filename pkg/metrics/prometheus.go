package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricInFlightGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "in_flight_requests",
			Help: "A gauge of requests currently being served by the wrapped handler.",
		})

	metricCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "A counter for requests to the wrapped handler.",
		},
		[]string{"code", "method"},
	)

	// metricDuration is partitioned by the HTTP method and handler.
	// It uses custom buckets based on the expected request duration.
	metricDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"handler", "method"},
	)

	// metricResponseSize has no labels, making it a zero-dimensional ObserverVec.
	metricResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{},
	)

	// metricErrorLevel counts errors by level
	metricError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_level_total",
			Help: "Number of error levels.",
		},
		[]string{"level"},
	)
)

// register all of the metrics in the standard registry.
func init() {
	prometheus.MustRegister(
		metricInFlightGauge,
		metricCounter,
		metricDuration,
		metricResponseSize,
		metricError,
	)
}

// Handler wraps an http.Handler to collect Prometheus metrics
func Handler(path string, handler http.HandlerFunc) http.Handler {
	return promhttp.InstrumentHandlerInFlight(
		metricInFlightGauge,
		promhttp.InstrumentHandlerDuration(
			metricDuration.MustCurryWith(prometheus.Labels{"handler": path}),
			promhttp.InstrumentHandlerCounter(
				metricCounter,
				promhttp.InstrumentHandlerResponseSize(metricResponseSize, handler),
			),
		),
	)
}

// IncLogLevelCounter counts the number of errors for each log level
func IncLogLevelCounter(level string) {
	metricError.With(prometheus.Labels{"level": level}).Inc()
}
