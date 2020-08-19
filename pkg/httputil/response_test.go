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
