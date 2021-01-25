package bootstrap

import (
	"context"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	"go.uber.org/zap"
)

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

func defaultCreateMetricsClientFunc() (*metrics.Client, error) {
	return metrics.New(metrics.DefaultCollectors...)
}

type config struct {
	context                 context.Context
	createLoggerFunc        CreateLoggerFunc
	createMetricsClientFunc CreateMetricsClientFunc
}
