package jsendx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		path        string
		setupRouter func(*httprouter.Router)
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
			setupRouter: func(r *httprouter.Router) {
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
			setupRouter: func(r *httprouter.Router) {
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

			defaultInstrumentHandler := func(path string, handler http.HandlerFunc) http.Handler { return handler }

			params := &AppInfo{
				ProgramName:    "test",
				ProgramVersion: "1.2.3",
				ProgramRelease: "12345",
			}
			r := NewRouter(params, defaultInstrumentHandler)

			if tt.setupRouter != nil {
				tt.setupRouter(r)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, httptest.NewRequest(tt.method, tt.path, nil))

			resp := rr.Result()
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			require.Equal(t, tt.wantStatus, resp.StatusCode, "status code got = %d, want = %d", resp.StatusCode, tt.wantStatus)
		})
	}
}
