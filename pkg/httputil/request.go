package httputil

import (
	"context"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/tecnickcom/gogen/pkg/encode"
)

// timeCtxKey is the type used for the context key to store request time.
type timeCtxKey string

// ReqTimeCtxKey is the Context key to retrieve the request time.
const ReqTimeCtxKey = timeCtxKey("request_time")

// Common HTTP headers and MIME types.
const (
	HeaderAuthorization = "Authorization"
	HeaderAuthBasic     = "Basic "
	HeaderAuthBearer    = "Bearer "
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	MimeTypeJSON        = "application/json"
)

// AddJsonHeaders sets application/json Accept and Content-Type headers on the request.
func AddJsonHeaders(r *http.Request) {
	r.Header.Set(HeaderAccept, MimeTypeJSON)
	r.Header.Set(HeaderContentType, MimeTypeJSON)
}

// AddAuthorizationHeader adds Authorization header with specified value to request.
func AddAuthorizationHeader(auth string, r *http.Request) {
	r.Header.Add(HeaderAuthorization, auth)
}

// AddBasicAuth adds Basic Authorization header with base64-encoded "apiKey:apiSecret" credentials.
func AddBasicAuth(apiKey, apiSecret string, r *http.Request) {
	AddAuthorizationHeader(HeaderAuthBasic+encode.Base64EncodeString(apiKey+":"+apiSecret), r)
}

// AddBearerToken adds Bearer Authorization header with the provided token.
func AddBearerToken(token string, r *http.Request) {
	AddAuthorizationHeader(HeaderAuthBearer+token, r)
}

// PathParam returns the value of a named path segment from httprouter params in request context.
func PathParam(r *http.Request, name string) string {
	v := httprouter.ParamsFromContext(r.Context()).ByName(name)
	return strings.TrimLeft(v, "/")
}

// HeaderOrDefault returns HTTP header value or returns defaultValue if header is not set.
func HeaderOrDefault(r *http.Request, key string, defaultValue string) string {
	return StringValueOrDefault(r.Header.Get(key), defaultValue)
}

// QueryStringOrDefault returns query parameter value or defaultValue if missing or empty.
func QueryStringOrDefault(q url.Values, key string, defaultValue string) string {
	return StringValueOrDefault(q.Get(key), defaultValue)
}

// QueryIntOrDefault parses query parameter as signed integer or returns defaultValue if missing or invalid.
func QueryIntOrDefault(q url.Values, key string, defaultValue int) int {
	v, err := strconv.ParseInt(q.Get(key), 10, 64)
	if err == nil && v >= math.MinInt && v <= math.MaxInt {
		return int(v)
	}

	return defaultValue
}

// QueryUintOrDefault parses query parameter as unsigned integer or returns defaultValue if missing or invalid.
func QueryUintOrDefault(q url.Values, key string, defaultValue uint) uint {
	v, err := strconv.ParseUint(q.Get(key), 10, 64)
	if err == nil && v <= math.MaxUint {
		return uint(v)
	}

	return defaultValue
}

// WithRequestTime returns new context with request timestamp attached via ReqTimeCtxKey.
func WithRequestTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, ReqTimeCtxKey, t)
}

// GetRequestTimeFromContext retrieves request timestamp from context, returning zero time if not present.
func GetRequestTimeFromContext(ctx context.Context) (time.Time, bool) {
	v := ctx.Value(ReqTimeCtxKey)
	t, ok := v.(time.Time)

	return t, ok
}

// GetRequestTime returns the request time from the http request.
func GetRequestTime(r *http.Request) (time.Time, bool) {
	return GetRequestTimeFromContext(r.Context())
}

// StringValueOrDefault returns the string value or a default value.
func StringValueOrDefault(v, def string) string {
	if v != "" {
		return v
	}

	return def
}
