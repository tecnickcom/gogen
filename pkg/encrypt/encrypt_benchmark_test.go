package encrypt_test

import (
	"testing"

	"github.com/tecnickcom/gogen/pkg/encrypt"
)

type benchData struct {
	Alpha string
	Beta  int
	Gamma float32
	Delta []string
}

// benchValue is a representative payload exercising strings, ints, floats, and slices.
func benchValue() benchData {
	return benchData{
		Alpha: "the quick brown fox jumps over the lazy dog",
		Beta:  -123456,
		Gamma: 3.14159,
		Delta: []string{"one", "two", "three", "four", "five"},
	}
}

// benchKey is a 32-byte AES-256 key; benchMsg is a representative plaintext.
const (
	benchKey = "abcdefghijklmnopqrstuvwxyz012345"
	benchMsg = "the quick brown fox jumps over the lazy dog"
)

func BenchmarkEncrypt(b *testing.B) {
	key, msg := []byte(benchKey), []byte(benchMsg)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = encrypt.Encrypt(key, msg)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key := []byte(benchKey)

	enc, err := encrypt.Encrypt(key, []byte(benchMsg))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = encrypt.Decrypt(key, enc)
	}
}

func BenchmarkEncryptSerializeAny(b *testing.B) {
	key, val := []byte(benchKey), benchValue()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = encrypt.EncryptSerializeAny(key, val)
	}
}

func BenchmarkDecryptSerializeAny(b *testing.B) {
	key := []byte(benchKey)

	enc, err := encrypt.EncryptSerializeAny(key, benchValue())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		var out benchData

		_ = encrypt.DecryptSerializeAny(key, enc, &out)
	}
}
