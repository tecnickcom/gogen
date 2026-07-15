package jwt

import (
	"log/slog"
	"time"
)

// Option is the interface that allows to set the options.
type Option func(c *JWT)

// WithExpirationTime sets the token lifetime from issuance to expiry. It must be
// at least one second: token times are stamped at whole-second resolution, so a
// shorter (or non-positive) value is rejected by [New]. Prefer a value comfortably
// above one second; because `exp` is truncated to whole seconds, a token issued
// just before a second boundary loses up to that boundary, so at exactly one
// second its usable lifetime can be a fraction of a second.
func WithExpirationTime(expirationTime time.Duration) Option {
	return func(c *JWT) {
		c.expirationTime = expirationTime
	}
}

// WithRenewTime sets how close to expiry renewal becomes allowed: [JWT.RenewHandler]
// refuses a token whose remaining lifetime still exceeds this window. It must be
// positive, and should be shorter than the expiration time; a value greater than
// or equal to it makes every token renewable from issuance, so the window has no
// effect.
func WithRenewTime(renewTime time.Duration) Option {
	return func(c *JWT) {
		c.renewTime = renewTime
	}
}

// WithMaxSessionLifetime caps how long a session may be kept alive through
// renewals, measured from the original login (`auth_time`). Once exceeded,
// [JWT.RenewHandler] refuses to renew and the user must log in again. Zero (the
// default) means unlimited; a negative value, or a positive value shorter than
// one second (the whole-second resolution of token times), is rejected by [New].
//
// The effective absolute bound is maxSessionLifetime plus any [WithClockSkewLeeway]:
// the last renewed token is capped at auth_time+maxSessionLifetime, but the leeway
// is added when verifying its `exp`, so it remains accepted until
// auth_time+maxSessionLifetime+clockSkewLeeway.
func WithMaxSessionLifetime(maxSessionLifetime time.Duration) Option {
	return func(c *JWT) {
		c.maxSessionLifetime = maxSessionLifetime
	}
}

// WithClockSkewLeeway sets the tolerance applied to the validated time claims
// (exp and nbf) during verification, to absorb clock skew between hosts. Zero
// (the default) applies no leeway; a negative value is rejected by [New].
//
// Keep it small relative to the expiration time: because it is added to `exp`
// when verifying, it extends every token's effective lifetime, and a leeway at or
// above the expiration time keeps expired tokens accepted for a comparable extra
// span. It likewise widens the [WithMaxSessionLifetime] bound.
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

// WithMaxTokenBytes sets the maximum accepted size, in bytes, of a compact-JWS
// token presented for verification. It must be positive; see
// [DefaultMaxTokenBytes] for the default. Raise it only for unusually large
// custom claims: the cap bounds the base64 and JSON decode of the JOSE header,
// which is necessarily parsed before the signature can be checked.
func WithMaxTokenBytes(maxTokenBytes int) Option {
	return func(c *JWT) {
		c.maxTokenBytes = maxTokenBytes
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
