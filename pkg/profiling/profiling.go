/*
Package profiling provides an HTTP handler that can be registered to a router to expose pprof profiling data.

For an example implementation, see the pkg/httpserver/config.go file.
*/
package profiling

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// PProfHandler exposes pprof profiling data.
// It is intended to be registered to a router.
// The option parameter is the pprof profile to be exposed.
// If option is empty, the pprof index page is shown.
// If option is "cmdline", "profile", "symbol" or "trace", the respective pprof handler is called.
// For any other value of option, the pprof.Handler is called with the option as argument.
func PProfHandler(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	profile := strings.TrimPrefix(ps.ByName("option"), "/")

	var handler http.HandlerFunc

	switch profile {
	case "":
		handler = pprof.Index
	case "cmdline":
		handler = pprof.Cmdline
	case "profile":
		handler = pprof.Profile
	case "symbol":
		handler = pprof.Symbol
	case "trace":
		handler = pprof.Trace
	default:
		handler = pprof.Handler(profile).ServeHTTP
	}

	handler(w, r)
}
