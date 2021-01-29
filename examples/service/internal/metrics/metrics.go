// Package metrics defines the instrumentation metrics for this program.
package metrics

import (
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	prom "github.com/nexmoinc/gosrvlib/pkg/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// NameExample is the name of an example custom collector.
	NameExample = "example_collector"

	labelCode = "code"
)

// Metrics groups the custom collectors to be shared with other packages.
type Metrics struct {
	// collectorExample is an example collector.
	collectorExample *prometheus.CounterVec
}

// New creates a new Metrics instance.
func New() *Metrics {
	return &Metrics{
		collectorExample: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: NameExample,
				Help: "Example of custom collector.",
			},
			[]string{labelCode},
		),
	}
}

// CreateMetricsClientFunc returns the metrics Client.
func (m *Metrics) CreateMetricsClientFunc() (metrics.Client, error) {
	opt := prom.WithCollector(m.collectorExample)
	return prom.New(opt)
}

// IncExampleCounter is an example function to increment a counter.
func (m *Metrics) IncExampleCounter(code string) {
	m.collectorExample.With(prometheus.Labels{labelCode: code}).Inc()
}
