package httpserver

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newDefaultRoutes(t *testing.T) {
	cfg := &config{
		defaultEnabledRoutes: allDefaultRoutes,
		metricsHandlerFunc:   func(w http.ResponseWriter, r *http.Request) {},
		pingHandlerFunc:      func(w http.ResponseWriter, r *http.Request) {},
		pprofHandlerFunc:     func(w http.ResponseWriter, r *http.Request) {},
		statusHandlerFunc:    func(w http.ResponseWriter, r *http.Request) {},
	}

	routes := newDefaultRoutes(cfg)
	expFuncs := []http.HandlerFunc{
		cfg.metricsHandlerFunc,
		cfg.pingHandlerFunc,
		cfg.pprofHandlerFunc,
		cfg.statusHandlerFunc,
	}

	boundCount := 0
	for _, expFn := range expFuncs {
		for _, r := range routes {
			if reflect.ValueOf(expFn).Pointer() == reflect.ValueOf(r.Handler).Pointer() {
				boundCount++
			}
		}
	}
	require.Equal(t, 4, boundCount)
}
