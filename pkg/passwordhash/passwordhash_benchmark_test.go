package passwordhash

import (
	"testing"
)

func BenchmarkPasswordHash(b *testing.B) {
	p := New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = p.PasswordHash("Benchmark-Password-Hash-Test")
	}
}

func BenchmarkPasswordVerify(b *testing.B) {
	p := New()

	hash := "eyJQIjp7IkEiOiJhcmdvbjJpZCIsIlYiOjE5LCJLIjozMiwiUyI6MTYsIlQiOjMsIk0iOjY1NTM2LCJQIjoxNn0sIlMiOiJ3UVltNGJma3RiSHEyb21Jd0Z1KzRRPT0iLCJLIjoiYVU4aE85MDBPZHE2YUt0V2lXejNSVzl5Z243MzRsaUphUHRNNnludmtZST0ifQo="

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = p.PasswordVerify("Test-Password-01234", hash)
	}
}

func Benchmark_EncryptPasswordHash(b *testing.B) {
	p := New()

	key := []byte("abcdefghijklmnopqrstuvwxyz012345")
	secret := "Benchmark-Password-Encrypt-Hash-Test"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = p.EncryptPasswordHash(key, secret)
	}
}
