package jwt

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

func TestWithExpirationTime(t *testing.T) {
	t.Parallel()

	var v time.Duration

	c := defaultJWT()

	v = 503 * time.Millisecond
	WithExpirationTime(v)(c)
	require.Equal(t, v, c.expirationTime)
}

func TestWithRenewTime(t *testing.T) {
	t.Parallel()

	var v time.Duration

	c := defaultJWT()

	v = 703 * time.Millisecond
	WithRenewTime(v)(c)
	require.Equal(t, v, c.renewTime)
}

func TestWithSendResponseFn(t *testing.T) {
	t.Parallel()

	c := &JWT{}

	v := func(ctx context.Context, w http.ResponseWriter, statusCode int, data string) {}
	WithSendResponseFn(v)(c)

	require.NotNil(t, v, c.sendResponseFn)
}

func TestWithAuthorizationHeader(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := "Authorization-Header-Name"
	WithAuthorizationHeader(want)(c)
	require.Equal(t, want, c.authorizationHeader)
}

func TestWithSigningMethod(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := jwt.SigningMethodHS384
	WithSigningMethod(want)(c)
	require.Equal(t, want, c.signingMethod)
}
