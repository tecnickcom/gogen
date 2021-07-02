package httputil

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func newMockResponseWriter() *mockResponseWriter {
	buf := bytes.NewBuffer([]byte{})
	return &mockResponseWriter{Buffer: buf}
}

type mockResponseWriter struct {
	*bytes.Buffer
	hijackCalled bool
	pushCalled   bool
}

func (rw *mockResponseWriter) Header() http.Header {
	return nil
}

// nolint:wrapcheck
func (rw *mockResponseWriter) Write(in []byte) (int, error) {
	return rw.Buffer.Write(in)
}

func (rw *mockResponseWriter) WriteHeader(statusCode int) {

}

func (rw *mockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw.hijackCalled = true
	return nil, nil, nil
}

func (rw *mockResponseWriter) Push(target string, opts *http.PushOptions) error {
	rw.pushCalled = true
	return nil
}

func TestNewWrapResponseWriter(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := NewResponseWriterWrapper(rr)
	require.NotNil(t, ww)
	require.Equal(t, reflect.ValueOf(rr).Pointer(), reflect.ValueOf(ww.(*responseWriterWrapper).ResponseWriter).Pointer())
}

func Test_responseWriterWrapper_Size(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := responseWriterWrapper{ResponseWriter: rr}
	count, err := ww.Write([]byte("test-counter"))
	require.Equal(t, 12, count)
	require.NoError(t, err)
	require.Equal(t, 12, ww.Size())
}

func Test_responseWriterWrapper_Flush(t *testing.T) {
	t.Parallel()

	ww := responseWriterWrapper{ResponseWriter: httptest.NewRecorder()}
	ww.Flush()
	require.True(t, ww.headerWritten, "expected flush to set headerWritten=true")
}

func Test_responseWriterWrapper_Status(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := responseWriterWrapper{ResponseWriter: rr}
	ww.WriteHeader(97)
	require.Equal(t, 97, ww.Status())
}

func Test_responseWriterWrapper_Tee(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := responseWriterWrapper{ResponseWriter: rr}

	buf := bytes.NewBuffer([]byte{})
	ww.Tee(buf)

	count, err := ww.Write([]byte("tee"))
	require.Equal(t, 3, count)
	require.NoError(t, err)
	require.Equal(t, 3, ww.Size())
	require.Equal(t, 3, buf.Len())
	require.Equal(t, "tee", buf.String())
}

func Test_responseWriterWrapper_Write(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := responseWriterWrapper{ResponseWriter: rr}
	_, err := ww.Write([]byte("written"))
	require.NoError(t, err)
	require.Equal(t, 7, ww.Size())
}

func Test_responseWriterWrapper_WriteHeader(t *testing.T) {
	t.Parallel()

	ww := responseWriterWrapper{ResponseWriter: httptest.NewRecorder()}
	ww.WriteHeader(19)
	require.Equal(t, 19, ww.Status())
	ww.WriteHeader(41)
	require.Equal(t, 19, ww.Status())
}

func Test_responseWriterWrapper_Hijack(t *testing.T) {
	t.Parallel()

	mock := newMockResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	_, _, err := ww.(*responseWriterWrapper).Hijack()
	require.NoError(t, err)
	require.True(t, mock.hijackCalled)
}

func Test_responseWriterWrapper_Push(t *testing.T) {
	t.Parallel()

	mock := newMockResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	_ = ww.(*responseWriterWrapper).Push("", &http.PushOptions{})

	require.True(t, mock.pushCalled)
}

func Test_responseWriterWrapper_ReadFrom(t *testing.T) {
	t.Parallel()

	// without tee
	mock := newMockResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	inputBuf := bytes.NewBufferString("0123456789")
	count, err := ww.(*responseWriterWrapper).ReadFrom(inputBuf)
	require.NoError(t, err)
	require.Equal(t, int64(10), count)

	// with tee writer
	mockTee := newMockResponseWriter()
	wwTee := NewResponseWriterWrapper(mockTee)
	require.NotNil(t, wwTee)

	teeBuf := bytes.NewBuffer([]byte{})
	wwTee.Tee(teeBuf)

	inputBufTee := bytes.NewBufferString("0123456789")
	countTee, err := wwTee.(*responseWriterWrapper).ReadFrom(inputBufTee)
	require.NoError(t, err)
	require.Equal(t, int64(10), countTee)
	require.Equal(t, "0123456789", teeBuf.String())
	require.True(t, wwTee.(*responseWriterWrapper).headerWritten)
}
