package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/tecnickcom/statsd"
)

var stats *statsd.Client

// initStats initialize the StatsD client
func initStats(cfg *params) (err error) {
	stats, err = statsd.New(
		statsd.Prefix(cfg.statsPrefix),
		statsd.Network(cfg.statsNetwork),
		statsd.Address(cfg.statsAddress),
		statsd.FlushPeriod(time.Duration(cfg.statsFlushPeriod)*time.Millisecond),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"statsAddress": cfg.statsAddress,
			"statsNetwork": cfg.statsNetwork,
		}).Error("Unable to connect to the StatD demon")
	}
	return err
}
