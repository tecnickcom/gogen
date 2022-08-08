package httputil

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

// NewResponseWriterWrapper wraps an http.ResponseWriter with an enhanced proxy.
func NewResponseWriterWrapper(w http.ResponseWriter) ResponseWriterWrapper {
	return &responseWriterWrapper{ResponseWriter: w}
}

// ResponseWriterWrapper is the interface defining the extendend functions of the proxy.
type ResponseWriterWrapper interface {
	http.ResponseWriter

	// Size returns the total number of bytes sent to the client.
	Size() int

	// Status returns the HTTP status of the request.
	Status() int

	// Tee sets a writer that will contain a copy of the bytes written to the response writer.
	Tee(io.Writer)
}

type responseWriterWrapper struct {
	http.ResponseWriter
	headerWritten bool
	size          int
	status        int
	tee           io.Writer
}

func (b *responseWriterWrapper) Size() int {
	return b.size
}

func (b *responseWriterWrapper) Flush() {
	b.headerWritten = true

	fl, ok := b.ResponseWriter.(http.Flusher)
	if ok {
		fl.Flush()
	}
}

//nolint:wrapcheck
func (b *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := b.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the Hijacker is not supported by the ResponseWriter")
	}

	return hj.Hijack()
}

//nolint:wrapcheck
func (b *responseWriterWrapper) Push(target string, opts *http.PushOptions) error {
	pusher, ok := b.ResponseWriter.(http.Pusher)
	if !ok {
		return fmt.Errorf("the Pusher is not supported by the ResponseWriter")
	}

	return pusher.Push(target, opts)
}

//nolint:wrapcheck
func (b *responseWriterWrapper) ReadFrom(r io.Reader) (int64, error) {
	if b.tee != nil {
		n, err := io.Copy(b, r)
		b.size += int(n)

		return n, err
	}

	rf, ok := b.ResponseWriter.(io.ReaderFrom)
	if !ok {
		return 0, fmt.Errorf("the ReaderFrom is not supported by the ResponseWriter")
	}

	b.maybeWriteHeader()

	n, err := rf.ReadFrom(r)

	b.size += int(n)

	return n, err
}

func (b *responseWriterWrapper) Status() int {
	return b.status
}

func (b *responseWriterWrapper) Tee(w io.Writer) {
	b.tee = w
}

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

func (b *responseWriterWrapper) WriteHeader(code int) {
	if !b.headerWritten {
		b.status = code
		b.headerWritten = true
		b.ResponseWriter.WriteHeader(code)
	}
}

func (b *responseWriterWrapper) maybeWriteHeader() {
	if !b.headerWritten {
		b.WriteHeader(http.StatusOK)
	}
}
