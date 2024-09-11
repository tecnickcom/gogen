package jsendx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/httpserver"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

	resp := rr.Result()
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
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, errors.New("io error"))
	Send(testutil.Context(), mockWriter, http.StatusOK, params, "message")
}

func TestDefaultNotFoundHandlerFunc(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "1.1.1",
		ProgramRelease: "1",
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultNotFoundHandlerFunc(appInfo)(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	bodyData, _ := io.ReadAll(resp.Body)

	body := string(bodyData)
	body = testutil.ReplaceDateTime(body, "<DT>")
	body = testutil.ReplaceUnixTimestamp(body, "<TS>")

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"1.1.1\",\"release\":\"1\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"fail\",\"code\":404,\"message\":\"Not Found\",\"data\":\"invalid endpoint\"}\n", body)
}

func TestDefaultMethodNotAllowedHandlerFunc(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "2.2.2",
		ProgramRelease: "2",
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultMethodNotAllowedHandlerFunc(appInfo)(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	bodyData, _ := io.ReadAll(resp.Body)

	body := string(bodyData)
	body = testutil.ReplaceDateTime(body, "<DT>")
	body = testutil.ReplaceUnixTimestamp(body, "<TS>")

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"2.2.2\",\"release\":\"2\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"fail\",\"code\":405,\"message\":\"Method Not Allowed\",\"data\":\"the request cannot be routed\"}\n", body)
}

func TestDefaultPanicHandlerFunc(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "3.3.3",
		ProgramRelease: "3",
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultPanicHandlerFunc(appInfo)(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	bodyData, _ := io.ReadAll(resp.Body)

	body := string(bodyData)
	body = testutil.ReplaceDateTime(body, "<DT>")
	body = testutil.ReplaceUnixTimestamp(body, "<TS>")

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"3.3.3\",\"release\":\"3\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"error\",\"code\":500,\"message\":\"Internal Server Error\",\"data\":\"internal error\"}\n", body)
}

func TestDefaultIndexHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "4.4.4",
		ProgramRelease: "4",
	}

	routes := []httpserver.Route{
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

	resp := rr.Result()
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
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"4.4.4\",\"release\":\"4\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":{\"routes\":[{\"method\":\"GET\",\"path\":\"/get\",\"description\":\"Get endpoint\"},{\"method\":\"POST\",\"path\":\"/post\",\"description\":\"Post endpoint\"}]}}\n", body)
}

func TestDefaultIPHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "5.5.5",
		ProgramRelease: "5",
	}

	tests := []struct {
		name    string
		ipFunc  httpserver.GetPublicIPFunc
		wantIP  string
		wantErr bool
	}{
		{
			name:    "success response",
			ipFunc:  func(_ context.Context) (string, error) { return "0.0.0.0", nil },
			wantIP:  "0.0.0.0",
			wantErr: false,
		},
		{
			name:    "error response",
			ipFunc:  func(_ context.Context) (string, error) { return "ERROR", errors.New("ERROR") },
			wantIP:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
			DefaultIPHandler(appInfo, tt.ipFunc).ServeHTTP(rr, req)

			resp := rr.Result()
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
				require.Equal(t, "{\"program\":\"Test\",\"version\":\"5.5.5\",\"release\":\"5\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"fail\",\"code\":424,\"message\":\"Failed Dependency\",\"data\":\"ERROR\"}\n", body)
			} else {
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, "{\"program\":\"Test\",\"version\":\"5.5.5\",\"release\":\"5\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"0.0.0.0\"}\n", body)
			}
		})
	}
}

func TestDefaultPingHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "6.6.6",
		ProgramRelease: "6",
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultPingHandler(appInfo)(rr, req)

	resp := rr.Result()
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
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"6.6.6\",\"release\":\"6\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"OK\"}\n", body)
}

func TestDefaultStatusHandler(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "7.7.7",
		ProgramRelease: "7",
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	DefaultStatusHandler(appInfo)(rr, req)

	resp := rr.Result()
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
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"7.7.7\",\"release\":\"7\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"OK\"}\n", body)
}

func TestHealthCheckResultWriter(t *testing.T) {
	t.Parallel()

	appInfo := &AppInfo{
		ProgramName:    "Test",
		ProgramVersion: "8.8.8",
		ProgramRelease: "8",
	}

	rr := httptest.NewRecorder()
	HealthCheckResultWriter(appInfo)(testutil.Context(), rr, http.StatusOK, "test body")

	resp := rr.Result()
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
	require.Equal(t, "{\"program\":\"Test\",\"version\":\"8.8.8\",\"release\":\"8\",\"datetime\":\"<DT>\",\"timestamp\":<TS>,\"status\":\"success\",\"code\":200,\"message\":\"OK\",\"data\":\"test body\"}\n", body)
}
