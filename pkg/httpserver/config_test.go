package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	require.NotNil(t, cfg)
	require.NotNil(t, cfg.metricsHandlerFunc)
	require.NotNil(t, cfg.pingHandlerFunc)
	require.NotNil(t, cfg.pprofHandlerFunc)
	require.NotNil(t, cfg.statusHandlerFunc)
	require.NotNil(t, cfg.ipHandlerFunc)
	require.NotEmpty(t, cfg.serverAddr)
	require.NotEqual(t, 0, cfg.shutdownTimeout)
	require.NotEmpty(t, cfg.traceIDHeaderName)
}

func Test_config_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupConfig func(c *config)
		wantErr     bool
	}{
		{
			name: "fail with invalid httpServer address",
			setupConfig: func(cfg *config) {
				cfg.serverAddr = "::"
			},
			wantErr: true,
		},
		{
			name: "fail with invalid shutdown timeout",
			setupConfig: func(cfg *config) {
				cfg.shutdownTimeout = 0
			},
			wantErr: true,
		},
		{
			name: "fail with missing router",
			setupConfig: func(cfg *config) {
				cfg.router = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing metrics handler",
			setupConfig: func(cfg *config) {
				cfg.metricsHandlerFunc = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing ping handler",
			setupConfig: func(cfg *config) {
				cfg.pingHandlerFunc = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing pprof handler",
			setupConfig: func(cfg *config) {
				cfg.pprofHandlerFunc = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing status handler",
			setupConfig: func(cfg *config) {
				cfg.statusHandlerFunc = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing ip handler",
			setupConfig: func(cfg *config) {
				cfg.ipHandlerFunc = nil
			},
			wantErr: true,
		},
		{
			name: "fail with empty trace id header name",
			setupConfig: func(cfg *config) {
				cfg.traceIDHeaderName = ""
			},
			wantErr: true,
		},
		{
			name: "succeed with valid configuration",
			setupConfig: func(cfg *config) {
				cfg.setRouter(testutil.Context())
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaultConfig()
			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			if err := cfg.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "invalid empty address",
			addr:    "",
			wantErr: true,
		},
		{
			name:    "bad address",
			addr:    "::",
			wantErr: true,
		},
		{
			name:    "invalid unspecified port",
			addr:    ":",
			wantErr: true,
		},
		{
			name:    "invalid address port",
			addr:    ":aaa",
			wantErr: true,
		},
		{
			name:    "address port out of range",
			addr:    ":67800",
			wantErr: true,
		},
		{
			name:    "valid address (no host)",
			addr:    ":8017",
			wantErr: false,
		},
		{
			name:    "valid address (localhost)",
			addr:    "localhost:8017",
			wantErr: false,
		},
		{
			name:    "valid address (ip)",
			addr:    "0.0.0.0:8017",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAddr(tt.addr)
			if tt.wantErr {
				require.NotNil(t, err, "validateAddr() addr = %q, error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			} else {
				require.NoError(t, err, "validateAddr() addr = %q, error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			}
		})
	}
}

func Test_config_isIndexRouteEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		defaultEnabledRoutes []DefaultRoute
		want                 bool
	}{
		{
			name:                 "should return true for enabled index route",
			defaultEnabledRoutes: []DefaultRoute{IndexRoute, MetricsRoute},
			want:                 true,
		},
		{
			name:                 "should return false for enabled index route",
			defaultEnabledRoutes: []DefaultRoute{MetricsRoute},
			want:                 false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &config{
				defaultEnabledRoutes: tt.defaultEnabledRoutes,
			}
			if got := c.isIndexRouteEnabled(); got != tt.want {
				t.Errorf("isIndexRouteEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setRouter(t *testing.T) {
	type testRouter interface {
		http.Handler

		// Handler is an http.Handler wrapper.
		Handler(method, path string, handler http.Handler)
	}

	t.Parallel()

	tests := []struct {
		name        string
		method      string
		path        string
		setupRouter func(testRouter)
		wantStatus  int
	}{
		{
			name:       "should handle 404",
			method:     http.MethodGet,
			path:       "/not/found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "should handle 405",
			method: http.MethodPost,
			setupRouter: func(r testRouter) {
				fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
				})
				r.Handler(http.MethodGet, "/not/allowed", fn)
			},
			path:       "/not/allowed",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "should handle panic in handler",
			method: http.MethodGet,
			setupRouter: func(r testRouter) {
				fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					panic("panicking!")
				})
				r.Handler(http.MethodGet, "/panic", fn)
			},
			path:       "/panic",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaultConfig()

			cfg.setRouter(testutil.Context())

			if tt.setupRouter != nil {
				tt.setupRouter(cfg.router)
			}

			rr := httptest.NewRecorder()
			cfg.router.ServeHTTP(rr, httptest.NewRequest(tt.method, tt.path, nil))

			resp := rr.Result() //nolint:bodyclose
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			require.Equal(t, tt.wantStatus, resp.StatusCode, "status code got = %d, want = %d", resp.StatusCode, tt.wantStatus)
		})
	}
}
