// Package metrics defines the instrumentation metrics for this program.
package metrics

import (
	"github.com/Vonage/gosrvlib/pkg/metrics"
	prom "github.com/Vonage/gosrvlib/pkg/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// NameExample is the name of an example custom collector.
	NameExample = "example_collector"

	labelCode = "code"
)

// Metrics is the interface for the custom metrics.
type Metrics interface {
	CreateMetricsClientFunc() (metrics.Client, error)
	IncExampleCounter(code string)
}

// Client groups the custom collectors to be shared with other packages.
type Client struct {
	// collectorExample is an example collector.
	collectorExample *prometheus.CounterVec
}

// New creates a new Client instance.
func New() *Client {
	return &Client{
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
func (m *Client) CreateMetricsClientFunc() (metrics.Client, error) {
	opt := prom.WithCollector(m.collectorExample)
	return prom.New(opt) //nolint:wrapcheck
}

// IncExampleCounter is an example function to increment a counter.
func (m *Client) IncExampleCounter(code string) {
	m.collectorExample.With(prometheus.Labels{labelCode: code}).Inc()
}
