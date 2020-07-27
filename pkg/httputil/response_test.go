// +build unit

package httputil_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/internal/mocks"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		status  httputil.Status
		want    []byte
		wantErr bool
	}{
		{
			name:   "success",
			status: httputil.Status(200),
			want:   []byte(`"success"`),
		},
		{
			name:   "error",
			status: httputil.Status(500),
			want:   []byte(`"error"`),
		},
		{
			name:   "fail",
			status: httputil.Status(400),
			want:   []byte(`"fail"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.status.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestSendJSON(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	httputil.SendJSON(testutil.Context(), rr, http.StatusOK, "hello")

	resp := rr.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, `"hello"`+"\n", string(body))

	// add coverage for error handling
	mockWriter := mocks.NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	httputil.SendJSON(testutil.Context(), mockWriter, http.StatusOK, "message")
}

func TestSendText(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	httputil.SendText(testutil.Context(), rr, http.StatusOK, "hello")

	resp := rr.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, `hello`, string(body))

	// add coverage for error handling
	mockWriter := mocks.NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	httputil.SendText(testutil.Context(), mockWriter, http.StatusOK, "message")
}

func TestSendStatus(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	httputil.SendStatus(testutil.Context(), rr, http.StatusUnauthorized)

	resp := rr.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, http.StatusText(http.StatusUnauthorized)+"\n", string(body))
}

// FIXME: broken test
// func Test_requestLogHandler(t *testing.T) {
// 	t.Parallel()
//
// 	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		l := logging.FromContext(r.Context())
// 		require.NotNil(t, l, "logger not found")
//
// 		http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
// 	})
//
// 	ctx, logs := testutil.ContextWithLogObserver(zapcore.DebugLevel)
// 	logging.FromContext(ctx)
//
// 	handler := requestLogHandler(nextHandler)
//
// 	rr := httptest.NewRecorder()
//
// 	req := httptest.NewRequest(http.MethodGet, "/", nil)
// 	reqCtx := logging.WithLogger(req.Context(), logging.FromContext(ctx))
// 	req = req.WithContext(reqCtx)
//
// 	handler.ServeHTTP(rr, req)
//
// 	resp := rr.Result()
// 	require.Equal(t, http.StatusOK, resp.StatusCode)
//
// 	logEntries := logs.All()
// 	require.Len(t, logEntries, 2, "expected 2 log messages")
//
// 	for i, e := range logEntries {
// 		switch i {
// 		case 0:
// 			require.Equal(t, "[start] HTTP request", e.Message, "invalid start message: %q", e.Message)
// 		case 1:
// 			logCtx := e.ContextMap()
//
// 			// check status
// 			statusValue, statusExists := logCtx["status"]
// 			require.True(t, statusExists, "status field missing")
// 			require.EqualValues(t, http.StatusOK, statusValue, "expected status value not found")
//
// 			// check latency
// 			latencyValue, latencyExists := logCtx["latency"]
// 			require.True(t, latencyExists, "latency field missing")
// 			require.NotEmpty(t, latencyValue, "latency value empty")
//
// 			require.Equal(t, "[end] HTTP request", e.Message, "invalid end message: %q", e.Message)
// 		}
// 	}
// }
