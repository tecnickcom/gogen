// Package jwt provides JWT Auth handlers.
package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultExpirationTime is the default JWT expiration time.
	DefaultExpirationTime = 5 * time.Minute

	// DefaultRenewTime is the default time before the JWT expiration when the renewal is allowed.
	DefaultRenewTime = 30 * time.Second

	// DefaultAuthorizationHeader is the default authorization header name.
	DefaultAuthorizationHeader = "Authorization"

	bearerHeader = "Bearer "
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
	Username string `json:"username"`
	jwt.StandardClaims
}

// JWT represents an instance of the HTTP retrier.
type JWT struct {
	key                 []byte         // JWT signing key.
	expirationTime      time.Duration  // JWT expiration time.
	renewTime           time.Duration  // Time before the JWT expiration when the renewal is allowed.
	sendResponseFn      SendResponseFn // Response function used to send back the HTTP responses.
	userHashFn          UserHashFn     // Function used to retrieve the password hash associated with each user.
	signingMethod       SigningMethod  // Signing Method function
	authorizationHeader string
}

func defaultJWT() *JWT {
	return &JWT{
		expirationTime:      DefaultExpirationTime,
		renewTime:           DefaultRenewTime,
		sendResponseFn:      defaultSendResponse,
		authorizationHeader: DefaultAuthorizationHeader,
		signingMethod:       defaultSigningMethod(),
	}
}

func defaultSendResponse(ctx context.Context, w http.ResponseWriter, statusCode int, data string) {
	httputil.SendText(ctx, w, statusCode, data)
}

func defaultSigningMethod() SigningMethod {
	return jwt.SigningMethodHS256
}

// New creates a new instance.
func New(key []byte, userHashFn UserHashFn, opts ...Option) (*JWT, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("empty JWT key")
	}

	if userHashFn == nil {
		return nil, fmt.Errorf("empty user hash function")
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

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		c.sendResponseFn(r.Context(), w, http.StatusBadRequest, err.Error())
		logging.FromContext(r.Context()).Error("invalid JWT body", zap.Error(err))

		return
	}

	hash, err := c.userHashFn(creds.Username)
	if err != nil {
		// invalid user
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, "invalid authentication credentials")
		logging.FromContext(r.Context()).With(
			zap.String("username", creds.Username),
		).Error("invalid JWT username", zap.Error(err))

		return
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(creds.Password))
	if err != nil {
		// invalid password
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, "invalid authentication credentials")
		logging.FromContext(r.Context()).With(
			zap.String("username", creds.Username),
		).Error("invalid JWT password", zap.Error(err))

		return
	}

	exp := time.Now().Add(c.expirationTime)
	claims := Claims{
		Username: creds.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp.Unix(),
		},
	}

	c.sendTokenResponse(w, r, &claims)
}

// RenewHandler handles the JWT renewal endpoint.
func (c *JWT) RenewHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := c.checkToken(r)
	if err != nil {
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, err.Error())
		logging.FromContext(r.Context()).With(
			zap.String("username", claims.Username),
		).Error("invalid JWT token", zap.Error(err))

		return
	}

	if time.Until(time.Unix(claims.ExpiresAt, 0)) > c.renewTime {
		c.sendResponseFn(r.Context(), w, http.StatusBadRequest, "the JWT token can be renewed only when it is close to expiration")
		logging.FromContext(r.Context()).With(
			zap.String("username", claims.Username),
		).Error("invalid JWT renewal time", zap.Error(err))

		return
	}

	c.sendTokenResponse(w, r, claims)
}

// IsAuthorized checks if the user is authorized via JWT token.
func (c *JWT) IsAuthorized(w http.ResponseWriter, r *http.Request) bool {
	claims, err := c.checkToken(r)
	if err != nil {
		c.sendResponseFn(r.Context(), w, http.StatusUnauthorized, err.Error())
		logging.FromContext(r.Context()).With(
			zap.String("username", claims.Username),
		).Error("unauthorized JWT user", zap.Error(err))

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
		logging.FromContext(r.Context()).With(
			zap.String("username", claims.Username),
		).Error("unable to sign the JWT token", zap.Error(err))

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

	authSplit := strings.Split(headAuth, bearerHeader)
	if len(authSplit) != 2 {
		return claims, errors.New("missing JWT token")
	}

	signedToken := authSplit[1]

	_, err := jwt.ParseWithClaims(
		signedToken,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return c.key, nil
		},
	)

	return claims, err // nolint:wrapcheck
}
