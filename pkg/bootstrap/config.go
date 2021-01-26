package bootstrap

import (
	"context"
	"net/http"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	"go.uber.org/zap"
)

// Metrics is the interface for instrument metrics.
type Metrics interface {
	InstrumentHandler(string, http.HandlerFunc) http.Handler
	MetricsHandlerFunc() http.HandlerFunc
	IncLogLevelCounter(string)
}

// CreateLoggerFunc creates a new logger.
type CreateLoggerFunc func() (*zap.Logger, error)

// CreateMetricsClientFunc creates a new metrics client.
type CreateMetricsClientFunc func() (Metrics, error)

// BindFunc represents the function responsible to wire up all components of the application.
type BindFunc func(context.Context, *zap.Logger, Metrics) error

type config struct {
	context                 context.Context
	createLoggerFunc        CreateLoggerFunc
	createMetricsClientFunc CreateMetricsClientFunc
}

func defaultConfig() *config {
	return &config{
		context:                 context.Background(),
		createLoggerFunc:        defaultCreateLogger,
		createMetricsClientFunc: defaultCreateMetricsClientFunc,
	}
}

func defaultCreateLogger() (*zap.Logger, error) {
	return logging.NewLogger()
}

func defaultCreateMetricsClientFunc() (Metrics, error) {
	return metrics.New(metrics.DefaultCollectors...)
}
