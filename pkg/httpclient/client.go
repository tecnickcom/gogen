package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/tecnickcom/nurago/pkg/random"
	"github.com/tecnickcom/nurago/pkg/redact"
	"github.com/tecnickcom/nurago/pkg/traceid"
)

// defaultMaxDumpSize is the default cap (in bytes, by advertised Content-Length)
// on request/response bodies buffered into debug-level payload dumps.
const defaultMaxDumpSize = 1 << 20 // 1 MiB

// defaultMaxIdleConnsPerHost is the per-host idle-connection pool size for the
// default transport. http.DefaultTransport leaves this at 0 (net/http's default
// of 2), which throttles connection reuse for a client that concentrates traffic
// on a few downstream hosts — the service-to-service use case this package
// targets. It is raised to match the default MaxIdleConns so the per-host pool is
// not the bottleneck.
const defaultMaxIdleConnsPerHost = 100

// logMessage is the constant slog message used for outbound-call log entries.
// It is deliberately independent of the configured log-field prefix so that
// messages stay stable and filterable across services regardless of prefixing.
const logMessage = "outbound"

// truncatedBodyMarker is appended to a response dump whose body was truncated to
// the configured dump cap (see WithMaxDumpSize).
const truncatedBodyMarker = "\r\n[response body truncated by httpclient dump cap]\r\n"

// Sentinel errors returned by Do for a malformed request, mirroring the standard
// library's behavior of returning an error (rather than panicking) instead of
// dereferencing a nil field.
var (
	// ErrNilRequest is returned by Do when the request is nil.
	ErrNilRequest = errors.New("httpclient: nil request")

	// ErrNilRequestURL is returned by Do when the request has a nil URL.
	ErrNilRequestURL = errors.New("httpclient: nil Request.URL")
)

// Client wraps http.Client with trace ID propagation, structured request/response logging, and optional debug payload dumps.
type Client struct {
	client            *http.Client
	timeout           time.Duration
	component         string
	logPrefix         string
	traceIDHeaderName string
	redactFn          RedactFn
	logger            *slog.Logger
	rnd               *random.Rnd
	maxDumpSize       int64
}

// defaultTransport returns a private transport for a new client.
// It clones http.DefaultTransport so that per-client options (e.g. WithDialContext)
// never mutate the process-wide http.DefaultTransport, which would race across
// concurrent New calls and change the dialer for every other consumer. It also
// raises MaxIdleConnsPerHost so per-host connection reuse is not capped at 2.
func defaultTransport() http.RoundTripper {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// http.DefaultTransport is always an *http.Transport in the standard
		// library; this guards against a future change or test override.
		return http.DefaultTransport
	}

	clone := t.Clone()
	clone.MaxIdleConnsPerHost = defaultMaxIdleConnsPerHost

	return clone
}

// defaultClient() returns a default client.
func defaultClient() *Client {
	return &Client{
		client: &http.Client{
			// Timeout is intentionally left zero: the per-request deadline is
			// applied via the request context in Do (see the timeout field), so
			// it also reaches custom round-trippers and dialers. Setting both
			// would arm two overlapping timers per request.
			Transport: defaultTransport(),
		},
		timeout:           1 * time.Minute,
		traceIDHeaderName: traceid.DefaultHeader,
		component:         "-",
		redactFn:          redact.Default().BytesToString,
		logger:            slog.Default(),
		rnd:               random.New(nil),
		maxDumpSize:       defaultMaxDumpSize,
	}
}

// New constructs an HTTP client with 1-minute timeout, trace ID propagation, and request/response logging from provided options.
func New(opts ...Option) *Client {
	c := defaultClient()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	return c
}

// CloseIdleConnections closes any connections on the client's transport that are
// currently idle, without interrupting in-flight requests. It is safe for
// concurrent use and no-ops for transports that do not implement the optional
// CloseIdleConnections method.
func (c *Client) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}

// cancelReadCloser wraps a response body so that closing it also releases the
// per-request timeout context (canceling its timer promptly) while still
// allowing the timeout to interrupt long reads that happen before Close.
type cancelReadCloser struct {
	io.ReadCloser

	cancel context.CancelFunc
}

// Close closes the underlying body and cancels the per-request context.
func (b *cancelReadCloser) Close() error {
	defer b.cancel()

	return b.ReadCloser.Close() //nolint:wrapcheck
}

// Do executes the request with trace ID attachment, structured logging, and optional debug payload dumps; returns error from underlying client.
//
// Do operates on a private clone, so the caller's request keeps its original
// headers and context. The request body, however, is consumed as usual for a
// single-use request (the clone shares the body with the original).
//
// A nil request returns ErrNilRequest and a request with a nil URL returns
// ErrNilRequestURL, mirroring the standard client's error-on-malformed-request
// behavior instead of panicking.
//
// The returned response body is wrapped so that the per-request timeout context
// is canceled when the body is closed. The caller is therefore responsible for
// closing resp.Body (e.g. defer resp.Body.Close()) on the success path to avoid
// leaking the timeout timer until the timeout elapses; failing to close the body
// also leaks the underlying connection as with the standard net/http client.
//
// The log entry records the response status as response_code and, when the
// response advertises one, its Content-Length as response_content_length. That
// is the advertised length, not the number of bytes actually read by the caller
// (which is unknown at log time); it is omitted for chunked or unknown length
// responses. On failure the error is logged with its URL query and userinfo
// redacted, so query-parameter secrets are not written to logs.
//
// At debug level the request and response are dumped (after redaction). Dumps
// buffer the body in memory up to the configured maximum (see WithMaxDumpSize):
// over-cap request bodies (and unknown-length request bodies) are dumped without
// their payload, while an unknown-length response body is truncated to the cap in
// the dump (the caller still receives the full body). Because a debug-level
// response dump reads the body before the entry is emitted, response_duration
// includes body transfer time (up to the cap) at debug level but only
// time-to-response-headers at non-debug levels.
//
//nolint:gocognit,gocyclo,cyclop,funlen
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	if r == nil {
		return nil, ErrNilRequest
	}

	if r.URL == nil {
		return nil, ErrNilRequestURL
	}

	// start keeps the monotonic clock reading so response_duration is immune to
	// wall-clock adjustments; reqTime is the wall-clock value for the log field
	// (time.Time.UTC strips the monotonic reading).
	start := time.Now()
	reqTime := start.UTC()

	// The per-request deadline is applied to the request context (not via
	// http.Client.Timeout) so it also propagates to custom round-trippers and
	// dialers. A non-positive timeout means "no timeout" (net/http convention),
	// so the context is only made cancelable, never given an expired deadline.
	ctx := r.Context()

	var cancel context.CancelFunc

	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	// cancel is transferred to the response body on the success path; until then
	// (and on every early return, including a panic in a user-supplied
	// round-tripper) the deferred guard releases it so the timeout timer never
	// leaks.
	ownedCancel := true

	defer func() {
		if ownedCancel {
			cancel()
		}
	}()

	l := c.logger.With(c.logPrefix+"component", c.component)
	debug := l.Enabled(ctx, slog.LevelDebug)

	var err error

	defer func() {
		resTime := time.Now()
		l = l.With(
			slog.Time(c.logPrefix+"response_time", resTime.UTC()),
			slog.Duration(c.logPrefix+"response_duration", resTime.Sub(start)),
		)

		if err != nil {
			l.With(slog.Any(c.logPrefix+"error", c.redactErrorForLog(err))).Error(logMessage)
			return
		}

		if debug {
			l.Debug(logMessage)
			return
		}

		l.Info(logMessage)
	}()

	// Operate on a private clone so the trace-ID header write and any debug body
	// dump never mutate the caller's request.
	r = r.Clone(ctx)

	reqID, r := c.propagateTraceID(ctx, r)

	l = l.With(
		slog.String(c.logPrefix+traceid.DefaultLogKey, reqID),
		slog.Time(c.logPrefix+"request_time", reqTime),
		slog.String(c.logPrefix+"request_method", r.Method),
		slog.String(c.logPrefix+"request_host", r.URL.Host),
		slog.String(c.logPrefix+"request_path", r.URL.Path),
		slog.String(c.logPrefix+"request_query", c.redactFn([]byte(r.URL.RawQuery))),
	)

	if debug {
		reqDump, errd := c.dumpRequest(r)
		if errd != nil {
			l = l.With(slog.String(c.logPrefix+"request_dump_error", errd.Error()))
		} else {
			l = l.With(slog.String(c.logPrefix+"request", c.redactFn(reqDump)))
		}
	}

	var resp *http.Response

	resp, err = c.client.Do(r)

	if resp != nil {
		// The response status is the primary diagnostic for an outbound call, so
		// it is logged at every level; the advertised length is logged only when
		// known (it is the Content-Length, not the bytes the caller reads).
		attrs := []any{slog.Int(c.logPrefix+"response_code", resp.StatusCode)}
		if resp.ContentLength >= 0 {
			attrs = append(attrs, slog.Int64(c.logPrefix+"response_content_length", resp.ContentLength))
		}

		l = l.With(attrs...)
	}

	if debug && resp != nil {
		respDump, errd := c.dumpResponse(resp)
		if errd != nil {
			l = l.With(slog.String(c.logPrefix+"response_dump_error", errd.Error()))
		} else {
			l = l.With(slog.String(c.logPrefix+"response", c.redactFn(respDump)))
		}
	}

	if err != nil || resp == nil {
		// No body to hang the cancel on; the deferred guard releases the timer.
		return resp, err //nolint:wrapcheck
	}

	// Tie the lifetime of the timeout context to the response body: closing the
	// body cancels the context (releasing the timer), while leaving it open
	// preserves the timeout's ability to interrupt long reads.
	resp.Body = &cancelReadCloser{ReadCloser: resp.Body, cancel: cancel}
	ownedCancel = false

	return resp, err //nolint:wrapcheck
}

// propagateTraceID resolves the outbound trace ID and writes it to the request,
// returning the resolved ID and the request to send (which may be re-wrapped
// with an updated context).
//
// The ID is chosen in priority order: a valid ID already in the context, then a
// valid ID already on the request header (so an explicit caller-set trace header
// is honored rather than clobbered), then a freshly generated UUIDv7. The
// traceid helpers validate the ID, so an invalid value never propagates and the
// UUID is only generated (and crypto/rand only read) when nothing valid is
// present. The resolved ID is forced onto the request context so the propagated
// header and the context value agree, skipping the re-wrap (and its allocation)
// when the context already carries the ID.
func (c *Client) propagateTraceID(ctx context.Context, r *http.Request) (string, *http.Request) {
	reqID := traceid.SetHTTPRequestHeaderFromContext(ctx, r, c.traceIDHeaderName, "")
	if reqID == "" {
		reqID = traceid.FromHTTPRequestHeader(r, c.traceIDHeaderName, "")
		if reqID == "" {
			reqID = c.rnd.UUIDv7().String()
		}

		r.Header.Set(c.traceIDHeaderName, reqID)
	}

	if newCtx := traceid.ForceContext(ctx, reqID); newCtx != ctx {
		r = r.WithContext(newCtx)
	}

	return reqID, r
}

// redactErrorForLog returns a log-safe view of err. A failed request yields a
// *url.Error whose message embeds the full request URL; its query string and
// userinfo are redacted here (matching how request_query is redacted) so
// query-parameter or basic-auth secrets do not leak into error logs. Any other
// error, and the error returned to the caller by Do, is left unchanged.
func (c *Client) redactErrorForLog(err error) error {
	var uerr *url.Error
	if !errors.As(err, &uerr) {
		// Errors that do not wrap a *url.Error (e.g. produced by a custom
		// round-tripper) are logged as-is; such round-trippers must redact their
		// own messages.
		return err
	}

	// Redact a copy so the error returned to the caller stays intact. If a
	// *url.Error were nested under outer wrapping, returning the copy drops that
	// wrapping from the log message; in practice Do's error comes straight from
	// http.Client.Do as a top-level *url.Error, so no context is lost.
	cp := *uerr
	cp.URL = c.redactURLForLog(uerr.URL)

	return &cp
}

// redactURLForLog masks a raw URL string for logging: any userinfo (which may
// carry a token) is dropped and the query string is redacted. A string that does
// not parse as a URL is still query-redacted by a plain split, so no secret is
// emitted either way.
func (c *Client) redactURLForLog(raw string) string {
	u, perr := url.Parse(raw)
	if perr != nil {
		return redactQueryTail(raw, c.redactFn)
	}

	u.User = nil

	if u.RawQuery != "" {
		u.RawQuery = c.redactFn([]byte(u.RawQuery))
	}

	return u.String()
}

// redactQueryTail redacts the query portion (everything after the first '?') of a
// raw string. It is the fallback used when a URL string does not parse.
func redactQueryTail(raw string, redactFn RedactFn) string {
	q := strings.IndexByte(raw, '?')
	if q < 0 {
		return raw
	}

	return raw[:q+1] + redactFn([]byte(raw[q+1:]))
}

// dumpRequest returns a debug dump of the outbound request, choosing whether to
// include the request body so that dumping never buffers an unbounded or
// streaming body.
//
//   - A nil or empty body is dumped as headers only.
//   - A body of unknown length (ContentLength < 0, e.g. a streaming io.Pipe) has
//     its body omitted: httputil.DumpRequestOut with body=true would block until
//     the body reaches EOF, which for an unsent streaming request would deadlock.
//   - A body whose known length exceeds maxDumpSize has its payload omitted (a
//     body-stripped copy is dumped) to bound memory; note that DumpRequestOut
//     with body=false would otherwise write ContentLength filler bytes, so a
//     copy with the body removed is required to actually bound the dump.
//   - Otherwise the real body is included.
//
// A non-positive maxDumpSize disables the size cap (streaming bodies are still
// omitted to avoid the deadlock).
func (c *Client) dumpRequest(r *http.Request) ([]byte, error) {
	switch {
	case r.Body == nil || r.Body == http.NoBody:
		return httputil.DumpRequestOut(r, false) //nolint:wrapcheck
	case r.ContentLength < 0:
		return httputil.DumpRequestOut(r, false) //nolint:wrapcheck
	case c.maxDumpSize > 0 && r.ContentLength > c.maxDumpSize:
		stripped := r.Clone(r.Context())
		stripped.Body = nil
		stripped.ContentLength = 0
		stripped.GetBody = nil

		return httputil.DumpRequestOut(stripped, false) //nolint:wrapcheck
	default:
		return httputil.DumpRequestOut(r, true) //nolint:wrapcheck
	}
}

// dumpResponse returns a debug dump of the response, bounding the buffered body
// to maxDumpSize even when the response has no advertised length (chunked or
// streaming).
//
//   - A known length within the cap (or a disabled cap) uses the standard dumper,
//     which is already bounded by that length.
//   - A known length over the cap has its body omitted (headers only).
//   - An unknown length with a cap is peeked up to the cap: the dump carries the
//     headers plus at most maxDumpSize body bytes (marked as truncated when the
//     body is larger), and the response body is restored so the caller still
//     receives the complete stream.
func (c *Client) dumpResponse(resp *http.Response) ([]byte, error) {
	if resp.ContentLength >= 0 || c.maxDumpSize <= 0 {
		return httputil.DumpResponse(resp, c.dumpResponseBody(resp.ContentLength)) //nolint:wrapcheck
	}

	original := resp.Body

	// Read one byte past the cap so truncation can be detected, guarding against
	// int64 overflow for an extreme cap.
	limit := c.maxDumpSize
	if limit < math.MaxInt64 {
		limit++
	}

	buf, err := io.ReadAll(io.LimitReader(original, limit))
	if err != nil {
		// Preserve whatever was read so the caller can still consume the body.
		resp.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(buf), original), closer: original}

		return nil, err //nolint:wrapcheck
	}

	// LimitReader caps the read at maxDumpSize+1, so an over-cap body yields
	// exactly maxDumpSize+1 bytes; dropping the last one gives the capped prefix.
	truncated := int64(len(buf)) > c.maxDumpSize

	dumpBytes := buf
	if truncated {
		dumpBytes = buf[:len(buf)-1]
	}

	// DumpResponse here reads from a bounded in-memory reader into a bytes.Buffer,
	// so it cannot fail; the only failure mode of an unknown-length dump is the
	// body read handled above. The error is therefore discarded.
	resp.Body = io.NopCloser(bytes.NewReader(dumpBytes))
	dump, _ := httputil.DumpResponse(resp, true)

	// Restore a body that replays the peeked prefix followed by the remainder.
	resp.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(buf), original), closer: original}

	if truncated {
		dump = append(dump, truncatedBodyMarker...)
	}

	return dump, nil
}

// dumpResponseBody reports whether a response body of the given advertised length
// should be included in the standard dump: a body whose known length exceeds
// maxDumpSize is omitted. A non-positive maxDumpSize disables the cap. Unknown
// lengths are handled separately by dumpResponse (which caps the buffered bytes).
func (c *Client) dumpResponseBody(contentLength int64) bool {
	return c.maxDumpSize <= 0 || contentLength <= c.maxDumpSize
}

// replayBody re-serves buffered response bytes followed by the unread remainder,
// so a response whose body was partially read to build a bounded debug dump can
// still be fully consumed by the caller. Closing it closes the underlying body.
type replayBody struct {
	io.Reader

	closer io.Closer
}

// Close closes the underlying response body.
func (b *replayBody) Close() error {
	return b.closer.Close() //nolint:wrapcheck
}
