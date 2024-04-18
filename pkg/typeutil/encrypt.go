package typeutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

func newAESGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return cipher.NewGCM(block) //nolint:wrapcheck
}

// Encrypt encrypts the input data with the specified key.
// The key argument must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func Encrypt(key, data []byte) ([]byte, error) {
	aesgcm, err := newAESGCM(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	_, _ = rand.Read(nonce)

	return aesgcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypt decrypts data encrypted with Encrypt().
// The key argument must be the same used to encrypt the data:
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func Decrypt(key, data []byte) ([]byte, error) {
	aesgcm, err := newAESGCM(key)
	if err != nil {
		return nil, err
	}

	ns := aesgcm.NonceSize()
	if len(data) < ns {
		return nil, errors.New("invalid data size")
	}

	return aesgcm.Open(nil, data[:ns], data[ns:], nil) //nolint:wrapcheck
}
