package httputil

import (
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// HeaderOrDefault returns the value of an HTTP header or a default value
func HeaderOrDefault(r *http.Request, key string, defaultValue string) string {
	v := r.Header.Get(key)
	if v == "" {
		return defaultValue
	}
	return v
}

// PathParam returns the value from the named path segment
func PathParam(r *http.Request, name string) string {
	v := httprouter.ParamsFromContext(r.Context()).ByName(name)
	return strings.TrimLeft(v, "/")
}
