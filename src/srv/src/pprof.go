package main

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// pprofHandler exposes pprof data
func pprofHandler(rw http.ResponseWriter, hr *http.Request) {
	ps := httprouter.ParamsFromContext(hr.Context())
	profile := strings.TrimPrefix(ps.ByName("option"), "/")
	hf := pprof.Index
	switch profile {
	case "":
	case "cmdline":
		hf = pprof.Cmdline
	case "profile":
		hf = pprof.Profile
	case "symbol":
		hf = pprof.Symbol
	case "trace":
		hf = pprof.Trace
	default:
		hf = pprof.Handler(profile).ServeHTTP
	}
	hf(rw, hr)
}
