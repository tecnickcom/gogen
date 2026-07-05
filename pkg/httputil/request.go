package httputil

import (
	"cmp"
	"context"
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

// AddJSONHeaders sets application/json Accept and Content-Type headers on the request.
func AddJSONHeaders(r *http.Request) {
	r.Header.Set(HeaderAccept, MimeTypeJSON)
	r.Header.Set(HeaderContentType, MimeTypeJSON)
}

// AddAuthorizationHeader sets the Authorization header to the specified value,
// replacing any previously set value. The Authorization header is a singleton
// per RFC 9110 §11.6.2, so repeated calls (e.g. refreshing a token before a
// retry) must not accumulate multiple values.
func AddAuthorizationHeader(auth string, r *http.Request) {
	r.Header.Set(HeaderAuthorization, auth)
}

// AddBasicAuth adds Basic Authorization header with base64-encoded "apiKey:apiSecret" credentials.
//
// Per RFC 7617 the user-id (apiKey) must not contain a colon, as the first colon
// separates the user-id from the password when the credentials are decoded.
func AddBasicAuth(apiKey, apiSecret string, r *http.Request) {
	AddAuthorizationHeader(HeaderAuthBasic+encode.Base64EncodeString(apiKey+":"+apiSecret), r)
}

// AddBearerToken adds Bearer Authorization header with the provided token.
func AddBearerToken(token string, r *http.Request) {
	AddAuthorizationHeader(HeaderAuthBearer+token, r)
}

// PathParam returns the value of a named path segment from httprouter params in request context.
//
// httprouter prefixes catch-all parameters with a single "/"; only that router-added
// slash is stripped, so slashes that are part of the client-supplied value are preserved.
func PathParam(r *http.Request, name string) string {
	v := httprouter.ParamsFromContext(r.Context()).ByName(name)
	return strings.TrimPrefix(v, "/")
}

// HeaderOrDefault returns the HTTP header value, or defaultValue if the header is
// missing or set to the empty string.
func HeaderOrDefault(r *http.Request, key string, defaultValue string) string {
	return StringValueOrDefault(r.Header.Get(key), defaultValue)
}

// QueryStringOrDefault returns query parameter value or defaultValue if missing or empty.
func QueryStringOrDefault(q url.Values, key string, defaultValue string) string {
	return StringValueOrDefault(q.Get(key), defaultValue)
}

// QueryIntOrDefault parses query parameter as signed integer or returns defaultValue if missing or invalid.
func QueryIntOrDefault(q url.Values, key string, defaultValue int) int {
	// strconv.Atoi parses into a platform int and reports a range error for
	// values that overflow it, so no explicit bounds check is needed.
	v, err := strconv.Atoi(q.Get(key))
	if err == nil {
		return v
	}

	return defaultValue
}

// QueryUintOrDefault parses query parameter as unsigned integer or returns defaultValue if missing or invalid.
func QueryUintOrDefault(q url.Values, key string, defaultValue uint) uint {
	// ParseUint with bitSize 0 parses into a platform uint and reports a range
	// error for values that overflow it, so the result always fits a uint and no
	// explicit bounds check is needed.
	v, err := strconv.ParseUint(q.Get(key), 10, 0)
	if err == nil {
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
	return cmp.Or(v, def)
}
