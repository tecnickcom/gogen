package httputil

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// nolint:tparallel
func TestStatus_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  Status
		want    []byte
		wantErr bool
	}{
		{
			name:   "success",
			status: Status(200),
			want:   []byte(`"success"`),
		},
		{
			name:   "error",
			status: Status(500),
			want:   []byte(`"error"`),
		},
		{
			name:   "fail",
			status: Status(400),
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
	SendJSON(testutil.Context(), rr, http.StatusOK, "hello")

	resp := rr.Result() // nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, `"hello"`+"\n", string(body))

	// add coverage for error handling
	mockWriter := NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	SendJSON(testutil.Context(), mockWriter, http.StatusOK, "message")
}

func TestSendText(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	SendText(testutil.Context(), rr, http.StatusOK, "hello")

	resp := rr.Result() // nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, `hello`, string(body))

	// add coverage for error handling
	mockWriter := NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	SendText(testutil.Context(), mockWriter, http.StatusOK, "message")
}

func TestSendStatus(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	SendStatus(testutil.Context(), rr, http.StatusUnauthorized)

	resp := rr.Result() // nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, http.StatusText(http.StatusUnauthorized)+"\n", string(body))
}
