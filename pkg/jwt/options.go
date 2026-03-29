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

// WithSendResponseFn overrides how auth handlers write HTTP responses.
func WithSendResponseFn(sendResponseFn SendResponseFn) Option {
	return func(c *JWT) {
		c.sendResponseFn = sendResponseFn
	}
}

// WithAuthorizationHeader sets the header key used to read bearer tokens.
func WithAuthorizationHeader(authorizationHeader string) Option {
	return func(c *JWT) {
		c.authorizationHeader = authorizationHeader
	}
}

// WithSigningMethod sets the JWT signing algorithm.
func WithSigningMethod(signingMethod SigningMethod) Option {
	return func(c *JWT) {
		c.signingMethod = signingMethod
	}
}

// WithClaimIssuer sets the `iss` (Issuer) JWT claim.
// See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.1
func WithClaimIssuer(issuer string) Option {
	return func(c *JWT) {
		c.issuer = issuer
	}
}

// WithClaimSubject sets the `sub` (Subject) claim.
// See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.2
func WithClaimSubject(subject string) Option {
	return func(c *JWT) {
		c.subject = subject
	}
}

// WithClaimAudience sets the `aud` (Audience) claim.
// See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
func WithClaimAudience(audience []string) Option {
	return func(c *JWT) {
		c.audience = audience
	}
}

// WithLogger sets the logger used by authentication handlers.
func WithLogger(logger *slog.Logger) Option {
	return func(c *JWT) {
		c.logger = logger
	}
}
