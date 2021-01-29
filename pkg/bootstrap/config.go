package bootstrap

import (
	"context"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	"go.uber.org/zap"
)

// CreateLoggerFunc creates a new logger.
type CreateLoggerFunc func() (*zap.Logger, error)

// CreateMetricsClientFunc creates a new metrics client.
type CreateMetricsClientFunc func() (metrics.Client, error)

// BindFunc represents the function responsible to wire up all components of the application.
type BindFunc func(context.Context, *zap.Logger, metrics.Client) error

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

func defaultCreateMetricsClientFunc() (metrics.Client, error) {
	return &metrics.Default{}, nil
}
