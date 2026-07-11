package healthcheck

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/httputil"
)

//nolint:gocognit
func TestCheckHttpStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		handlerMethod     string
		handlerDelay      time.Duration
		handlerStatusCode int
		checkMethod       string
		checkExtraPath    string
		checkTimeout      time.Duration
		checkOpts         []CheckOption
		checkWantStatus   int
		wantErr           bool
	}{
		{
			name:              "fails with invalid context",
			checkMethod:       http.MethodGet,
			checkExtraPath:    "/!@£$%^",
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           true,
		},
		{
			name:              "fails with wrong status code response",
			checkMethod:       http.MethodGet,
			checkTimeout:      1 * time.Second,
			checkWantStatus:   http.StatusOK,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusTeapot,
			wantErr:           true,
		},
		{
			name:              "fails with wrong check method",
			checkMethod:       http.MethodHead,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           true,
		},
		{
			name:              "fails with handler timeout",
			checkMethod:       http.MethodGet,
			checkTimeout:      1 * time.Second,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			handlerDelay:      2 * time.Second,
			wantErr:           true,
		},
		{
			name:              "succeed HEAD with 200 response",
			checkMethod:       http.MethodHead,
			checkTimeout:      1 * time.Second,
			checkWantStatus:   http.StatusOK,
			handlerMethod:     http.MethodHead,
			handlerStatusCode: http.StatusOK,
			wantErr:           false,
		},
		{
			name:              "succeed GET with 200 response",
			checkMethod:       http.MethodGet,
			checkTimeout:      1 * time.Second,
			checkWantStatus:   http.StatusOK,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           false,
		},
		{
			name:            "succeed GET with 200 response with opts",
			checkMethod:     http.MethodGet,
			checkTimeout:    1 * time.Second,
			checkWantStatus: http.StatusOK,
			checkOpts: []CheckOption{
				WithConfigureRequest(
					func(_ *http.Request) {},
				),
			},
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           false,
		},
		{
			name:              "succeed with non-positive timeout (no added deadline)",
			checkMethod:       http.MethodGet,
			checkTimeout:      0,
			checkWantStatus:   http.StatusOK,
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusOK,
			wantErr:           false,
		},
		{
			name:            "succeed with accept-status predicate on a 2xx",
			checkMethod:     http.MethodGet,
			checkTimeout:    1 * time.Second,
			checkWantStatus: 0, // ignored when a predicate is set
			checkOpts: []CheckOption{
				WithAcceptStatus(func(code int) bool { return code >= 200 && code < 300 }),
			},
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusAccepted, // 202: a 2xx other than the default
			wantErr:           false,
		},
		{
			name:            "fails when accept-status predicate rejects",
			checkMethod:     http.MethodGet,
			checkTimeout:    1 * time.Second,
			checkWantStatus: 0,
			checkOpts: []CheckOption{
				WithAcceptStatus(func(code int) bool { return code == http.StatusOK }),
			},
			handlerMethod:     http.MethodGet,
			handlerStatusCode: http.StatusTeapot,
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hres := httputil.NewHTTPResp(slog.Default())
			mux := http.NewServeMux()

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if tt.handlerMethod != r.Method {
					hres.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
					return
				}

				if tt.handlerMethod == r.Method {
					if tt.handlerDelay != 0 {
						time.Sleep(tt.handlerDelay)
					}

					hres.SendStatus(r.Context(), w, tt.handlerStatusCode)
				}
			})

			ts := httptest.NewServer(mux)
			defer ts.Close()

			testHTTPClient := &http.Client{Timeout: 2 * time.Second}

			err := CheckHTTPStatus(t.Context(), testHTTPClient, tt.checkMethod, ts.URL+tt.checkExtraPath, tt.checkWantStatus, tt.checkTimeout, tt.checkOpts...)

			t.Logf("check error: %v", err)

			if tt.wantErr {
				require.Error(t, err, "CheckHTTPStatus() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.NoError(t, err, "CheckHTTPStatus() unexpected error = %v", err)
			}
		})
	}
}

// nilResponseClient is a misbehaving client that returns no response and no
// error, violating the http.Client contract, to exercise the nil-response guard.
type nilResponseClient struct{}

func (nilResponseClient) Do(_ *http.Request) (*http.Response, error) {
	return nil, nil //nolint:nilnil
}

func TestCheckHTTPStatus_NilResponse(t *testing.T) {
	t.Parallel()

	err := CheckHTTPStatus(t.Context(), nilResponseClient{}, http.MethodGet, "http://example.invalid", http.StatusOK, time.Second)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil response")
}

// nilBodyClient returns a response whose Body is nil, which the real http.Client
// never does but a custom HTTPClient can, to exercise the nil-body guard.
type nilBodyClient struct{}

func (nilBodyClient) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestCheckHTTPStatus_NilBody(t *testing.T) {
	t.Parallel()

	err := CheckHTTPStatus(t.Context(), nilBodyClient{}, http.MethodGet, "http://example.invalid", http.StatusOK, time.Second)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil response or body")
}
