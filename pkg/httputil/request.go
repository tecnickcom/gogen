package httputil

import (
	"net/http"
)

// HeaderOrDefault returns the value of an HTTP header or a default value
func HeaderOrDefault(r *http.Request, key string, defaultValue string) string {
	v := r.Header.Get(key)
	if v == "" {
		return defaultValue
	}
	return v
}
