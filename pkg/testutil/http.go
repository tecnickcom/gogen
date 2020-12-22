//go:generate mockgen -package httputil -destination ../httputil/testutil_mock_test.go . TestHTTPResponseWriter
//go:generate mockgen -package jsendx -destination ../httputil/jsendx/testutil_mock_test.go . TestHTTPResponseWriter

package testutil

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// TestHTTPResponseWriter wraps a standard lib http.ResponseWriter to allow mock generation
type TestHTTPResponseWriter interface {
	http.ResponseWriter
}

// RouterWithHandler returns a new httprouter instance with the give handler function attached
func RouterWithHandler(method, path string, handlerFunc http.HandlerFunc) http.Handler {
	r := httprouter.New()
	r.HandlerFunc(method, path, handlerFunc)
	return r
}
