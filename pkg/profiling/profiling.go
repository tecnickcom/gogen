/*
Package profiling bridges Go's built-in [net/http/pprof] profiling tool and the
[httprouter] request router, allowing all pprof endpoints to be served through a
single wildcard route without manual per-handler registration.

# Problem

Go's standard [net/http/pprof] package registers its handlers on
[http.DefaultServeMux]. Applications that use a custom router (such as
httprouter) cannot use DefaultServeMux directly, and registering each pprof
endpoint individually is tedious and error-prone.

# Solution

[PProfHandler] is a single [http.HandlerFunc] that reads an `*option` wildcard
parameter from the httprouter request context and dispatches to the correct
pprof handler automatically. Registering one wildcard route is all that is
needed:

	router.GET("/pprof/*option", profiling.PProfHandler)

# Supported Endpoints

The following pprof paths are handled after the wildcard prefix:

	/pprof/             — interactive index page listing all available profiles
	/pprof/cmdline      — running program's command line
	/pprof/profile      — 30-second (or ?seconds=N) CPU profile
	/pprof/symbol       — symbol lookup for program counters
	/pprof/trace        — execution trace (use ?seconds=N to set duration)
	/pprof/<name>       — any named runtime profile, e.g. heap, goroutine,
	                      allocs, block, mutex, threadcreate

# Features

  - Zero-configuration: a single handler covers every pprof endpoint.
  - Router-agnostic path extraction: uses httprouter's context parameter so the
    mount prefix can be anything (e.g. /debug/pprof/*option).
  - No global state: does not touch [http.DefaultServeMux].
  - Extensible: named profiles (heap, goroutine, …) are forwarded to
    [pprof.Handler] automatically; no code changes are required when the Go
    runtime adds new profiles.

# Caveats

The route's wildcard parameter must be named exactly [WildcardParamName]
("option"); [PProfHandler] reads that parameter to select the endpoint. A route
registered with a different wildcard name (e.g. /pprof/*path) would make the
handler serve the index page for every request.

The /pprof/profile and /pprof/trace endpoints block for the requested duration
(?seconds=N). Mount this handler on a route that is exempt from any per-request
timeout, otherwise long profiles are truncated. The
[github.com/tecnickcom/gogen/pkg/httpserver] integration disables the request
timeout for the pprof route for this reason.

The index page served at the empty path links to the other profiles using
relative URLs (e.g. heap?debug=1), so it must be reached at a path ending in a
slash (/pprof/). httprouter's default RedirectTrailingSlash handling redirects
/pprof to /pprof/, which keeps those links working; keep that redirect enabled.

# Security Note

pprof endpoints expose detailed internals of a running process (memory layout,
goroutine stacks, CPU traces). Mount this handler only on an internal or
administrative server that is not reachable from the public internet, and
protect it with authentication middleware appropriate for your environment.

# Integration

The [github.com/tecnickcom/gogen/pkg/httpserver] package registers
[PProfHandler] as the default pprof handler on its internal router. See
pkg/httpserver/config.go for a complete integration example.
*/
package profiling

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// WildcardParamName is the httprouter wildcard parameter name that
// [PProfHandler] reads to determine which pprof endpoint to serve. Routes must
// be registered using exactly this name:
//
//	router.GET("/pprof/*"+profiling.WildcardParamName, profiling.PProfHandler)
const WildcardParamName = "option"

// PProfHandler is an [http.HandlerFunc] that exposes all pprof profiling
// endpoints through a single httprouter wildcard route.
//
// Register it with an httprouter-compatible router using a wildcard route
// whose parameter is named [WildcardParamName]:
//
//	router.GET("/pprof/*option", profiling.PProfHandler)
//
// The `*option` wildcard determines which pprof handler is invoked:
//   - ""          → [pprof.Index]  (the interactive profile listing page)
//   - "cmdline"   → [pprof.Cmdline]
//   - "profile"   → [pprof.Profile] (accepts ?seconds=N, default 30)
//   - "symbol"    → [pprof.Symbol]
//   - "trace"     → [pprof.Trace]   (accepts ?seconds=N, default 1)
//   - any other   → [pprof.Handler](option), covering heap, goroutine,
//     allocs, block, mutex, threadcreate, and future runtime profiles
func PProfHandler(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	// Trim slashes on both ends so the switch matches regardless of a leading
	// slash (always present in httprouter catch-all values) or a stray
	// trailing slash (e.g. "/profile/").
	profile := strings.Trim(ps.ByName(WildcardParamName), "/")

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
