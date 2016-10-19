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

/*
func BenchmarkLog(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithFields(log.Fields{
			"id": i,
		}).Info("benchmark log")
	}
}
*/
