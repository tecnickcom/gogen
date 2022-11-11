package jwt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        []byte
		userHashFn UserHashFn
		opts       []Option
		wantErr    bool
	}{
		{
			name:       "success with default options",
			key:        []byte("test-key-01"),
			userHashFn: func(username string) ([]byte, error) { return []byte("hash-01"), nil },
			wantErr:    false,
		},
		{
			name:       "success with custom options",
			key:        []byte("test-key-02"),
			userHashFn: func(username string) ([]byte, error) { return []byte("hash-02"), nil },
			opts: []Option{
				WithExpirationTime(1 * time.Minute),
				WithRenewTime(10 * time.Second),
				WithSendResponseFn(func(ctx context.Context, w http.ResponseWriter, statusCode int, data string) {}),
			},
			wantErr: false,
		},
		{
			name:       "failure with empty key",
			userHashFn: func(username string) ([]byte, error) { return []byte("hash-01"), nil },
			wantErr:    true,
		},
		{
			name:    "failure with empty userHashFn",
			key:     []byte("test-key-01"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(tt.key, tt.userHashFn, tt.opts...)
			if tt.wantErr {
				require.Nil(t, c, "New() returned value should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.NotNil(t, c, "New() returned value should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func TestLoginHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		body          string
		want          string
		key           []byte
		status        int
		signingMethod SigningMethod
	}{
		{
			name:   "fails with empty body",
			key:    []byte("signing-key"),
			want:   "EOF",
			status: http.StatusBadRequest,
		},
		{
			name:   "fails with invalid body",
			key:    []byte("signing-key"),
			body:   `{"broken":"...`,
			want:   "unexpected EOF",
			status: http.StatusBadRequest,
		},
		{
			name:   "fails with invalid username",
			key:    []byte("signing-key"),
			body:   `{"username":"", "password":"test-secret"}`,
			want:   "invalid authentication credentials",
			status: http.StatusUnauthorized,
		},
		{
			name:   "fails with empty password",
			key:    []byte("signing-key"),
			body:   `{"username":"test-name", "password":""}`,
			want:   "invalid authentication credentials",
			status: http.StatusUnauthorized,
		},
		{
			name:   "fails with invalid password",
			key:    []byte("signing-key"),
			body:   `{"username":"test-name", "password":"invalid-password"}`,
			want:   "invalid authentication credentials",
			status: http.StatusUnauthorized,
		},
		{
			name:          "fails with signing error",
			key:           []byte("signing-key"),
			body:          `{"username":"test-name", "password":"test-name"}`,
			want:          "unable to sign the JWT token",
			status:        http.StatusInternalServerError,
			signingMethod: &testSigningMethodError{},
		},
		{
			name:   "success",
			key:    []byte("signing-key"),
			body:   `{"username":"test-name", "password":"test-name"}`,
			want:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QtbmFtZSIsImV4cCI6MTYxNzE5MjY1OX0.PfE7ulkdDhCDkQrj3NwqF4bw7K4f2QbTs4rcLvaJrtM",
			status: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts []Option

			if tt.signingMethod != nil {
				opts = append(opts, WithSigningMethod(tt.signingMethod))
			}

			c, err := New(tt.key, testUserHash, opts...)
			require.NotNil(t, c)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", strings.NewReader(tt.body))
			c.LoginHandler(rr, req)

			resp := rr.Result() //nolint:bodyclose
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			body, _ := io.ReadAll(resp.Body)

			require.Equal(t, tt.status, resp.StatusCode)
			if tt.status != http.StatusOK {
				require.Equal(t, tt.want, string(body))
			} else {
				require.Greater(t, len(body), 100)
			}
		})
	}
}

func TestRenewHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		status              int
		expirationTime      time.Duration
		authorizationHeader string
		bearerHeader        string
		badToken            bool
	}{
		{
			name:                "unauthorized",
			status:              http.StatusUnauthorized,
			expirationTime:      1 * time.Second,
			authorizationHeader: DefaultAuthorizationHeader,
			bearerHeader:        bearerHeader,
			badToken:            true,
		},
		{
			name:                "wrong authorization header",
			status:              http.StatusUnauthorized,
			expirationTime:      1 * time.Second,
			authorizationHeader: "ERROR",
			bearerHeader:        bearerHeader,
		},
		{
			name:                "wrong authorization value",
			status:              http.StatusUnauthorized,
			expirationTime:      1 * time.Second,
			authorizationHeader: DefaultAuthorizationHeader,
			bearerHeader:        "ERROR",
		},
		{
			name:                "too early",
			status:              http.StatusBadRequest,
			expirationTime:      5 * time.Second,
			authorizationHeader: DefaultAuthorizationHeader,
			bearerHeader:        bearerHeader,
		},
		{
			name:                "success",
			status:              http.StatusOK,
			expirationTime:      1 * time.Second,
			authorizationHeader: DefaultAuthorizationHeader,
			bearerHeader:        bearerHeader,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(
				[]byte("signing-key"),
				testUserHash,
				WithExpirationTime(tt.expirationTime),
				WithRenewTime(1*time.Second),
			)
			require.NotNil(t, c)
			require.NoError(t, err)

			reqBody := `{"username":"test-name", "password":"test-name"}`

			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", strings.NewReader(reqBody))
			c.LoginHandler(rr, req)

			resp := rr.Result() //nolint:bodyclose
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			token, _ := io.ReadAll(resp.Body)

			rr2 := httptest.NewRecorder()
			req2, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)

			header := tt.bearerHeader + string(token)

			if tt.badToken {
				header += "CORRUPT"
			}

			req2.Header.Set(tt.authorizationHeader, header)
			c.RenewHandler(rr2, req2)

			resp2 := rr2.Result() //nolint:bodyclose
			require.NotNil(t, resp2)

			defer func() {
				err := resp2.Body.Close()
				require.NoError(t, err, "error closing resp2.Body")
			}()

			require.Equal(t, tt.status, resp2.StatusCode)
		})
	}
}

func TestIsAuthorized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		status              int
		authorizationHeader string
		bearerHeader        string
		badToken            bool
	}{
		{
			name:                "unauthorized",
			status:              http.StatusUnauthorized,
			authorizationHeader: DefaultAuthorizationHeader,
			bearerHeader:        bearerHeader,
			badToken:            true,
		},
		{
			name:                "wrong authorization header",
			status:              http.StatusUnauthorized,
			authorizationHeader: "ERROR",
			bearerHeader:        bearerHeader,
		},
		{
			name:                "wrong authorization value",
			status:              http.StatusUnauthorized,
			authorizationHeader: DefaultAuthorizationHeader,
			bearerHeader:        "ERROR",
		},
		{
			name:                "success",
			status:              0,
			authorizationHeader: DefaultAuthorizationHeader,
			bearerHeader:        bearerHeader,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(
				[]byte("signing-key"),
				testUserHash,
			)
			require.NotNil(t, c)
			require.NoError(t, err)

			reqBody := `{"username":"test-name", "password":"test-name"}`

			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", strings.NewReader(reqBody))
			c.LoginHandler(rr, req)

			resp := rr.Result() //nolint:bodyclose
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			token, _ := io.ReadAll(resp.Body)

			rr2 := httptest.NewRecorder()
			req2, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)

			header := tt.bearerHeader + string(token)

			if tt.badToken {
				header += "CORRUPT"
			}

			req2.Header.Set(tt.authorizationHeader, header)
			got := c.IsAuthorized(rr2, req2)

			if tt.status == 0 {
				require.True(t, got)
			} else {
				resp2 := rr2.Result() //nolint:bodyclose
				require.NotNil(t, resp2)

				defer func() {
					err := resp2.Body.Close()
					require.NoError(t, err, "error closing resp2.Body")
				}()

				require.Equal(t, tt.status, resp2.StatusCode)
			}
		})
	}
}

// testUserHash assumes password = username.
func testUserHash(username string) ([]byte, error) {
	if username == "" {
		return nil, fmt.Errorf("invalid username")
	}

	return bcrypt.GenerateFromPassword([]byte(username), bcrypt.MinCost) //nolint:wrapcheck
}

type testSigningMethodError struct{}

func (c *testSigningMethodError) Verify(signingString, signature string, key interface{}) error {
	return fmt.Errorf("VERIFY ERROR")
}

func (c *testSigningMethodError) Sign(signingString string, key interface{}) (string, error) {
	return "", fmt.Errorf("SIGN ERROR")
}

func (c *testSigningMethodError) Alg() string {
	return ""
}
