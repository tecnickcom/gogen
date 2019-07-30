package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	logrus "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	syslog "log/syslog"
)

var stdLogger *log.Logger

// LogData store a single log configuration
type LogData struct {
	Level   string `json:"level"`   // Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG.
	Network string `json:"network"` // Network type used by the Syslog (i.e. udp or tcp).
	Address string `json:"address"` // Network address of the Syslog daemon (ip:port) or just (:port).
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logJSONFormatter{})

	// Output to stderr instead of stdout, could also be a file.
	logrus.SetOutput(os.Stderr)

	// Only log the warning severity or above.
	logrus.SetLevel(logrus.DebugLevel)
}

type logJSONFormatter struct {
}

// Format is a custom JSON formatter for the logs
func (f *logJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+3)
	for k, v := range entry.Data {
		key := "ext_" + k
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[key] = v.Error()
		default:
			data[key] = v
		}
	}

	nowTime := time.Now().UTC()

	hostname, err := os.Hostname()
	if err == nil {
		hostname = ""
	}
	data["hostname"] = hostname
	data["program"] = ProgramName
	data["version"] = ProgramVersion
	data["release"] = ProgramRelease
	data["datetime"] = nowTime.Format(time.RFC3339)
	data["timestamp"] = nowTime.UnixNano()
	data["level"] = entry.Level.String()
	data["msg"] = entry.Message

	if stats != nil {
		// count each error type
		stats.Increment(fmt.Sprintf("logrus.%s", data["level"]))
	}

	serialized, err := jsonMarshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}

// parseLogLevel takes a string level and returns the Logrus log level constant and he syslog priority
func (ld *LogData) parseLogLevel() (logrus.Level, syslog.Priority, error) {
	switch strings.ToLower(ld.Level) {
	case "emergency":
		return logrus.PanicLevel, syslog.LOG_EMERG, nil
	case "alert":
		return logrus.PanicLevel, syslog.LOG_ALERT, nil
	case "crit", "critical":
		return logrus.FatalLevel, syslog.LOG_CRIT, nil
	case "err", "error":
		return logrus.ErrorLevel, syslog.LOG_ERR, nil
	case "warn", "warning":
		return logrus.WarnLevel, syslog.LOG_WARNING, nil
	case "notice":
		return logrus.InfoLevel, syslog.LOG_NOTICE, nil
	case "info":
		return logrus.InfoLevel, syslog.LOG_INFO, nil
	case "debug":
		return logrus.DebugLevel, syslog.LOG_DEBUG, nil
	}

	return logrus.DebugLevel, syslog.LOG_DEBUG, fmt.Errorf("not a valid log Level: %q", ld.Level)
}

// setLog configure the log
func (ld *LogData) setLog() error {
	logLevel, syslogPriority, err := ld.parseLogLevel()
	if err != nil {
		return fmt.Errorf("the logLevel must be one of the following: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG")
	}
	logrus.SetLevel(logLevel)
	hook, err := logrus_syslog.NewSyslogHook(ld.Network, ld.Address, syslogPriority, "")
	if err == nil {
		logrus.AddHook(hook)
	}

	logger := logrus.StandardLogger()
	stdLogger = log.New(logger.Out, "", 0)

	return nil
}
