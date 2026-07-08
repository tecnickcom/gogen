package jwt

import "testing"

// Package-level sinks keep the results observable so the compiler cannot
// dead-code-eliminate the work being measured.

//nolint:gochecknoglobals // benchmark sinks that defeat dead-code elimination
var (
	sinkToken  string
	sinkClaims *Claims
	errSink    error
)

// BenchmarkIssueToken measures the full mint path: claims construction (with a
// fresh UUIDv7 jti), JSON encoding, HMAC signing, and base64url serialization.
func BenchmarkIssueToken(b *testing.B) {
	c, err := New(testKey, testVerify)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		sinkToken, errSink = c.IssueToken("bench-user")
	}

	if errSink != nil {
		b.Fatal(errSink)
	}
}

// BenchmarkVerifyToken measures the full verify path: compact-JWS splitting,
// base64url decoding, HMAC verification, JSON decoding, and claim validation.
func BenchmarkVerifyToken(b *testing.B) {
	c, err := New(testKey, testVerify)
	if err != nil {
		b.Fatal(err)
	}

	token, err := c.IssueToken("bench-user")
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		sinkClaims, errSink = c.VerifyToken(token)
	}

	if errSink != nil {
		b.Fatal(errSink)
	}
}
