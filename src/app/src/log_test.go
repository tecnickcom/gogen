package main

import (
	"testing"

	log "github.com/Sirupsen/logrus"
)

func TestPrefixFieldClashes(t *testing.T) {
	log.WithFields(log.Fields{
		"msg": "additional message",
	}).Info("testing log")
}

func TestLogJsonError(t *testing.T) {
	oldJsonMarshal := jsonMarshal
	defer func() { jsonMarshal = oldJsonMarshal }()
	jsonMarshal = mockJsonMarshalError

	log.Info("testing log error")
}
