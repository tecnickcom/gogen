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

type mockResponseWriter struct {
	*bytes.Buffer
	hijackCalled bool
	pushCalled   bool
}

func newMockResponseWriter() *mockResponseWriter {
	buf := bytes.NewBuffer([]byte{})
	return &mockResponseWriter{Buffer: buf}
}

func (rw *mockResponseWriter) Header() http.Header {
	return nil
}

//nolint:wrapcheck
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

type mockBrokenResponseWriter struct {
}

func newMockBrokenResponseWriter() *mockBrokenResponseWriter {
	return &mockBrokenResponseWriter{}
}

func (rw *mockBrokenResponseWriter) Header() http.Header {
	return nil
}

func (rw *mockBrokenResponseWriter) Write(in []byte) (int, error) {
	return 0, nil
}

func (rw *mockBrokenResponseWriter) WriteHeader(statusCode int) {

}

func TestNewWrapResponseWriter(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := NewResponseWriterWrapper(rr)
	require.NotNil(t, ww)
	wwResponseWriterWrapper, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)
	require.Equal(t, reflect.ValueOf(rr).Pointer(), reflect.ValueOf(wwResponseWriterWrapper.ResponseWriter).Pointer())
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
	ww.WriteHeader(207)
	require.Equal(t, 207, ww.Status())
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
	ww.WriteHeader(204)
	require.Equal(t, 204, ww.Status())
	ww.WriteHeader(301)
	require.Equal(t, 204, ww.Status())
}

func Test_responseWriterWrapper_Hijack(t *testing.T) {
	t.Parallel()

	mock := newMockResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	wwResponseWriterWrapper, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	_, _, err := wwResponseWriterWrapper.Hijack()
	require.NoError(t, err)
	require.True(t, mock.hijackCalled)
}

func Test_broken_responseWriterWrapper_Hijack(t *testing.T) {
	t.Parallel()

	mock := newMockBrokenResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	wwResponseWriterWrapper, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	_, _, err := wwResponseWriterWrapper.Hijack()
	require.Error(t, err)
}

func Test_responseWriterWrapper_Push(t *testing.T) {
	t.Parallel()

	mock := newMockResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	wwResponseWriterWrapper, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	_ = wwResponseWriterWrapper.Push("", &http.PushOptions{})

	require.True(t, mock.pushCalled)
}

func Test_broken_responseWriterWrapper_Push(t *testing.T) {
	t.Parallel()

	mock := newMockBrokenResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	wwResponseWriterWrapper, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	err := wwResponseWriterWrapper.Push("", &http.PushOptions{})
	require.Error(t, err)
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

	wwTeeResponseWriterWrapper, ok := wwTee.(*responseWriterWrapper)
	require.True(t, ok)
	require.True(t, wwTeeResponseWriterWrapper.headerWritten)
}

func Test_broken_responseWriterWrapper_ReadFrom(t *testing.T) {
	t.Parallel()

	mock := newMockBrokenResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	inputBuf := bytes.NewBufferString("-")
	_, err := ww.(*responseWriterWrapper).ReadFrom(inputBuf)
	require.Error(t, err)
}
