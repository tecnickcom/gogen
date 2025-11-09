/*
Package jwt provides simple wrapper functions for managing basic JWT
Authentication with username/password credentials.

The package is designed to be used in conjunction with the net/http package in
the Go standard library. It includes functions for handling login, renewal, and
authorization of JWT tokens.
*/
package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/uidc"
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultExpirationTime is the default JWT expiration time.
	DefaultExpirationTime = 5 * time.Minute

	// DefaultRenewTime is the default time before the JWT expiration when the renewal is allowed.
	DefaultRenewTime = 30 * time.Second

	// DefaultAuthorizationHeader is the default authorization header name.
	DefaultAuthorizationHeader = httputil.HeaderAuthorization
)

// SendResponseFn is the type of function used to send back the HTTP responses.
type SendResponseFn func(ctx context.Context, w http.ResponseWriter, statusCode int, data string)

// UserHashFn is the type of function used to retrieve the password hash associated with each user.
// The hash values should be generated via bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost).
type UserHashFn func(username string) ([]byte, error)

// SigningMethod is a type alias for the Signing Method interface.
type SigningMethod jwt.SigningMethod

// Credentials holds the user name and password from the request body.
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Claims holds the JWT information to be encoded.
type Claims struct {
	jwt.RegisteredClaims

	Username string `json:"username"`
}

// JWT represents an instance of the JWT object.
type JWT struct {
	key                 []byte         // JWT signing key.
	expirationTime      time.Duration  // JWT expiration time.
	renewTime           time.Duration  // Time before the JWT expiration when the renewal is allowed.
	sendResponseFn      SendResponseFn // Response function used to send back the HTTP responses.
	userHashFn          UserHashFn     // Function used to retrieve the password hash associated with each user.
	signingMethod       SigningMethod  // Signing Method function
	authorizationHeader string
	issuer              string   // the `iss` (Issuer) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.1
	subject             string   // the `sub` (Subject) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.2
	audience            []string // the `aud` (Audience) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
	logger              *slog.Logger
}

// defaultJWT creates a JWT instance with default values.
func defaultJWT() *JWT {
	return &JWT{
		expirationTime:      DefaultExpirationTime,
		renewTime:           DefaultRenewTime,
		sendResponseFn:      defaultSendResponse,
		authorizationHeader: DefaultAuthorizationHeader,
		signingMethod:       defaultSigningMethod(),
		logger:              slog.Default(),
	}
}

// defaultSendResponse is the default function used to send back the HTTP responses.
func defaultSendResponse(ctx context.Context, w http.ResponseWriter, statusCode int, data string) {
	httputil.SendText(ctx, w, statusCode, data)
}

// defaultSigningMethod returns the default JWT signing method.
func defaultSigningMethod() SigningMethod {
	return jwt.SigningMethodHS256
}

// New creates a new instance.
func New(key []byte, userHashFn UserHashFn, opts ...Option) (*JWT, error) {
	if len(key) == 0 {
		return nil, errors.New("empty JWT key")
	}

	if userHashFn == nil {
		return nil, errors.New("empty user hash function")
	}

	c := defaultJWT()
	c.key = key
	c.userHashFn = userHashFn

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	return c, nil
}

// LoginHandler handles the login endpoint.
func (c *JWT) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	defer func() {
		cerr := r.Body.Close()
		if cerr != nil {
			c.logger.With(slog.Any("error", cerr)).Error("error closing request body")
		}
	}()

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		c.sendResponseFn(r.Context(), w, http.StatusBadRequest, err.Error())
		c.logger.With(slog.Any("error", err)).Error("invalid JWT body")

		return
	}

	hash, err := c.userHashFn(creds.Username)
	if err != nil {
		// invalid user
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, "invalid authentication credentials")
		c.logger.With(
			slog.String("username", creds.Username),
			slog.Any("error", err),
		).Error("invalid JWT username")

		return
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(creds.Password))
	if err != nil {
		// invalid password
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, "invalid authentication credentials")
		c.logger.With(
			slog.String("username", creds.Username),
			slog.Any("error", err),
		).Error("invalid JWT password")

		return
	}

	tnow := time.Now().UTC()
	claims := Claims{
		Username: creds.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(tnow.Add(c.expirationTime)), // exp
			IssuedAt:  jwt.NewNumericDate(tnow),                       // iat
			NotBefore: jwt.NewNumericDate(tnow),                       // nbf
			ID:        uidc.NewID128(),                                // jti
			Issuer:    c.issuer,                                       // iss
			Subject:   c.subject,                                      // sub
			Audience:  c.audience,                                     // aud
		},
	}

	c.sendTokenResponse(w, r, &claims)
}

// RenewHandler handles the JWT renewal endpoint.
func (c *JWT) RenewHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := c.checkToken(r)
	if err != nil {
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, err.Error())
		c.logger.With(
			slog.String("username", claims.Username),
			slog.Any("error", err),
		).Error("invalid JWT token")

		return
	}

	if time.Until(claims.ExpiresAt.Time) > c.renewTime {
		c.sendResponseFn(r.Context(), w, http.StatusBadRequest, "the JWT token can be renewed only when it is close to expiration")
		c.logger.With(
			slog.String("username", claims.Username),
			slog.Any("error", err),
		).Error("invalid JWT renewal time")

		return
	}

	c.sendTokenResponse(w, r, claims)
}

// IsAuthorized checks if the user is authorized via JWT token.
func (c *JWT) IsAuthorized(w http.ResponseWriter, r *http.Request) bool {
	claims, err := c.checkToken(r)
	if err != nil {
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, err.Error())
		c.logger.With(
			slog.String("username", claims.Username),
			slog.Any("error", err),
		).Error("unauthorized JWT user")

		return false
	}

	return true
}

// sendTokenResponse sends the signed JWT token if claims are valid.
func (c *JWT) sendTokenResponse(w http.ResponseWriter, r *http.Request, claims *Claims) {
	token := jwt.NewWithClaims(c.signingMethod, claims)

	signedToken, err := token.SignedString(c.key)
	if err != nil {
		c.sendResponseFn(r.Context(), w, http.StatusInternalServerError, "unable to sign the JWT token")
		c.logger.With(
			slog.String("username", claims.Username),
			slog.Any("error", err),
		).Error("unable to sign the JWT token")

		return
	}

	c.sendResponseFn(r.Context(), w, http.StatusOK, signedToken)
}

// checkToken extracts the JWT token from the header "Authorization: Bearer <TOKEN>"
// and returns an error if the token is invalid.
func (c *JWT) checkToken(r *http.Request) (*Claims, error) {
	claims := &Claims{}

	headAuth := r.Header.Get(c.authorizationHeader)
	if len(headAuth) == 0 {
		return claims, errors.New("missing Authorization header")
	}

	authSplit := strings.Split(headAuth, httputil.HeaderAuthBearer)
	if len(authSplit) != 2 {
		return claims, errors.New("missing JWT token")
	}

	signedToken := authSplit[1]

	_, err := jwt.ParseWithClaims(
		signedToken,
		claims,
		func(_ *jwt.Token) (any, error) {
			return c.key, nil
		},
	)

	return claims, err //nolint:wrapcheck
}
