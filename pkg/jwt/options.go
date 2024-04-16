package jwt

import (
	"time"
)

// Option is the interface that allows to set the options.
type Option func(c *JWT)

// WithExpirationTime set the JWT expiration time.
func WithExpirationTime(expirationTime time.Duration) Option {
	return func(c *JWT) {
		c.expirationTime = expirationTime
	}
}

// WithRenewTime set the time before the JWT expiration when the renewal is allowed.
func WithRenewTime(renewTime time.Duration) Option {
	return func(c *JWT) {
		c.renewTime = renewTime
	}
}

// WithSendResponseFn set the function used to send back the HTTP responses.
func WithSendResponseFn(sendResponseFn SendResponseFn) Option {
	return func(c *JWT) {
		c.sendResponseFn = sendResponseFn
	}
}

// WithAuthorizationHeader sets the authorization header name.
func WithAuthorizationHeader(authorizationHeader string) Option {
	return func(c *JWT) {
		c.authorizationHeader = authorizationHeader
	}
}

// WithSigningMethod sets the signing method function.
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
