package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// collector names

	// NameAPIRequests is the name of the collector that counts the total inbound http requests.
	NameAPIRequests = "api_requests_total"

	// NameInFlightRequests is the name of the collector that counts in-flight inbound http requests.
	NameInFlightRequests = "in_flight_requests"

	// NameRequestDuration is the name of the collector that measures the inbound http request duration in seconds.
	NameRequestDuration = "request_duration_seconds"

	// NameRequestSize is the name of the collector that measures the http request size in bytes.
	NameRequestSize = "requeste_size_bytes"

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

	// labels

	labelCode      = "code"
	labelHandler   = "handler"
	labelLevel     = "level"
	labelMethod    = "method"
	labelOperation = "operation"
	labelTask      = "task"
)

var (

	// DefaultSizeBuckets default prometheus buckets for size in bytes.
	DefaultSizeBuckets = prometheus.ExponentialBuckets(100, 10, 6)

	// DefaultDurationBuckets default prometheus buckets for duration in seconds.
	DefaultDurationBuckets = prometheus.ExponentialBuckets(0.001, 10, 6)

	// DefaultCollectors contains the list of default collectors
	DefaultCollectors = []Option{
		WithCollector(prometheus.NewGoCollector()),
		WithCollector(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{})),
		WithCollector(collectorInFlightRequests),
		WithCollector(collectorAPIRequests),
		WithCollector(collectorRequestDuration),
		WithCollector(collectorResponseSize),
		WithCollector(collectorRequestSize),
		WithCollector(collectorOutboundRequests),
		WithCollector(collectorOutboundRequestsDuration),
		WithCollector(collectorOutboundInFlightRequests),
		WithCollector(collectorErrorLevel),
		WithCollector(collectorErrorCode),
	}

	collectorInFlightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: NameInFlightRequests,
			Help: "Number of In-flight http requests.",
		},
	)

	collectorAPIRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameAPIRequests,
			Help: "Total number of http requests.",
		},
		[]string{labelCode, labelMethod},
	)

	collectorRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameRequestDuration,
			Help:    "Requests duration in seconds.",
			Buckets: DefaultDurationBuckets,
		},
		[]string{labelHandler, labelMethod},
	)

	collectorResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameResponseSize,
			Help:    "Response size in bytes.",
			Buckets: DefaultSizeBuckets,
		},
		[]string{},
	)

	collectorRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameRequestSize,
			Help:    "Requests size in bytes.",
			Buckets: DefaultSizeBuckets,
		},
		[]string{},
	)

	collectorOutboundRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameOutboundRequests,
			Help: "Total number of outbound http requests.",
		},
		[]string{labelCode, labelMethod},
	)

	collectorOutboundRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    NameOutboundRequestsDuration,
			Help:    "Outbound requests duration in seconds.",
			Buckets: DefaultDurationBuckets,
		},
		[]string{labelCode, labelMethod},
	)

	collectorOutboundInFlightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: NameOutboundInFlightRequests,
			Help: "Number of outbound In-flight http requests.",
		},
	)

	collectorErrorLevel = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameErrorLevel,
			Help: "Number of errors by severity level.",
		},
		[]string{labelLevel},
	)

	collectorErrorCode = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: NameErrorCode,
			Help: "Number of errors by task, operation and error code.",
		},
		[]string{labelTask, labelOperation, labelCode},
	)
)
