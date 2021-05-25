package httputil

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// HeaderOrDefault returns the value of an HTTP header or a default value.
func HeaderOrDefault(r *http.Request, key string, defaultValue string) string {
	v := r.Header.Get(key)
	if v == "" {
		return defaultValue
	}

	return v
}

// PathParam returns the value from the named path segment.
func PathParam(r *http.Request, name string) string {
	v := httprouter.ParamsFromContext(r.Context()).ByName(name)
	return strings.TrimLeft(v, "/")
}

// AddBasicAuth decorates the provided http.Request with Basic Authorization.
func AddBasicAuth(apiKey, apiSecret string, r *http.Request) {
	r.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(apiKey+":"+apiSecret)))
}

// QueryIntOrDefault set default value for a particular int key.
func QueryIntOrDefault(q url.Values, key string, def int) int {
	if v, err := strconv.ParseInt(q.Get(key), 10, 64); err == nil {
		return int(v)
	}

	return def
}

// QueryUintOrDefault set default value for a particular uint key.
func QueryUintOrDefault(q url.Values, key string, def uint) uint {
	if v, err := strconv.ParseUint(q.Get(key), 10, 64); err == nil {
		return uint(v)
	}

	return def
}
