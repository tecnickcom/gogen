// Package metrics defines the instrumentation metrics for this program.
package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tecnickcom/gogen/pkg/metrics"
	prom "github.com/tecnickcom/gogen/pkg/metrics/prometheus"
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
	InstrumentDB(dbName string, db *sql.DB) error
}

// Client groups the custom collectors to be shared with other packages.
type Client struct {
	libClient metrics.Client
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
	var err error

	opts := []prom.Option{
		prom.WithCollector(m.collectorExample),
	}

	m.libClient, err = prom.New(opts...)

	return m.libClient, err //nolint:wrapcheck
}

// IncExampleCounter is an example function to increment a counter.
func (m *Client) IncExampleCounter(code string) {
	m.collectorExample.With(prometheus.Labels{labelCode: code}).Inc()
}

// InstrumentDB wraps a sql.DB to collect metrics.
func (m *Client) InstrumentDB(dbName string, db *sql.DB) error {
	return m.libClient.InstrumentDB(dbName, db) //nolint:wrapcheck
}
