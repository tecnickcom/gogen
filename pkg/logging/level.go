package logging

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ParseLevel converts syslog standard levels to zap a log level
func ParseLevel(l string) (zapcore.Level, error) {
	switch strings.ToLower(l) {
	case "emergency":
		return zap.PanicLevel, nil
	case "alert":
		return zap.PanicLevel, nil
	case "crit", "critical":
		return zap.FatalLevel, nil
	case "err", "error":
		return zap.ErrorLevel, nil
	case "warn", "warning":
		return zap.WarnLevel, nil
	case "notice":
		return zap.InfoLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "debug":
		return zap.DebugLevel, nil
	}
	return zap.DebugLevel, fmt.Errorf("invalid log level %q", l)
}
