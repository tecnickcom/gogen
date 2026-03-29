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
	SqlOpen(driverName, dsn string) (*sql.DB, error)
}

// Client groups the custom collectors to be shared with other packages.
type Client struct {
	libClient metrics.Client
	// collectorExample is an example collector.
	collectorExample *prometheus.CounterVec
}

// New creates a metrics client wrapper with service-specific collectors.
//
// It solves a common observability need: keeping custom application metrics
// close to business events while still integrating with the shared gogen
// metrics client.
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

// CreateMetricsClientFunc constructs the underlying metrics client and
// registers all custom collectors.
//
// This method is passed to bootstrap so metrics initialization stays
// centralized and deterministic during startup.
func (m *Client) CreateMetricsClientFunc() (metrics.Client, error) {
	var err error

	opts := []prom.Option{
		prom.WithCollector(m.collectorExample),
	}

	m.libClient, err = prom.New(opts...)

	return m.libClient, err //nolint:wrapcheck
}

// IncExampleCounter increments the example counter for a given status code
// label.
//
// Labeled counters help developers slice behavior by outcome category, which
// is useful for dashboards and alerting.
func (m *Client) IncExampleCounter(code string) {
	m.collectorExample.With(prometheus.Labels{labelCode: code}).Inc()
}

// InstrumentDB instruments a SQL connection so query and connection pool
// telemetry is exported with the provided database name.
//
// Wrapping DB handles with metrics is key to spotting saturation and latency
// regressions before they become incidents.
func (m *Client) InstrumentDB(dbName string, db *sql.DB) error {
	return m.libClient.InstrumentDB(dbName, db) //nolint:wrapcheck
}

// SqlOpen opens a SQL database through the metrics client so returned handles
// are compatible with the service instrumentation pipeline.
func (m *Client) SqlOpen(driverName, dsn string) (*sql.DB, error) {
	return m.libClient.SqlOpen(driverName, dsn) //nolint:wrapcheck
}
