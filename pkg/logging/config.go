package logging

import (
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// IncrementLogMetricsFunc is a type alias for the logging metric function
type IncrementLogMetricsFunc func(string)

func defaultConfig() *config {
	return &config{
		fields:            make([]zap.Field, 0),
		format:            JSONFormat,
		level:             zap.DebugLevel,
		outputPaths:       []string{"stderr"},
		errorOutputPaths:  []string{"stderr"},
		incMetricLogLevel: metrics.IncLogLevelCounter,
	}
}

type config struct {
	fields            []zap.Field
	format            Format
	level             zapcore.Level
	outputPaths       []string
	errorOutputPaths  []string
	incMetricLogLevel IncrementLogMetricsFunc
}
