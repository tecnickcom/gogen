/*
Package encrypt contains a collection of utility functions to encrypt and decrypt data.
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

	"github.com/Vonage/gosrvlib/pkg/random"
)

// randReader is the default random number generator.
var randReader io.Reader //nolint:gochecknoglobals

func newAESGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return cipher.NewGCM(block) //nolint:wrapcheck
}

// Encrypt encrypts the byte-slice input msg with the specified key.
// The key argument must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func Encrypt(key, msg []byte) ([]byte, error) {
	aesgcm, err := newAESGCM(key)
	if err != nil {
		return nil, err
	}

	nonce, err := random.New(randReader).RandomBytes(aesgcm.NonceSize())
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return aesgcm.Seal(nonce, nonce, msg, nil), nil
}

// Decrypt decrypts a byte-slice data encrypted with the Encrypt function.
// The key argument must be the same used to encrypt the data:
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
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

func byteEncryptEncoded(key []byte, data []byte) ([]byte, error) {
	msg, err := Encrypt(key, data)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	dst := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(dst, msg)

	return dst, nil
}

func byteDecryptEncoded(key, msg []byte) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(msg)))

	n, err := base64.StdEncoding.Decode(dst, msg)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	return Decrypt(key, dst[:n])
}

// ByteEncryptAny encrypts the input data with the specified key and returns a base64 byte slice.
// The input data is serialized using gob, encrypted with the Encrypt method and encoded as base64.
// The key argument must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func ByteEncryptAny(key []byte, data any) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(data); err != nil {
		return nil, fmt.Errorf("encode gob: %w", err)
	}

	return byteEncryptEncoded(key, buf.Bytes())
}

// ByteDecryptAny decrypts a byte-slice message produced with the ByteEncryptAny function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
// The key argument must be the same used to encrypt the data:
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func ByteDecryptAny(key, msg []byte, data any) error {
	dec, err := byteDecryptEncoded(key, msg)
	if err != nil {
		return err
	}

	if err := gob.NewDecoder(bytes.NewBuffer(dec)).Decode(data); err != nil {
		return fmt.Errorf("decode gob: %w", err)
	}

	return nil
}

// EncryptAny wraps the ByteEncryptAny function to return a string instead of a byte slice.
func EncryptAny(key []byte, data any) (string, error) { //nolint:revive
	b, err := ByteEncryptAny(key, data)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(b), nil
}

// DecryptAny wraps the ByteDecryptAny function to accept a msg string instead of a byte slice.
func DecryptAny(key []byte, msg string, data any) error {
	return ByteDecryptAny(key, []byte(msg), data)
}

// ByteEncryptSerializeAny encrypts the input data with the specified key and returns a base64 byte slice.
// The input data is serialized using json, encrypted with the Encrypt method and encoded as base64.
// The key argument must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func ByteEncryptSerializeAny(key []byte, data any) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		return nil, fmt.Errorf("encode gob: %w", err)
	}

	return byteEncryptEncoded(key, buf.Bytes())
}

// ByteDecryptSerializeAny decrypts a byte-slice message produced with the ByteEncryptSerializeAny function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
// The key argument must be the same used to encrypt the data:
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func ByteDecryptSerializeAny(key, msg []byte, data any) error {
	dec, err := byteDecryptEncoded(key, msg)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(bytes.NewBuffer(dec)).Decode(data); err != nil {
		return fmt.Errorf("decode gob: %w", err)
	}

	return nil
}

// EncryptSerializeAny wraps the ByteEncrypSerializetAny function to return a string instead of a byte slice.
func EncryptSerializeAny(key []byte, data any) (string, error) { //nolint:revive
	b, err := ByteEncryptSerializeAny(key, data)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(b), nil
}

// DecryptSerializeAny wraps the ByteDecryptSerializeAny function to accept a msg string instead of a byte slice.
func DecryptSerializeAny(key []byte, msg string, data any) error {
	return ByteDecryptSerializeAny(key, []byte(msg), data)
}
