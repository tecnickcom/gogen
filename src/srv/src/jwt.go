package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"

	log "github.com/sirupsen/logrus"
)

// JwtData store a single JWT configuration
type JwtData struct {
	Enabled   bool   `json:"enabled"`   // Enable or disable JWT authentication
	Key       []byte `json:"key"`       // JWT signing key
	Exp       int    `json:"exp"`       // JWT expiration time in minutes
	RenewTime int    `json:"renewTime"` // Time in second before the JWT expiration time when the renewal is allowed
}

// Credentials holds the user name and password from the request body
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Claims holds the JWT information to be encoded
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// loginHandler handles the /auth/login
func loginHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.auth.login.in")
	defer stats.Increment("http.auth.login.out")
	log.Debug("handler: loginHandler")

	var creds Credentials
	err := json.NewDecoder(hr.Body).Decode(&creds)
	if err != nil {
		sendResponse(rw, hr, ps, http.StatusBadRequest, err.Error())
		return
	}
	hash, ok := appParams.user[creds.Username]
	if !ok {
		// invalid user
		sendResponse(rw, hr, ps, http.StatusUnauthorized, "invalid authentication credentials")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(creds.Password))
	if err != nil {
		// invalid password
		sendResponse(rw, hr, ps, http.StatusUnauthorized, "invalid authentication credentials")
		return
	}

	exp := time.Now().Add(time.Duration(appParams.jwt.Exp) * time.Minute)
	claims := &Claims{
		Username: creds.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(appParams.jwt.Key)
	if err != nil {
		sendResponse(rw, hr, ps, http.StatusInternalServerError, "unable to sign the JWT token")
		return
	}

	sendResponse(rw, hr, ps, http.StatusOK, signedToken)
}

// checkJwtToken extract the JWT token from the header "Authorization: Bearer <TOKEN>"
// and returns an error if the token is invalid.
func checkJwtToken(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) (*Claims, error) {
	headAuth := hr.Header.Get("Authorization")
	if len(headAuth) == 0 {
		return nil, errors.New("missing Authorization header")
	}
	authSplit := strings.Split(headAuth, "Bearer ")
	if len(authSplit) != 2 {
		return nil, errors.New("missing JWT token")
	}
	signedToken := authSplit[1]
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(signedToken, claims, func(token *jwt.Token) (interface{}, error) {
		return appParams.jwt.Key, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid JWT token")
	}
	return claims, nil
}

// renewJwtHandler handles the /auth/refresh
func renewJwtHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.auth.refresh.in")
	defer stats.Increment("http.auth.refresh.out")
	log.Debug("handler: renewJwtHandler")

	claims, err := checkJwtToken(rw, hr, ps)
	if err != nil {
		sendResponse(rw, hr, ps, http.StatusUnauthorized, err.Error())
		return
	}

	if time.Until(time.Unix(claims.ExpiresAt, 0)) > time.Duration(appParams.jwt.RenewTime)*time.Second {
		sendResponse(rw, hr, ps, http.StatusBadRequest, "the JWT token can be renewed only when it is close to expiration")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(appParams.jwt.Key)
	if err != nil {
		sendResponse(rw, hr, ps, http.StatusInternalServerError, "unable to sign the JWT token")
		return
	}

	sendResponse(rw, hr, ps, http.StatusOK, signedToken)
}

// isAuthorized checks if the user is authorized via JWT token
func isAuthorized(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) bool {
	if !appParams.jwt.Enabled {
		return true
	}
	_, err := checkJwtToken(rw, hr, ps)
	if err != nil {
		sendResponse(rw, hr, ps, http.StatusUnauthorized, err.Error())
		return false
	}
	return true
}
