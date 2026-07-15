package jwt

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

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

	require.NotNil(t, c.sendResponseFn)
}

func TestWithMaxSessionLifetime(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := 42 * time.Minute
	WithMaxSessionLifetime(want)(c)
	require.Equal(t, want, c.maxSessionLifetime)
}

func TestWithClockSkewLeeway(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := 7 * time.Second
	WithClockSkewLeeway(want)(c)
	require.Equal(t, want, c.clockSkewLeeway)
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
	want := SigningMethodHS384
	WithSigningMethod(want)(c)
	require.Equal(t, want, c.signingMethod)
}

func TestWithPreviousKeys(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	keyA := []byte("previous-key-A")
	keyB := []byte("previous-key-B")
	WithPreviousKeys(keyA, keyB)(c)
	require.Equal(t, [][]byte{keyA, keyB}, c.previousKeys)
}

func TestWithClaimIssuer(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := "Test_Issuer_01"
	WithClaimIssuer(want)(c)
	require.Equal(t, want, c.issuer)
}

func TestWithMaxBodyBytes(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := int64(4096)
	WithMaxBodyBytes(want)(c)
	require.Equal(t, want, c.maxBodyBytes)
}

func TestWithMaxTokenBytes(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := 4096
	WithMaxTokenBytes(want)(c)
	require.Equal(t, want, c.maxTokenBytes)
}

func TestWithClaimAudience(t *testing.T) {
	t.Parallel()

	c := &JWT{}
	want := []string{"Audience_01", "Audience_02"}
	WithClaimAudience(want)(c)
	require.Equal(t, want, c.audience)
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	c := &JWT{}

	want := slog.Default()
	WithLogger(want)(c)
	require.Equal(t, want, c.logger)
}
