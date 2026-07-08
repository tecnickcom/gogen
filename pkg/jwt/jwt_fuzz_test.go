package jwt

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/httputil"
)

// FuzzLoginHandler feeds arbitrary bodies to the login endpoint: it must never
// panic and must only ever answer with one of the defined status codes. A 500
// is impossible with a non-failing verifier and a valid signing setup, so any
// occurrence is a bug.
func FuzzLoginHandler(f *testing.F) {
	c, err := New(testKey, testVerify, WithLogger(slog.New(slog.DiscardHandler)))
	if err != nil {
		f.Fatal(err)
	}

	seeds := []string{
		"",
		`{"broken":"...`,
		`{"username":"test-name", "password":"test-name"}`,
		`{"username":"test-name", "password":"wrong"}`,
		`{"username":"a","password":"b"}{"x":1}`,
		`{"username":"test-name", "password":"test-name", "extra":"x"}`,
		`[1,2,3]`,
		"\x00\xff\xfe",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		rr := httptest.NewRecorder()

		req, rerr := http.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(body))
		require.NoError(t, rerr)

		c.LoginHandler(rr, req)

		switch rr.Code {
		case http.StatusOK,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusRequestEntityTooLarge:
		default:
			t.Fatalf("unexpected status %d for body %q", rr.Code, body)
		}
	})
}

// FuzzCheckToken feeds arbitrary Authorization header values to the token
// checker: it must never panic, must always return non-nil claims, and must
// only accept structurally valid tokens correctly signed with the secret key
// (which the fuzzer cannot forge; acceptances can therefore only come from
// mutations of the seeded valid token that keep its signed segments intact).
func FuzzCheckToken(f *testing.F) {
	c, err := New(testKey, testVerify, WithLogger(slog.New(slog.DiscardHandler)))
	if err != nil {
		f.Fatal(err)
	}

	validToken, err := c.signToken(c.newClaims("fuzz-user", nil))
	if err != nil {
		f.Fatal(err)
	}

	seeds := []string{
		"",
		"Basic abc",
		"Bearer",
		"Bearer ",
		"Bearer abc.def.ghi",
		"bearer x",
		httputil.HeaderAuthBearer + validToken,
		"BEARER  " + validToken,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, header string) {
		req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		require.NoError(t, rerr)

		if header != "" {
			req.Header.Set(DefaultAuthorizationHeader, header)
		}

		claims, cerr := c.checkToken(req)
		require.NotNil(t, claims, "claims must never be nil")

		if cerr == nil {
			// Anything accepted must be a complete, validated token.
			require.NotNil(t, claims.ExpiresAt, "accepted tokens must carry exp")
		}
	})
}
