package bootstrap

import (
	"context"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func defaultConfig() *config {
	return &config{
		context:                  context.Background(),
		createLoggerFunc:         defaultCreateLogger,
		createMetricRegisterFunc: defaultCreateMetricRegister,
	}
}

func defaultCreateLogger() (*zap.Logger, error) {
	return logging.NewLogger()
}

func defaultCreateMetricRegister() prometheus.Registerer {
	return prometheus.DefaultRegisterer
}

type config struct {
	context                  context.Context
	createLoggerFunc         CreateLoggerFunc
	createMetricRegisterFunc CreateMetricRegisterFunc
}
