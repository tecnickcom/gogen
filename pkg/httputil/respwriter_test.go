package httputil

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

func (rw *mockResponseWriter) WriteHeader(_ int) {
}

func (rw *mockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw.hijackCalled = true
	return nil, nil, nil
}

func (rw *mockResponseWriter) Push(_ string, _ *http.PushOptions) error {
	rw.pushCalled = true
	return nil
}

// mockPlainResponseWriter is a working http.ResponseWriter that does NOT
// implement io.ReaderFrom, exercising the generic-copy fallback in ReadFrom.
type mockPlainResponseWriter struct {
	buf bytes.Buffer
}

func (rw *mockPlainResponseWriter) Header() http.Header {
	return nil
}

//nolint:wrapcheck
func (rw *mockPlainResponseWriter) Write(in []byte) (int, error) {
	return rw.buf.Write(in)
}

func (rw *mockPlainResponseWriter) WriteHeader(_ int) {
}

type mockBrokenResponseWriter struct{}

func newMockBrokenResponseWriter() *mockBrokenResponseWriter {
	return &mockBrokenResponseWriter{}
}

func (rw *mockBrokenResponseWriter) Header() http.Header {
	return nil
}

func (rw *mockBrokenResponseWriter) Write(_ []byte) (int, error) {
	return 0, nil
}

func (rw *mockBrokenResponseWriter) WriteHeader(_ int) {
}

// recordingResponseWriter records every WriteHeader code it receives, unlike
// httptest.ResponseRecorder which latches on the first call. This is needed to
// verify that informational 1xx headers are forwarded to the underlying writer.
type recordingResponseWriter struct {
	header http.Header
	codes  []int
	body   bytes.Buffer
}

func newRecordingResponseWriter() *recordingResponseWriter {
	return &recordingResponseWriter{header: http.Header{}}
}

func (w *recordingResponseWriter) Header() http.Header {
	return w.header
}

//nolint:wrapcheck
func (w *recordingResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *recordingResponseWriter) WriteHeader(code int) {
	w.codes = append(w.codes, code)
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

	rr := httptest.NewRecorder()
	ww := responseWriterWrapper{ResponseWriter: rr}
	ww.Flush()
	require.True(t, ww.headerWritten, "expected flush to set headerWritten=true")
	require.Equal(t, http.StatusOK, ww.Status(), "expected flush to record the implicit 200 status")
	require.Equal(t, http.StatusOK, rr.Code)

	// A later WriteHeader must not override the status recorded by Flush.
	ww.WriteHeader(http.StatusInternalServerError)
	require.Equal(t, http.StatusOK, ww.Status())
}

func Test_responseWriterWrapper_Status(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := responseWriterWrapper{ResponseWriter: rr}
	ww.WriteHeader(http.StatusMultiStatus)
	require.Equal(t, http.StatusMultiStatus, ww.Status())
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
	ww.WriteHeader(http.StatusNoContent)
	require.Equal(t, http.StatusNoContent, ww.Status())
	ww.WriteHeader(http.StatusMovedPermanently)
	require.Equal(t, http.StatusNoContent, ww.Status())
}

func Test_responseWriterWrapper_WriteHeader_InformationalNotLatched(t *testing.T) {
	t.Parallel()

	rec := newRecordingResponseWriter()
	ww := responseWriterWrapper{ResponseWriter: rec}

	// 1xx responses (except 101) must be forwarded without being recorded as the
	// final status, so the real final status is still captured.
	ww.WriteHeader(http.StatusEarlyHints)       // 103, forwarded, not latched
	ww.WriteHeader(http.StatusProcessing)       // 102, forwarded, not latched
	ww.WriteHeader(http.StatusNotFound)         // 404, final, latched
	ww.WriteHeader(http.StatusMovedPermanently) // 301, ignored (already latched)

	require.Equal(t, []int{
		http.StatusEarlyHints,
		http.StatusProcessing,
		http.StatusNotFound,
	}, rec.codes, "1xx forwarded then final latched; a later WriteHeader is ignored")
	require.Equal(t, http.StatusNotFound, ww.Status())
}

func Test_responseWriterWrapper_WriteHeader_SwitchingProtocolsLatches(t *testing.T) {
	t.Parallel()

	rec := newRecordingResponseWriter()
	ww := responseWriterWrapper{ResponseWriter: rec}

	// 101 is a final status (net/http sends no further headers after it).
	ww.WriteHeader(http.StatusSwitchingProtocols)
	ww.WriteHeader(http.StatusOK) // ignored

	require.Equal(t, []int{http.StatusSwitchingProtocols}, rec.codes)
	require.Equal(t, http.StatusSwitchingProtocols, ww.Status())
}

// Test_responseWriterWrapper_EarlyHints_EndToEnd is a regression test: through a
// real net/http server, a handler that sends 103 Early Hints and then a final
// status must deliver that final status (not 200) to the client.
func Test_responseWriterWrapper_EarlyHints_EndToEnd(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rw := NewResponseWriterWrapper(w)
		rw.Header().Add("Link", "</style.css>; rel=preload; as=style")
		rw.WriteHeader(http.StatusEarlyHints)
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte("not found"))
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "client must see the final status, not the 103")
	require.Equal(t, "not found", string(body))
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

	rww, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	count, err := rww.ReadFrom(inputBuf)
	require.NoError(t, err)
	require.Equal(t, int64(10), count)

	// with tee writer
	mockTee := newMockResponseWriter()
	wwTee := NewResponseWriterWrapper(mockTee)
	require.NotNil(t, wwTee)

	teeBuf := bytes.NewBuffer([]byte{})
	wwTee.Tee(teeBuf)

	inputBufTee := bytes.NewBufferString("0123456789")

	rwwTee, ok := wwTee.(*responseWriterWrapper)
	require.True(t, ok)

	countTee, err := rwwTee.ReadFrom(inputBufTee)
	require.NoError(t, err)
	require.Equal(t, int64(10), countTee)
	require.Equal(t, "0123456789", teeBuf.String())

	wwTeeResponseWriterWrapper, ok := wwTee.(*responseWriterWrapper)
	require.True(t, ok)
	require.True(t, wwTeeResponseWriterWrapper.headerWritten)
}

// Test_responseWriterWrapper_ReadFrom_LimitReaderTee is a regression test for
// the tee-branch infinite recursion: io.Copy on a source lacking io.WriterTo
// (e.g. io.LimitReader) dispatches to the destination's ReadFrom, which used to
// call io.Copy on itself and overflow the stack. It also verifies the size is
// counted exactly once (no double count).
func Test_responseWriterWrapper_ReadFrom_LimitReaderTee(t *testing.T) {
	t.Parallel()

	mock := newMockResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	teeBuf := bytes.NewBuffer([]byte{})
	ww.Tee(teeBuf)

	src := io.LimitReader(strings.NewReader("0123456789"), 10)

	rww, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	count, err := rww.ReadFrom(src)
	require.NoError(t, err)
	require.Equal(t, int64(10), count)
	require.Equal(t, 10, ww.Size(), "size must be counted exactly once")
	require.Equal(t, "0123456789", teeBuf.String())
	require.Equal(t, "0123456789", mock.String())
}

// Test_responseWriterWrapper_ReadFrom_NoReaderFromFallback verifies that when
// the underlying ResponseWriter does not implement io.ReaderFrom, ReadFrom
// falls back to a generic copy instead of returning an error.
func Test_responseWriterWrapper_ReadFrom_NoReaderFromFallback(t *testing.T) {
	t.Parallel()

	mock := &mockPlainResponseWriter{}
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	src := io.LimitReader(strings.NewReader("0123456789"), 10)

	rww, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	count, err := rww.ReadFrom(src)
	require.NoError(t, err)
	require.Equal(t, int64(10), count)
	require.Equal(t, 10, ww.Size())
	require.Equal(t, "0123456789", mock.buf.String())
	require.True(t, rww.headerWritten)
	require.Equal(t, http.StatusOK, ww.Status())
}

func Test_broken_responseWriterWrapper_ReadFrom(t *testing.T) {
	t.Parallel()

	mock := newMockBrokenResponseWriter()
	ww := NewResponseWriterWrapper(mock)
	require.NotNil(t, ww)

	inputBuf := bytes.NewBufferString("-")

	rww, ok := ww.(*responseWriterWrapper)
	require.True(t, ok)

	// The broken writer accepts no bytes (0, nil), so the generic-copy
	// fallback must surface a short-write error.
	_, err := rww.ReadFrom(inputBuf)
	require.Error(t, err)
}

func Test_responseWriterWrapper_Unwrap(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	ww := responseWriterWrapper{ResponseWriter: rr}

	// Unwrap must return the wrapped writer so http.ResponseController can
	// reach the underlying implementation.
	require.Equal(t, reflect.ValueOf(rr).Pointer(), reflect.ValueOf(ww.Unwrap()).Pointer())
}
