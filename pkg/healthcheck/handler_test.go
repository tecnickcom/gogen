package healthcheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vonage/gosrvlib/pkg/httputil"
	"github.com/vonage/gosrvlib/pkg/testutil"
)

func TestNewHandler(t *testing.T) {
	t.Parallel()

	testChecks := []HealthCheck{
		New("testcheck_1", &testHealthChecker{}),
		New("testcheck_2", &testHealthChecker{}),
	}

	// No options
	h1 := NewHandler(testChecks)
	require.Equal(t, 2, len(h1.checks))
	require.Equal(t, 2, h1.checksCount)
	require.Equal(t, reflect.ValueOf(httputil.SendJSON).Pointer(), reflect.ValueOf(h1.writeResult).Pointer())

	// With options
	rw := func(ctx context.Context, w http.ResponseWriter, statusCode int, data interface{}) {}
	h2 := NewHandler(testChecks, WithResultWriter(rw))
	require.Equal(t, 2, len(h2.checks))
	require.Equal(t, 2, h2.checksCount)
	require.Equal(t, reflect.ValueOf(rw).Pointer(), reflect.ValueOf(h2.writeResult).Pointer())
}

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		checks         []HealthCheck
		opts           []HandlerOption
		wantStatus     int
		wantBody       string
		wantMaxElapsed time.Duration
	}{
		{
			name: "success multiple OK",
			checks: []HealthCheck{
				New("test_01", &testHealthChecker{delay: 100 * time.Millisecond, err: nil}),
				New("test_02", &testHealthChecker{delay: 100 * time.Millisecond, err: nil}),
			},
			wantStatus:     http.StatusOK,
			wantBody:       `{"test_01":"OK","test_02":"OK"}`,
			wantMaxElapsed: 200 * time.Millisecond,
		},
		{
			name: "success multiple OK with custom response writer",
			checks: []HealthCheck{
				New("test_11", &testHealthChecker{delay: 100 * time.Millisecond, err: nil}),
				New("test_12", &testHealthChecker{delay: 100 * time.Millisecond, err: nil}),
			},
			opts: []HandlerOption{
				WithResultWriter(func(ctx context.Context, w http.ResponseWriter, statusCode int, data interface{}) {
					type wrapper struct {
						Data interface{} `json:"data"`
					}
					httputil.SendJSON(ctx, w, statusCode, &wrapper{
						Data: data,
					})
				}),
			},
			wantStatus:     http.StatusOK,
			wantBody:       `{"data":{"test_11":"OK","test_12":"OK"}}`,
			wantMaxElapsed: 200 * time.Millisecond,
		},
		{
			name: "success mixed results",
			checks: []HealthCheck{
				New("test_31", &testHealthChecker{delay: 100 * time.Millisecond, err: nil}),
				New("test_32", &testHealthChecker{delay: 200 * time.Millisecond, err: fmt.Errorf("check error")}),
			},
			wantStatus:     http.StatusServiceUnavailable,
			wantBody:       `{"test_31":"OK","test_32":"check error"}`,
			wantMaxElapsed: 300 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
			require.NoError(t, err, "no error expected reading body data")

			h := NewHandler(tt.checks, tt.opts...)

			st := time.Now()
			h.ServeHTTP(rr, req)
			el := time.Since(st)

			resp := rr.Result() //nolint:bodyclose
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			payloadData, _ := io.ReadAll(resp.Body)
			payload := string(payloadData)

			require.Equal(t, tt.wantStatus, resp.StatusCode)
			require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
			require.Equal(t, tt.wantBody+"\n", payload)

			// ensure we are running concurrently
			require.True(t, el < tt.wantMaxElapsed, "check time = %s, want < %s", el, tt.wantMaxElapsed)
		})
	}
}
