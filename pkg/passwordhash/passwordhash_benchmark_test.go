package passwordhash

import (
	"testing"
)

func BenchmarkPasswordHash(b *testing.B) {
	p := New()

	for b.Loop() {
		_, _ = p.PasswordHash("Benchmark-Password-Hash-Test")
	}
}

func BenchmarkPasswordVerify(b *testing.B) {
	p := New()

	// Mint the hash with the current defaults so the benchmark always measures
	// the configuration the package actually ships, rather than a frozen blob
	// whose embedded parameters go stale when defaults change.
	hash, err := p.PasswordHash("Test-Password-01234")
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, _ = p.PasswordVerify("Test-Password-01234", hash)
	}
}

func Benchmark_EncryptPasswordHash(b *testing.B) {
	p := New()

	key := []byte("abcdefghijklmnopqrstuvwxyz012345")
	secret := "Benchmark-Password-Encrypt-Hash-Test"

	for b.Loop() {
		_, _ = p.EncryptPasswordHash(key, secret)
	}
}
