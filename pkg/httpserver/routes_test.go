package httpserver

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newDefaultRoutes(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	cfg.defaultEnabledRoutes = allDefaultRoutes()
	cfg.metricsHandlerFunc = func(w http.ResponseWriter, r *http.Request) {}
	cfg.pingHandlerFunc = func(w http.ResponseWriter, r *http.Request) {}
	cfg.pprofHandlerFunc = func(w http.ResponseWriter, r *http.Request) {}
	cfg.statusHandlerFunc = func(w http.ResponseWriter, r *http.Request) {}
	cfg.ipHandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

	cfg.disableDefaultRouteLogger[IndexRoute] = true
	cfg.disableDefaultRouteLogger[IPRoute] = true
	cfg.disableDefaultRouteLogger[MetricsRoute] = true
	cfg.disableDefaultRouteLogger[PingRoute] = true
	cfg.disableDefaultRouteLogger[PprofRoute] = true
	cfg.disableDefaultRouteLogger[StatusRoute] = true

	routes := newDefaultRoutes(cfg)
	expFuncs := []http.HandlerFunc{
		cfg.metricsHandlerFunc,
		cfg.pingHandlerFunc,
		cfg.pprofHandlerFunc,
		cfg.statusHandlerFunc,
		cfg.ipHandlerFunc,
	}

	boundCount := 0

	for _, expFn := range expFuncs {
		for _, r := range routes {
			if reflect.ValueOf(expFn).Pointer() == reflect.ValueOf(r.Handler).Pointer() {
				boundCount++

				require.True(t, r.DisableLogger, r.Path)
			}
		}
	}

	require.Equal(t, 5, boundCount)
}
