package main

import (
	"crypto/tls"
)

// TLSData store a single TLS configuration
type TLSData struct {
	Enabled   bool   `json:"enabled"` // Enable the server in HTTS-only mode
	CertPem   []byte `json:"certPem"` // TLS certificate in PEM format
	KeyPem    []byte `json:"keyPem"`  // TLS private key in PEM format
	tlsConfig *tls.Config
}

// initTLS initialize TLS certificates
func (cfg *TLSData) initTLS() error {
	if !cfg.Enabled {
		return nil
	}
	cert, err := tls.X509KeyPair(cfg.CertPem, cfg.KeyPem)
	if err != nil {
		return err
	}
	cfg.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	return nil
}
