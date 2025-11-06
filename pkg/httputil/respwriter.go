package httputil

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
)

// ResponseWriterWrapper is the interface defining the extendend functions of the proxy.
type ResponseWriterWrapper interface {
	http.ResponseWriter

	// Size returns the total number of bytes sent to the client.
	Size() int

	// Status returns the HTTP status of the request.
	Status() int

	// Tee sets a writer that will contain a copy of the bytes written to the response writer.
	Tee(w io.Writer)
}

// responseWriterWrapper implements the ResponseWriterWrapper interface.
type responseWriterWrapper struct {
	http.ResponseWriter

	headerWritten bool
	size          int
	status        int
	tee           io.Writer
}

// NewResponseWriterWrapper wraps an http.ResponseWriter with an enhanced proxy.
func NewResponseWriterWrapper(w http.ResponseWriter) ResponseWriterWrapper {
	return &responseWriterWrapper{ResponseWriter: w}
}

// Size returns the total number of bytes sent to the client.
func (b *responseWriterWrapper) Size() int {
	return b.size
}

// Flush implements the http.Flusher interface.
func (b *responseWriterWrapper) Flush() {
	b.headerWritten = true

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
	if b.tee != nil {
		n, err := io.Copy(b, r)
		b.size += int(n)

		return n, err
	}

	rf, ok := b.ResponseWriter.(io.ReaderFrom)
	if !ok {
		return 0, errors.New("the ReaderFrom is not supported by the ResponseWriter")
	}

	b.maybeWriteHeader()

	n, err := rf.ReadFrom(r)

	b.size += int(n)

	return n, err
}

// Status returns the HTTP status of the request.
func (b *responseWriterWrapper) Status() int {
	return b.status
}

// Tee sets a writer that will contain a copy of the bytes written to the response writer.
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
func (b *responseWriterWrapper) WriteHeader(code int) {
	if !b.headerWritten {
		b.status = code
		b.headerWritten = true
		b.ResponseWriter.WriteHeader(code)
	}
}

// maybeWriteHeader writes the header if it has not been written yet.
func (b *responseWriterWrapper) maybeWriteHeader() {
	if !b.headerWritten {
		b.WriteHeader(http.StatusOK)
	}
}
