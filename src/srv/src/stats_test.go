package main

import (
	"fmt"
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
		t.Error(fmt.Errorf("An error was expected while initializing Stats"))
	}
}
