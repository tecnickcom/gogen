package jwt

import (
	"context"
	"net/http"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
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

	v := func(_ context.Context, _ http.ResponseWriter, _ int, _ string) {}
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

func TestWithClaimIssuer(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := "Test_Issuer_01"
	WithClaimIssuer(want)(c)
	require.Equal(t, want, c.issuer)
}

func TestWithClaimSubject(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := "Test_Subject_02"
	WithClaimSubject(want)(c)
	require.Equal(t, want, c.subject)
}

func TestWithClaimAudience(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := []string{"Audience_01", "Audience_02"}
	WithClaimAudience(want)(c)
	require.Equal(t, want, c.audience)
}
