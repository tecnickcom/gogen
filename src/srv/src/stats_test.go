package main

import (
	"testing"
)

func TestInitStatsError(t *testing.T) {
	cfg := &StatsData{
		Prefix:      "",
		Network:     "",
		Address:     "",
		FlushPeriod: 0,
	}

	err := initStats(cfg)
	if err == nil {
		t.Errorf("An error was expected while initializing Stats")
	}
}
