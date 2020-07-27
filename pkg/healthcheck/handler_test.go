// +build unit

package healthcheck

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type testHealthChecker struct {
	delay  time.Duration
	result Result
}

func (th *testHealthChecker) HealthCheck(ctx context.Context) Result {
	if th.delay != 0 {
		time.Sleep(th.delay)
	}
	return th.result
}

func TestHandler(t *testing.T) {
	tests := []struct {
		name           string
		checkers       []HealthChecker
		wantBody       string
		wantMaxElapsed time.Duration
	}{
		{
			name: "success multiple OK",
			checkers: []HealthChecker{
				&testHealthChecker{delay: 100 * time.Millisecond, result: Result{Status: OK}},
				&testHealthChecker{delay: 100 * time.Millisecond, result: Result{Status: OK}},
			},
			wantBody:       `{"0":{"status":"OK"},"1":{"status":"OK"}}`,
			wantMaxElapsed: 200 * time.Millisecond,
		},
		{
			name: "success mixed results",
			checkers: []HealthChecker{
				&testHealthChecker{delay: 100 * time.Millisecond, result: Result{Status: OK}},
				&testHealthChecker{
					delay:  200 * time.Millisecond,
					result: Result{Status: Err, Error: fmt.Errorf("check error")},
				},
			},
			wantBody:       `{"0":{"status":"OK"},"1":{"status":"ERR","error":{}}}`,
			wantMaxElapsed: 300 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checkers := make(map[string]HealthChecker)
			for i, v := range tt.checkers {
				checkers[fmt.Sprintf("%d", i)] = v
			}

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
			require.NoError(t, err, "no error expected reading body data")

			handler := Handler(checkers)

			st := time.Now()
			handler(rr, req)
			el := time.Since(st)

			resp := rr.Result()
			payload, _ := ioutil.ReadAll(resp.Body)

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
			require.Equal(t, tt.wantBody+"\n", string(payload))

			// ensure we are running concurrently
			t.Logf("check time = %s, want = <%s", el, tt.wantMaxElapsed)
			require.True(t, el < tt.wantMaxElapsed, "the check took longer than %v", tt.wantMaxElapsed)
		})
	}
}

func Test_runCheckWithTimeout(t *testing.T) {
	tests := []struct {
		name       string
		checker    HealthChecker
		timeout    time.Duration
		wantResult Result
	}{
		{
			name:       "check fails with timeout",
			checker:    &testHealthChecker{delay: 100 * time.Millisecond},
			timeout:    10 * time.Millisecond,
			wantResult: Result{Status: Err, Error: context.DeadlineExceeded},
		},
		{
			name: "check succeed with OK result",
			checker: &testHealthChecker{delay: 100 * time.Millisecond, result: Result{
				Status: OK,
			}},
			timeout:    500 * time.Millisecond,
			wantResult: Result{Status: OK},
		},
		{
			name: "check succeed with ERR result",
			checker: &testHealthChecker{delay: 100 * time.Millisecond, result: Result{
				Status: Err,
				Error:  fmt.Errorf("check failed"),
			}},
			timeout:    500 * time.Millisecond,
			wantResult: Result{Status: Err, Error: fmt.Errorf("check failed")},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context()
			got := runCheckWithTimeout(ctx, tt.checker, tt.timeout)
			if !reflect.DeepEqual(got, tt.wantResult) {
				t.Errorf("runCheck() = %#v, want %#v", got, tt.wantResult)
			}
		})
	}
}
