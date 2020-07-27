package bootstrap

import (
	"context"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

func defaultConfig() *config {
	return &config{
		context:          context.Background(),
		createLoggerFunc: defaultCreateLogger,
	}
}

func defaultCreateLogger() (*zap.Logger, error) {
	return logging.NewLogger()
}

type config struct {
	context          context.Context
	createLoggerFunc CreateLoggerFunc
}
