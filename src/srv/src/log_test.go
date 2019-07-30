package main

import (
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestPrefixFieldClashes(t *testing.T) {
	log.WithFields(log.Fields{
		"msg": "additional message",
	}).Info("Testing log")
}

func TestLogError(t *testing.T) {
	log.WithFields(log.Fields{
		"error": fmt.Errorf("ERROR"),
	}).Error("Testing log error")
}

func TestLogJsonError(t *testing.T) {
	oldJSONMarshal := jsonMarshal
	defer func() { jsonMarshal = oldJSONMarshal }()
	jsonMarshal = mockJSONMarshalError

	log.Info("Testing log error")
}

func TestParseLogLevel(t *testing.T) {
	ld := &LogData{}

	ld.Level = "WRONG"
	_, _, err := ld.parseLogLevel()
	if err == nil {
		t.Error(fmt.Errorf("An error was expected"))
	}

	level := []string{"EMERGENCY", "ALERT", "CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG"}

	for i := range level {
		ld.Level = level[i]
		logLevel, syslogPriority, err := ld.parseLogLevel()
		if err != nil {
			t.Error(fmt.Errorf("An error was not expected: %v", err))
		}
		if logLevel > 7 {
			t.Error(fmt.Errorf("logLevel for %s should be < 8", ld.Level))
		}
		if syslogPriority < 0 {
			t.Error(fmt.Errorf("syslogPriority for %s should be >= 0", ld.Level))
		}
	}
}
