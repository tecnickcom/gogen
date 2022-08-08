package jsendx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/julienschmidt/httprouter"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {
	t.Parallel()

	params := &AppInfo{
		ProgramName:    "test",
		ProgramVersion: "1.2.3",
		ProgramRelease: "12345",
	}

	rr := httptest.NewRecorder()
	Send(testutil.Context(), rr, http.StatusOK, params, "hello test")

	resp := rr.Result() //nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var okResp Response
	_ = json.Unmarshal(body, &okResp)

	require.Equal(t, "test", okResp.Program, "unexpected response: %s", body)
	require.Equal(t, "1.2.3", okResp.Version, "unexpected response: %s", body)
	require.Equal(t, "12345", okResp.Release, "unexpected response: %s", body)
	require.Equal(t, "OK", okResp.Message, "unexpected response: %s", body)
	require.Equal(t, "hello test", okResp.Data, "unexpected response: %s", body)

	// add coverage for error handling
	mockWriter := NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	Send(testutil.Context(), mockWriter, http.StatusOK, params, "message")
}

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

func TestDefaultIndexHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "1.2.3",
		ProgramRelease: "1",
	}

	routes := []route.Route{
		{
			Method:      http.MethodGet,
			Path:        "/get",
			Handler:     nil,
			Description: "Get endpoint",
		},
		{
			Method:      http.MethodPost,
			Path:        "/post",
			Handler:     nil,
			Description: "Post endpoint",
		},
	}
	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultIndexHandler(appInfo)(routes).ServeHTTP(rr, req)

	resp := rr.Result() //nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	bodyData, _ := io.ReadAll(resp.Body)
	body := string(bodyData)
	body = testutil.ReplaceDateTime(body, "<DT>")
	body = testutil.ReplaceUnixTimestamp(body, "<TS>")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"1.2.3\",\"release\":\"1\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":{\"routes\":[{\"method\":\"GET\",\"path\":\"/get\",\"description\":\"Get endpoint\"},{\"method\":\"POST\",\"path\":\"/post\",\"description\":\"Post endpoint\"}]}}\n", body)
}

func TestDefaultIPHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "2.3.4",
		ProgramRelease: "2",
	}

	tests := []struct {
		name    string
		ipFunc  httpserver.GetPublicIPFunc
		wantIP  string
		wantErr bool
	}{
		{
			name:    "success response",
			ipFunc:  func(ctx context.Context) (string, error) { return "0.0.0.0", nil },
			wantIP:  "0.0.0.0",
			wantErr: false,
		},
		{
			name:    "error response",
			ipFunc:  func(ctx context.Context) (string, error) { return "ERROR", fmt.Errorf("ERROR") },
			wantIP:  "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
			DefaultIPHandler(appInfo, tt.ipFunc).ServeHTTP(rr, req)

			resp := rr.Result() //nolint:bodyclose
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			bodyData, _ := io.ReadAll(resp.Body)
			body := string(bodyData)
			body = testutil.ReplaceDateTime(body, "<DT>")
			body = testutil.ReplaceUnixTimestamp(body, "<TS>")

			require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

			if tt.wantErr {
				require.Equal(t, http.StatusFailedDependency, resp.StatusCode)
				require.Equal(t, "{\"program\":\"Test\",\"version\":\"2.3.4\",\"release\":\"2\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"fail\",\"code\":424,\"message\":\"Failed Dependency\",\"data\":\"ERROR\"}\n", body)
			} else {
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, "{\"program\":\"Test\",\"version\":\"2.3.4\",\"release\":\"2\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"0.0.0.0\"}\n", body)
			}
		})
	}
}

//nolint:dupl
func TestDefaultPingHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "3.4.5",
		ProgramRelease: "3",
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultPingHandler(appInfo)(rr, req)

	resp := rr.Result() //nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	bodyData, _ := io.ReadAll(resp.Body)

	body := string(bodyData)
	body = testutil.ReplaceDateTime(body, "<DT>")
	body = testutil.ReplaceUnixTimestamp(body, "<TS>")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"3.4.5\",\"release\":\"3\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"OK\"}\n", body)
}

//nolint:dupl
func TestDefaultStatusHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "5.6.7",
		ProgramRelease: "4",
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultStatusHandler(appInfo)(rr, req)

	resp := rr.Result() //nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	bodyData, _ := io.ReadAll(resp.Body)

	body := string(bodyData)
	body = testutil.ReplaceDateTime(body, "<DT>")
	body = testutil.ReplaceUnixTimestamp(body, "<TS>")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"5.6.7\",\"release\":\"4\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"OK\"}\n", body)
}

func TestHealthCheckResultWriter(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "6.7.8",
		ProgramRelease: "5",
	}

	rr := httptest.NewRecorder()
	HealthCheckResultWriter(appInfo)(testutil.Context(), rr, http.StatusOK, "test body")

	resp := rr.Result() //nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	bodyData, _ := io.ReadAll(resp.Body)

	body := string(bodyData)
	body = testutil.ReplaceDateTime(body, "<DT>")
	body = testutil.ReplaceUnixTimestamp(body, "<TS>")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"6.7.8\",\"release\":\"5\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"test body\"}\n", body)
}
