package jwt

import (
	"log/slog"
	"time"
)

// Option is the interface that allows to set the options.
type Option func(c *JWT)

// WithExpirationTime sets the token lifetime from issuance to expiry.
func WithExpirationTime(expirationTime time.Duration) Option {
	return func(c *JWT) {
		c.expirationTime = expirationTime
	}
}

// WithRenewTime sets how close to expiry renewal becomes allowed.
func WithRenewTime(renewTime time.Duration) Option {
	return func(c *JWT) {
		c.renewTime = renewTime
	}
}

// WithMaxSessionLifetime caps how long a session may be kept alive through
// renewals, measured from the original login (`auth_time`). Once exceeded,
// [JWT.RenewHandler] refuses to renew and the user must log in again. Zero (the
// default) means unlimited; a negative value is rejected by [New].
func WithMaxSessionLifetime(maxSessionLifetime time.Duration) Option {
	return func(c *JWT) {
		c.maxSessionLifetime = maxSessionLifetime
	}
}

// WithClockSkewLeeway sets the tolerance applied to the validated time claims
// (exp and nbf) during verification, to absorb clock skew between hosts. Zero
// (the default) applies no leeway; a negative value is rejected by [New].
func WithClockSkewLeeway(clockSkewLeeway time.Duration) Option {
	return func(c *JWT) {
		c.clockSkewLeeway = clockSkewLeeway
	}
}

// WithMaxBodyBytes sets the maximum accepted size, in bytes, of a login request
// body. It must be positive; see [DefaultMaxBodyBytes] for the default.
func WithMaxBodyBytes(maxBodyBytes int64) Option {
	return func(c *JWT) {
		c.maxBodyBytes = maxBodyBytes
	}
}

// WithSendResponseFn overrides how auth handlers write HTTP responses. A nil
// function restores the default plain-text responder.
func WithSendResponseFn(sendResponseFn SendResponseFn) Option {
	return func(c *JWT) {
		c.sendResponseFn = sendResponseFn
	}
}

// WithAuthorizationHeader sets the header key used to read bearer tokens. An
// empty name restores the default ([DefaultAuthorizationHeader]).
func WithAuthorizationHeader(authorizationHeader string) Option {
	return func(c *JWT) {
		c.authorizationHeader = authorizationHeader
	}
}

// WithSigningMethod sets the HMAC signing algorithm ([SigningMethodHS256],
// [SigningMethodHS384] or [SigningMethodHS512]); any other value is rejected by
// [New]. The default is [SigningMethodHS256].
func WithSigningMethod(signingMethod SigningMethod) Option {
	return func(c *JWT) {
		c.signingMethod = signingMethod
	}
}

// WithPreviousKeys registers previous signing keys that remain accepted for
// verification during a key-rotation window. New tokens are always signed with
// the current key; tokens signed with a previous key keep verifying until they
// expire. Each previous key must satisfy the same minimum-length rule as the
// signing key (at least the signing method's hash output size), enforced by
// [New]. Remove previous keys once the rotation window (expiration time plus
// renew window) has elapsed. The slice is copied here, and [New] also copies
// the key bytes themselves.
func WithPreviousKeys(previousKeys ...[]byte) Option {
	return func(c *JWT) {
		c.previousKeys = append([][]byte(nil), previousKeys...)
	}
}

// WithClaimIssuer sets the `iss` (Issuer) JWT claim. An empty string (the
// default) disables both issuing and enforcing the claim.
// See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.1
func WithClaimIssuer(issuer string) Option {
	return func(c *JWT) {
		c.issuer = issuer
	}
}

// WithClaimAudience sets the `aud` (Audience) claim. When set, every configured
// audience must also be present in a token for it to be accepted. An empty or
// nil slice (the default) disables both issuing and enforcing the claim. The
// slice is copied so later mutation by the caller does not affect the
// configuration.
// See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
func WithClaimAudience(audience []string) Option {
	return func(c *JWT) {
		c.audience = append([]string(nil), audience...)
	}
}

// WithLogger sets the logger used by authentication handlers. A nil logger
// restores slog.Default().
func WithLogger(logger *slog.Logger) Option {
	return func(c *JWT) {
		c.logger = logger
	}
}
