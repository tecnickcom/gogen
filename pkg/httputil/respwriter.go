package httputil

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
)

// ResponseWriterWrapper augments [http.ResponseWriter] with status/size tracking and tee support.
type ResponseWriterWrapper interface {
	http.ResponseWriter

	// Size returns the total number of bytes sent to the client.
	Size() int

	// Status returns the HTTP response status code written to the client.
	// It returns 0 until the header is written; callers that need the effective
	// value should treat 0 as the implicit 200 that net/http sends.
	Status() int

	// Tee sets a writer that receives a copy of the bytes written to the response
	// writer. Only bytes written after this call are copied. A tee-write failure is
	// surfaced as the error return of Write, after the client bytes have already
	// been sent. Setting a tee also routes ReadFrom through a generic copy,
	// disabling any io.ReaderFrom (sendfile) fast path on the underlying writer.
	Tee(w io.Writer)
}

// responseWriterWrapper is the concrete [ResponseWriterWrapper] implementation.
type responseWriterWrapper struct {
	http.ResponseWriter

	headerWritten bool
	size          int
	status        int
	tee           io.Writer
}

// NewResponseWriterWrapper wraps http.ResponseWriter with status/size capture and optional tee support.
func NewResponseWriterWrapper(w http.ResponseWriter) ResponseWriterWrapper {
	return &responseWriterWrapper{ResponseWriter: w}
}

// Size returns the total number of bytes written to the response.
func (b *responseWriterWrapper) Size() int {
	return b.size
}

// Flush implements the http.Flusher interface.
func (b *responseWriterWrapper) Flush() {
	// Record the implicit 200 status (as net/http does when flushing before
	// any explicit WriteHeader call) so that Status() reflects reality.
	b.maybeWriteHeader()

	fl, ok := b.ResponseWriter.(http.Flusher)
	if ok {
		fl.Flush()
	}
}

// Hijack implements the http.Hijacker interface.
//
//nolint:wrapcheck
func (b *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := b.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("the Hijacker is not supported by the ResponseWriter")
	}

	return hj.Hijack()
}

// Push implements the http.Pusher interface.
//
//nolint:wrapcheck
func (b *responseWriterWrapper) Push(target string, opts *http.PushOptions) error {
	pusher, ok := b.ResponseWriter.(http.Pusher)
	if !ok {
		return errors.New("the Pusher is not supported by the ResponseWriter")
	}

	return pusher.Push(target, opts)
}

// ReadFrom implements the io.ReaderFrom interface.
//
//nolint:wrapcheck
func (b *responseWriterWrapper) ReadFrom(r io.Reader) (int64, error) {
	rf, ok := b.ResponseWriter.(io.ReaderFrom)
	if b.tee != nil || !ok {
		// Copy through Write (which tees, counts the size, and records the
		// implicit header) via a wrapper that hides this ReadFrom method,
		// otherwise io.Copy would dispatch straight back here and recurse
		// infinitely for sources lacking io.WriterTo (e.g. io.LimitReader).
		return io.Copy(struct{ io.Writer }{b}, r)
	}

	b.maybeWriteHeader()

	n, err := rf.ReadFrom(r)

	b.size += int(n)

	return n, err
}

// Status returns the final HTTP status code written to the response, or 0 if no
// final header has been written yet (in which case net/http sends an implicit
// 200). Informational 1xx responses (except 101) are not reported here.
func (b *responseWriterWrapper) Status() int {
	return b.status
}

// Unwrap returns the wrapped http.ResponseWriter, allowing http.ResponseController
// to reach functionality of the underlying implementation that this wrapper does
// not re-expose (e.g. SetReadDeadline, SetWriteDeadline, EnableFullDuplex).
func (b *responseWriterWrapper) Unwrap() http.ResponseWriter {
	return b.ResponseWriter
}

// Tee sets a writer to receive a copy of all bytes written to the response.
func (b *responseWriterWrapper) Tee(w io.Writer) {
	b.tee = w
}

// Write writes data to the connection as part of an HTTP reply.
func (b *responseWriterWrapper) Write(buf []byte) (int, error) {
	b.maybeWriteHeader()
	n, err := b.ResponseWriter.Write(buf)

	if b.tee != nil {
		_, teeErr := b.tee.Write(buf[:n])

		if err == nil {
			err = teeErr
		}
	}

	b.size += n

	return n, err
}

// WriteHeader sends an HTTP response header with the provided status code.
//
// Informational responses (1xx, except 101 Switching Protocols) do not conclude
// the header exchange: they are forwarded to the underlying writer without being
// recorded as the final status, so a subsequent WriteHeader still captures the
// real status. This mirrors net/http, which treats 101 as a final status.
func (b *responseWriterWrapper) WriteHeader(code int) {
	if b.headerWritten {
		return
	}

	if code >= 100 && code <= 199 && code != http.StatusSwitchingProtocols {
		b.ResponseWriter.WriteHeader(code)
		return
	}

	b.status = code
	b.headerWritten = true
	b.ResponseWriter.WriteHeader(code)
}

// maybeWriteHeader writes the header if it has not been written yet.
func (b *responseWriterWrapper) maybeWriteHeader() {
	if !b.headerWritten {
		b.WriteHeader(http.StatusOK)
	}
}
