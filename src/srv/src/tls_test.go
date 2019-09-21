package main

import (
	"fmt"
	"testing"
)

func TestInitTLSDisabled(t *testing.T) {
	cfg := &TLSData{}
	err := cfg.initTLS()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected while initializing disabled TLS: %v", err))
	}
}

func TestInitTLSError(t *testing.T) {
	cfg := &TLSData{Enabled: true}
	err := cfg.initTLS()
	if err == nil {
		t.Error(fmt.Errorf("An error was expected while initializing TLS"))
	}
}
