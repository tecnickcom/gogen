// Package profiling is used to expose runtime profiling data (pprof).
package profiling

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// PProfHandler exposes pprof data
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
