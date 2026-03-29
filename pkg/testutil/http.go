package testutil

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// TestHTTPResponseWriter wraps [http.ResponseWriter] to simplify mock generation.
type TestHTTPResponseWriter interface {
	http.ResponseWriter
}

// RouterWithHandler returns a new httprouter with handlerFunc registered for method and path.
func RouterWithHandler(method, path string, handlerFunc http.HandlerFunc) http.Handler {
	r := httprouter.New()
	r.HandlerFunc(method, path, handlerFunc)

	return r
}
