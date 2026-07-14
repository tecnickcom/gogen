package redact

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestPEMPGPPrivateKey verifies armored OpenPGP secret-key blocks
// ("BEGIN PGP PRIVATE KEY BLOCK") are redacted, both standalone and inline.
func TestPEMPGPPrivateKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{
			"-----BEGIN PGP PRIVATE KEY BLOCK-----\n\nlQOYBGKxSECRET\n=abcd\n-----END PGP PRIVATE KEY BLOCK-----\n",
			"-----BEGIN PGP PRIVATE KEY BLOCK-----\n***\n-----END PGP PRIVATE KEY BLOCK-----\n",
		},
		{
			`{"blob":"-----BEGIN PGP PRIVATE KEY BLOCK-----\nlQOYSECRET\n-----END PGP PRIVATE KEY BLOCK-----"}`,
			`{"blob":"-----BEGIN PGP PRIVATE KEY BLOCK-----***-----END PGP PRIVATE KEY BLOCK-----"}`,
		},
		// A PGP PUBLIC key block stays visible.
		{
			"-----BEGIN PGP PUBLIC KEY BLOCK-----\nmQENBGKx\n-----END PGP PUBLIC KEY BLOCK-----",
			"-----BEGIN PGP PUBLIC KEY BLOCK-----\nmQENBGKx\n-----END PGP PUBLIC KEY BLOCK-----",
		},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
		once := Default().String(tc.input)
		require.Equal(t, once, Default().String(once), "not idempotent: %s", tc.input)
	}
}

// TestInlinePEMScanIsLinear guards against the quadratic blow-up on a payload
// packed with unterminated BEGIN markers: redaction must stay well under a
// second even for a large adversarial input.
func TestInlinePEMScanIsLinear(t *testing.T) {
	t.Parallel()

	for _, input := range [][]byte{
		[]byte(strings.Repeat("-----BEGIN A PRIVATE KEY-----x\n", 40000)), // ~1.2 MB, no END markers
		[]byte(strings.Repeat("-----BEGIN A PRIVATE KEY-----x ", 40000)),  // ~1.2 MB, no newlines
	} {
		done := make(chan struct{})

		go func() {
			_ = Default().Bytes(input)

			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("inline PEM scan did not finish in 2s for a %d-byte input (quadratic?)", len(input))
		}
	}
}

//nolint:gosec // The PEM fixtures are fake fragments, not real credentials.
func TestRedactPEMPrivateKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "rsa private key body replaced",
			input: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA7Zx9\nqqLNM5pFz\n-----END RSA PRIVATE KEY-----",
			want:  "-----BEGIN RSA PRIVATE KEY-----\n***\n-----END RSA PRIVATE KEY-----",
		},
		{
			name:  "openssh private key with surrounding text",
			input: "before\n-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaA==\n-----END OPENSSH PRIVATE KEY-----\nafter",
			want:  "before\n-----BEGIN OPENSSH PRIVATE KEY-----\n***\n-----END OPENSSH PRIVATE KEY-----\nafter",
		},
		{
			name:  "crlf body keeps crlf terminators",
			input: "-----BEGIN EC PRIVATE KEY-----\r\nMHcCAQEE\r\n-----END EC PRIVATE KEY-----\r\n",
			want:  "-----BEGIN EC PRIVATE KEY-----\r\n***\r\n-----END EC PRIVATE KEY-----\r\n",
		},
		{
			name:  "unterminated block redacts to end of input",
			input: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA\nqqLNM5pFz",
			want:  "-----BEGIN RSA PRIVATE KEY-----\n***",
		},
		{
			name:  "certificate block is public and stays visible",
			input: "-----BEGIN CERTIFICATE-----\nMIIC\n-----END CERTIFICATE-----",
			want:  "-----BEGIN CERTIFICATE-----\nMIIC\n-----END CERTIFICATE-----",
		},
		{
			name:  "public key block stays visible",
			input: "-----BEGIN PUBLIC KEY-----\nMFkw\n-----END PUBLIC KEY-----",
			want:  "-----BEGIN PUBLIC KEY-----\nMFkw\n-----END PUBLIC KEY-----",
		},
		{
			name:  "empty body left to the normal scan",
			input: "-----BEGIN RSA PRIVATE KEY-----\n-----END RSA PRIVATE KEY-----",
			want:  "-----BEGIN RSA PRIVATE KEY-----\n-----END RSA PRIVATE KEY-----",
		},
		{
			name:  "begin line at end of input without newline",
			input: "-----BEGIN RSA PRIVATE KEY-----",
			want:  "-----BEGIN RSA PRIVATE KEY-----",
		},
		{
			name:  "dashes without pem prefix are untouched",
			input: "----- not a pem line -----",
			want:  "----- not a pem line -----",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input))
		})
	}
}

// TestRedactInlinePEMPrivateKeys covers PEM blocks embedded mid-line — most
// commonly a JSON string value carrying a whole blob with escaped "\n"
// sequences — and the line-start blob regression (a BEGIN line with trailing
// content must not leak the blob nor swallow the following lines).
func TestRedactInlinePEMPrivateKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{
			`{"data":"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIB\n-----END RSA PRIVATE KEY-----"}`,
			`{"data":"-----BEGIN RSA PRIVATE KEY-----***-----END RSA PRIVATE KEY-----"}`,
		},
		{
			`cfg: -----BEGIN EC PRIVATE KEY-----\nMHcC\n-----END EC PRIVATE KEY----- done`,
			`cfg: -----BEGIN EC PRIVATE KEY-----***-----END EC PRIVATE KEY----- done`,
		},
		// Escaped blob at line start with a following unrelated line: the blob
		// is redacted and the next line is untouched.
		{
			"-----BEGIN RSA PRIVATE KEY-----\\nMIIE\\n-----END RSA PRIVATE KEY-----\nHello world",
			"-----BEGIN RSA PRIVATE KEY-----***-----END RSA PRIVATE KEY-----\nHello world",
		},
		// Unterminated inline blob inside a JSON string stops at the quote.
		{
			`{"data":"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIB","next":"VISIBLE"}`,
			`{"data":"-----BEGIN RSA PRIVATE KEY-----***","next":"VISIBLE"}`,
		},
		// Unterminated inline blob in prose stops at the line end.
		{
			"key -----BEGIN RSA PRIVATE KEY-----\\nMIIE\nnext=ok",
			"key -----BEGIN RSA PRIVATE KEY-----***\nnext=ok",
		},
		// Public blocks stay visible mid-line too (under a non-sensitive key;
		// "cert" itself is a sensitive token and would redact the whole value).
		{
			`{"material":"-----BEGIN CERTIFICATE-----\nMIIC\n-----END CERTIFICATE-----"}`,
			`{"material":"-----BEGIN CERTIFICATE-----\nMIIC\n-----END CERTIFICATE-----"}`,
		},
		// Whitespace-only body: nothing to hide.
		{"x -----BEGIN RSA PRIVATE KEY----- \ny", "x -----BEGIN RSA PRIVATE KEY----- \ny"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)

		once := Default().String(tc.input)
		require.Equal(t, once, Default().String(once), "not idempotent for input: %s", tc.input)
	}
}
