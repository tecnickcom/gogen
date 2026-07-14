package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInsecureNoRedaction pins the deliberate non-redaction contract: the input
// comes back verbatim, secrets included. The paired Default assertions show what
// is given up by choosing it.
func TestInsecureNoRedaction(t *testing.T) {
	t.Parallel()

	cases := []string{
		"",
		"nothing sensitive here",
		"Authorization: Bearer SUPERSECRET",
		"password=SUPERSECRET&reference=VISIBLE",
		`{"api_key":"SUPERSECRET"}`,
		"postgres://app:SUPERSECRET@10.0.0.5/db",
	}

	for _, in := range cases {
		require.Equal(t, in, InsecureNoRedaction([]byte(in)), "input must be returned verbatim: %s", in)
	}

	// The last four carry secrets that the default redactor removes; the bypass
	// leaves every one of them in the clear.
	for _, in := range cases[2:] {
		require.Contains(t, InsecureNoRedaction([]byte(in)), "SUPERSECRET")
		require.NotContains(t, Default().String(in), "SUPERSECRET")
	}
}
