package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	logrus_syslog "github.com/Sirupsen/logrus/hooks/syslog"
	syslog "log/syslog"
)

// LogData store a single log configuration
type LogData struct {
	Level   string `json:"level"`   // Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG.
	Network string `json:"network"` // Network type used by the Syslog (i.e. udp or tcp).
	Address string `json:"address"` // Network address of the Syslog daemon (ip:port) or just (:port).
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&logJSONFormatter{})

	// Output to stderr instead of stdout, could also be a file.
	log.SetOutput(os.Stderr)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

type logJSONFormatter struct {
}

// Format is a custom JSON formatter for the logs
func (f *logJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	data := make(log.Fields, len(entry.Data)+3)
	for k, v := range entry.Data {
		key := "ext_" + k
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/Sirupsen/logrus/issues/137
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
		stats.Increment(fmt.Sprintf("log.%s", data["level"]))
	}

	serialized, err := jsonMarshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}

// parseLogLevel takes a string level and returns the Logrus log level constant and he syslog priority
func (ld *LogData) parseLogLevel() (log.Level, syslog.Priority, error) {
	switch strings.ToLower(ld.Level) {
	case "emergency":
		return log.PanicLevel, syslog.LOG_EMERG, nil
	case "alert":
		return log.PanicLevel, syslog.LOG_ALERT, nil
	case "crit", "critical":
		return log.FatalLevel, syslog.LOG_CRIT, nil
	case "err", "error":
		return log.ErrorLevel, syslog.LOG_ERR, nil
	case "warn", "warning":
		return log.WarnLevel, syslog.LOG_WARNING, nil
	case "notice":
		return log.InfoLevel, syslog.LOG_NOTICE, nil
	case "info":
		return log.InfoLevel, syslog.LOG_INFO, nil
	case "debug":
		return log.DebugLevel, syslog.LOG_DEBUG, nil
	}

	return log.DebugLevel, syslog.LOG_DEBUG, fmt.Errorf("Not a valid log Level: %q", ld.Level)
}

// setLog configure the log
func (ld *LogData) setLog() error {
	logLevel, syslogPriority, err := ld.parseLogLevel()
	if err != nil {
		return fmt.Errorf("The logLevel must be one of the following: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG")
	}
	log.SetLevel(logLevel)
	hook, err := logrus_syslog.NewSyslogHook(ld.Network, ld.Address, syslogPriority, "")
	if err == nil {
		log.AddHook(hook)
	}
	return nil
}
