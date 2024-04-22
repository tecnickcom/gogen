package typeutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
)

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

	nonce := make([]byte, aesgcm.NonceSize())
	_, _ = rand.Read(nonce)

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

// ByteEncryptAny encrypts data with the specified key and returns a base64 byte slice.
// The key argument must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func ByteEncryptAny(key []byte, data any) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(data); err != nil {
		return nil, fmt.Errorf("encode gob: %w", err)
	}

	msg, err := Encrypt(key, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	dst := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(dst, msg)

	return dst, nil
}

// ByteDecryptAny decrypts a byte-slice message produced with the ByteEncryptAny function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
// The key argument must be the same used to encrypt the data:
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func ByteDecryptAny(key, msg []byte, data any) error {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(msg)))

	n, err := base64.StdEncoding.Decode(dst, msg)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	dec, err := Decrypt(key, dst[:n])
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}

	if err := gob.NewDecoder(bytes.NewBuffer(dec)).Decode(data); err != nil {
		return fmt.Errorf("decode gob: %w", err)
	}

	return nil
}

// EncryptAny encrypts data with the specified key and returns a base64 string
// The key argument must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func EncryptAny(key []byte, data any) (string, error) {
	b, err := ByteEncryptAny(key, data)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(b), nil
}

// DecryptAny decrypts string message produced with the EncryptAny function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
// The key argument must be the same used to encrypt the data:
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func DecryptAny(key []byte, msg string, data any) error {
	return ByteDecryptAny(key, []byte(msg), data)
}
