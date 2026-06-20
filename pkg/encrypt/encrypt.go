/*
Package encrypt provides helpers for encrypting and decrypting data safely for
transport and storage.

It solves the problem of protecting application payloads when data moves between
systems such as databases, queues, caches, or external services.

This package uses AES-GCM authenticated encryption with a random nonce prefixed
to the ciphertext. It provides both raw byte-level APIs and convenience helpers
that serialize arbitrary values with gob or JSON before encryption.

Top features:

  - AES-GCM encryption and decryption with standard key sizes (16, 24, or 32 bytes
    for AES-128, AES-192, and AES-256)
  - nonce generation via secure random bytes and nonce-prefix output format for
    self-contained ciphertext payloads
  - base64 encoded byte and string helpers for transport-safe interchange
  - gob and JSON wrappers for encrypting and decrypting structured Go values
  - layered error propagation from encoding, base64, and cryptographic operations

Benefits:

  - reduce boilerplate for secure payload handling
  - avoid accidental use of insecure or unauthenticated encryption modes
  - simplify encryption of structured data in distributed systems
*/
package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/tecnickcom/gogen/pkg/random"
)

// config holds the resolved options for an encryption call.
//
// The zero value uses a nil random reader, which random.New interprets as
// crypto/rand.Reader. This keeps Encrypt non-failing in production while
// allowing tests to inject a deterministic or failing reader via Option,
// without any runtime-mutated package global.
type config struct {
	randReader io.Reader
}

// Option customizes the behavior of EncryptWith.
//
// Options are additive: existing Encrypt/Decrypt signatures are unchanged and
// always use the secure default reader (crypto/rand.Reader).
type Option func(*config)

// WithRandReader overrides the random source used to generate the AES-GCM nonce.
//
// It is primarily useful for tests that need a deterministic or failing reader.
// Production code should rely on the default (crypto/rand.Reader) via Encrypt.
func WithRandReader(r io.Reader) Option {
	return func(c *config) {
		c.randReader = r
	}
}

// newAESGCM creates an AES-GCM AEAD from key.
//
// key must be a valid AES key length (16, 24, or 32 bytes).
func newAESGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return cipher.NewGCM(block) //nolint:wrapcheck
}

// Encrypt seals msg with AES-GCM and prepends the random nonce.
//
// key must be 16, 24, or 32 bytes for AES-128/192/256.
// The output is self-contained: nonce || ciphertext.
// The nonce is generated from crypto/rand.Reader.
func Encrypt(key, msg []byte) ([]byte, error) {
	return EncryptWith(key, msg)
}

// EncryptWith behaves like Encrypt but accepts options that customize the
// random source used for nonce generation (see WithRandReader).
//
// Without options it is identical to Encrypt and uses crypto/rand.Reader.
func EncryptWith(key, msg []byte, opts ...Option) ([]byte, error) {
	cfg := &config{}
	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	aesgcm, err := newAESGCM(key)
	if err != nil {
		return nil, err
	}

	nonce, err := random.New(cfg.randReader).RandomBytes(aesgcm.NonceSize())
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return aesgcm.Seal(nonce, nonce, msg, nil), nil
}

// Decrypt opens a nonce-prefixed AES-GCM payload produced by Encrypt.
//
// key must match the key used during encryption.
func Decrypt(key, msg []byte) ([]byte, error) {
	aesgcm, err := newAESGCM(key)
	if err != nil {
		return nil, err
	}

	ns := aesgcm.NonceSize()
	if len(msg) < ns {
		return nil, errors.New("invalid input size")
	}

	return aesgcm.Open(nil, msg[:ns], msg[ns:], nil) //nolint:wrapcheck
}

// byteEncryptEncoded encrypts data and returns Base64-encoded ciphertext bytes.
func byteEncryptEncoded(key []byte, data []byte) ([]byte, error) {
	msg, err := Encrypt(key, data)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	dst := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(dst, msg)

	return dst, nil
}

// byteDecryptEncoded decodes Base64 payload bytes and decrypts them.
func byteDecryptEncoded(key, msg []byte) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(msg)))

	n, err := base64.StdEncoding.Decode(dst, msg)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	return Decrypt(key, dst[:n])
}

// ByteEncryptAny gob-encodes data, encrypts it, and returns Base64 bytes.
//
// This helper is useful for encrypting arbitrary Go values for transport.
func ByteEncryptAny(key []byte, data any) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := gob.NewEncoder(buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("encode gob: %w", err)
	}

	return byteEncryptEncoded(key, buf.Bytes())
}

// ByteDecryptAny decrypts Base64 bytes produced by ByteEncryptAny into data.
//
// data must be a pointer to the destination type.
func ByteDecryptAny(key, msg []byte, data any) error {
	dec, err := byteDecryptEncoded(key, msg)
	if err != nil {
		return err
	}

	err = gob.NewDecoder(bytes.NewBuffer(dec)).Decode(data)
	if err != nil {
		return fmt.Errorf("decode gob: %w", err)
	}

	return nil
}

// EncryptAny wraps ByteEncryptAny and returns a string payload.
func EncryptAny(key []byte, data any) (string, error) {
	b, err := ByteEncryptAny(key, data)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	return string(b), nil
}

// DecryptAny wraps ByteDecryptAny for string input payloads.
func DecryptAny(key []byte, msg string, data any) error {
	return ByteDecryptAny(key, []byte(msg), data)
}

// ByteEncryptSerializeAny JSON-encodes data, encrypts it, and returns Base64 bytes.
//
// Use this helper for cross-language payloads where JSON interoperability is required.
func ByteEncryptSerializeAny(key []byte, data any) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := json.NewEncoder(buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}

	return byteEncryptEncoded(key, buf.Bytes())
}

// ByteDecryptSerializeAny decrypts Base64 bytes into JSON-decoded data.
//
// data must be a pointer to the destination type.
func ByteDecryptSerializeAny(key, msg []byte, data any) error {
	dec, err := byteDecryptEncoded(key, msg)
	if err != nil {
		return err
	}

	err = json.NewDecoder(bytes.NewBuffer(dec)).Decode(data)
	if err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	return nil
}

// EncryptSerializeAny wraps ByteEncryptSerializeAny and returns a string payload.
func EncryptSerializeAny(key []byte, data any) (string, error) {
	b, err := ByteEncryptSerializeAny(key, data)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	return string(b), nil
}

// DecryptSerializeAny wraps ByteDecryptSerializeAny for string input payloads.
func DecryptSerializeAny(key []byte, msg string, data any) error {
	return ByteDecryptSerializeAny(key, []byte(msg), data)
}
