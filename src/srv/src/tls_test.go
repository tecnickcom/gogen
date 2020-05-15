package main

import (
	"testing"
)

func TestInitTLSDisabled(t *testing.T) {
	cfg := &TLSData{}
	err := cfg.initTLS()
	if err != nil {
		t.Errorf("An error was not expected while initializing disabled TLS: %v", err)
	}
}

func TestInitTLSError(t *testing.T) {
	cfg := &TLSData{Enabled: true}
	err := cfg.initTLS()
	if err == nil {
		t.Errorf("An error was expected while initializing TLS")
	}
}
