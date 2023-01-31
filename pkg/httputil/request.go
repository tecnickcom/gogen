package httputil

import (
	"context"
	"encoding/base64"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

type timeCtxKey string

// ReqTimeCtxKey is the Context key to retrieve the request time.
const ReqTimeCtxKey = timeCtxKey("request_time")

// AddBasicAuth decorates the provided http.Request with Basic Authorization.
func AddBasicAuth(apiKey, apiSecret string, r *http.Request) {
	r.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(apiKey+":"+apiSecret)))
}

// PathParam returns the value from the named path segment.
func PathParam(r *http.Request, name string) string {
	v := httprouter.ParamsFromContext(r.Context()).ByName(name)
	return strings.TrimLeft(v, "/")
}

// HeaderOrDefault returns the value of an HTTP header or a default value.
func HeaderOrDefault(r *http.Request, key string, defaultValue string) string {
	v := r.Header.Get(key)
	if v != "" {
		return v
	}

	return defaultValue
}

// QueryStringOrDefault returns the string value of the specified URL query parameter or a default value.
func QueryStringOrDefault(q url.Values, key string, defaultValue string) string {
	v := q.Get(key)
	if v != "" {
		return v
	}

	return defaultValue
}

// QueryIntOrDefault returns the integer value of the specified URL query parameter or a default value.
func QueryIntOrDefault(q url.Values, key string, defaultValue int) int {
	if v, err := strconv.ParseInt(q.Get(key), 10, 64); err == nil && v >= math.MinInt && v <= math.MaxInt {
		return int(v)
	}

	return defaultValue
}

// QueryUintOrDefault returns the unsigned integer value of the specified URL query parameter or a default value.
func QueryUintOrDefault(q url.Values, key string, defaultValue uint) uint {
	if v, err := strconv.ParseUint(q.Get(key), 10, 64); err == nil && v <= math.MaxUint {
		return uint(v)
	}

	return defaultValue
}

// WithRequestTime returns a new context with the added request time.
func WithRequestTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, ReqTimeCtxKey, t)
}

// GetRequestTimeFromContext returns the request time from the context.
func GetRequestTimeFromContext(ctx context.Context) (time.Time, bool) {
	v := ctx.Value(ReqTimeCtxKey)
	t, ok := v.(time.Time)

	return t, ok
}

// GetRequestTime returns the request time from the http request.
func GetRequestTime(r *http.Request) (time.Time, bool) {
	return GetRequestTimeFromContext(r.Context())
}
