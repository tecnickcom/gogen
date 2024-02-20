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
	cfg.metricsHandlerFunc = func(_ http.ResponseWriter, _ *http.Request) {}
	cfg.pingHandlerFunc = func(_ http.ResponseWriter, _ *http.Request) {}
	cfg.pprofHandlerFunc = func(_ http.ResponseWriter, _ *http.Request) {}
	cfg.statusHandlerFunc = func(_ http.ResponseWriter, _ *http.Request) {}
	cfg.ipHandlerFunc = func(_ http.ResponseWriter, _ *http.Request) {}

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
