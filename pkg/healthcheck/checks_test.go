package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

//nolint:gocognit
func TestCheckHttpStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		handlerMethod     string
		handlerDelay      time.Duration
		handlerStatusCode int
		checkContext      context.Context
		checkMethod       string
		checkExtraPath    string
		checkTimeout      time.Duration
		checkOpts         []CheckOption
		checkWantStatus   int
		wantErr           bool
	}{
		{
			name:              "fails with invalid context",
			checkContext:      testutil.Context(),
			checkMethod:       http.MethodGet,
			checkExtraPath:    "/!@£$%^",
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           true,
		},
		{
			name:              "fails with wrong status code response",
			checkContext:      testutil.Context(),
			checkMethod:       http.MethodGet,
			checkTimeout:      1 * time.Second,
			checkWantStatus:   http.StatusOK,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusTeapot,
			wantErr:           true,
		},
		{
			name:              "fails with wrong check method",
			checkContext:      testutil.Context(),
			checkMethod:       http.MethodHead,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           true,
		},
		{
			name:              "fails with handler timeout",
			checkContext:      testutil.Context(),
			checkMethod:       http.MethodGet,
			checkTimeout:      1 * time.Second,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			handlerDelay:      2 * time.Second,
			wantErr:           true,
		},
		{
			name:              "succeed HEAD with 200 response",
			checkContext:      testutil.Context(),
			checkMethod:       http.MethodHead,
			checkTimeout:      1 * time.Second,
			checkWantStatus:   http.StatusOK,
			handlerMethod:     http.MethodHead,
			handlerStatusCode: http.StatusOK,
			wantErr:           false,
		},
		{
			name:              "succeed GET with 200 response",
			checkContext:      testutil.Context(),
			checkMethod:       http.MethodGet,
			checkTimeout:      1 * time.Second,
			checkWantStatus:   http.StatusOK,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           false,
		},
		{
			name:            "succeed GET with 200 response with opts",
			checkContext:    testutil.Context(),
			checkMethod:     http.MethodGet,
			checkTimeout:    1 * time.Second,
			checkWantStatus: http.StatusOK,
			checkOpts: []CheckOption{
				WithConfigureRequest(
					func(r *http.Request) {},
				),
			},
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if tt.handlerMethod != r.Method {
					httputil.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
					return
				}
				if tt.handlerMethod == r.Method {
					if tt.handlerDelay != 0 {
						time.Sleep(tt.handlerDelay)
					}
					httputil.SendStatus(r.Context(), w, tt.handlerStatusCode)
				}
			})

			ts := httptest.NewServer(mux)
			defer ts.Close()

			testHTTPClient := &http.Client{Timeout: 2 * time.Second}

			err := CheckHTTPStatus(tt.checkContext, testHTTPClient, tt.checkMethod, ts.URL+tt.checkExtraPath, tt.checkWantStatus, tt.checkTimeout, tt.checkOpts...)
			t.Logf("check error: %v", err)
			if tt.wantErr {
				require.Error(t, err, "CheckHTTPStatus() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.Nil(t, err, "CheckHTTPStatus() unexpected error = %v", err)
			}
		})
	}
}
