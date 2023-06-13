package httputil

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

func TestSendStatus(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	SendStatus(testutil.Context(), rr, http.StatusUnauthorized)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, http.StatusText(http.StatusUnauthorized)+"\n", string(body))
}

func TestSendText(t *testing.T) {
	t.Parallel()

	data := "text_data"

	rr := httptest.NewRecorder()
	SendText(testutil.Context(), rr, http.StatusOK, data)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, MimeTextPlain, resp.Header.Get("Content-Type"))
	require.Equal(t, data, string(body))

	// test error condition
	mockWriter := NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	SendText(testutil.Context(), mockWriter, http.StatusOK, data)
}

func TestSendJSON(t *testing.T) {
	t.Parallel()

	data := "json_data"

	rr := httptest.NewRecorder()
	SendJSON(testutil.Context(), rr, http.StatusOK, data)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, MimeApplicationJSON, resp.Header.Get("Content-Type"))
	require.Equal(t, "\""+data+"\"\n", string(body))

	// test error condition
	mockWriter := NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	SendJSON(testutil.Context(), mockWriter, http.StatusOK, data)
}

func TestSendXML(t *testing.T) {
	t.Parallel()

	data := "xml_data"

	rr := httptest.NewRecorder()
	SendXML(testutil.Context(), rr, http.StatusOK, XMLHeader, data)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, MimeApplicationXML, resp.Header.Get("Content-Type"))
	require.Equal(t, XMLHeader+"<string>"+data+"</string>", string(body))

	// test error condition
	mockWriter := NewMockTestHTTPResponseWriter(gomock.NewController(t))
	mockWriter.EXPECT().Header().AnyTimes().Return(http.Header{})
	mockWriter.EXPECT().WriteHeader(http.StatusOK)
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	mockWriter.EXPECT().Write(gomock.Any()).Return(0, fmt.Errorf("io error"))
	SendXML(testutil.Context(), mockWriter, http.StatusOK, XMLHeader, data)
}
