package logsrv

import (
	"github.com/rs/zerolog"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

// SlogLevel converts a zerolog level into an slog one.
//
//nolint:cyclop
func SlogLevel(l zerolog.Level) logutil.LogLevel {
	switch l {
	case zerolog.PanicLevel:
		return logutil.LevelEmergency
	case zerolog.FatalLevel:
		return logutil.LevelAlert
	case zerolog.ErrorLevel:
		return logutil.LevelError
	case zerolog.WarnLevel:
		return logutil.LevelWarning
	case zerolog.InfoLevel:
		return logutil.LevelInfo
	case zerolog.DebugLevel:
		return logutil.LevelDebug
	case zerolog.TraceLevel:
		return logutil.LevelTrace
	case zerolog.NoLevel:
		return logutil.LevelInfo
	case zerolog.Disabled:
		return logutil.LevelEmergency + 1
	default:
		return logutil.LevelInfo
	}
}
