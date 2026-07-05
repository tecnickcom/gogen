package healthcheck

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/tecnickcom/gogen/pkg/httputil"
)

const (
	// StatusOK is the canonical success value used in per-check results.
	StatusOK = "OK"
)

var (
	// ErrCheckTimeout is reported for a check that did not return before the
	// configured handler timeout (or the request context) elapsed. See
	// [WithTimeout].
	ErrCheckTimeout = errors.New("healthcheck timed out")

	// ErrCheckPanic is reported in place of a checker panic value. The panic and
	// its stack trace are logged through the handler logger instead of being
	// exposed in the HTTP response.
	ErrCheckPanic = errors.New("healthcheck panicked")

	// ErrNoChecker is reported for a registered check whose Checker is nil. It is
	// a configuration error (also warned at construction) surfaced as a plain
	// failure rather than a recovered nil-pointer panic.
	ErrNoChecker = errors.New("healthcheck has no checker")
)

// ResultWriter writes aggregated healthcheck output to an HTTP response.
type ResultWriter func(ctx context.Context, w http.ResponseWriter, statusCode int, data any)

// Handler aggregates and serves healthcheck results over HTTP.
//
// It responds to any HTTP method and spawns one goroutine per registered check.
// The fan-out is per request, so the number of in-flight goroutines is
// (concurrent requests × checks); keep the check set bounded and the checks
// reasonably fast, and avoid probing this endpoint at extreme frequency.
type Handler struct {
	checks      []HealthCheck
	writeResult ResultWriter
	logger      *slog.Logger
	timeout     time.Duration
}

// NewHandler builds an HTTP healthcheck aggregator handler.
//
// It executes registered checks concurrently and writes a combined response via
// the configured ResultWriter. The provided slice is copied, so later caller
// mutations do not affect the handler.
func NewHandler(checks []HealthCheck, opts ...HandlerOption) *Handler {
	h := &Handler{
		checks: append([]HealthCheck(nil), checks...),
		logger: slog.Default(),
	}

	for _, apply := range opts {
		apply(h)
	}

	// Normalize a nil logger (for example from WithLogger(nil)) to slog.Default()
	// so warning and recovered-panic logging never dereference a nil pointer.
	if h.logger == nil {
		h.logger = slog.Default()
	}

	// Build the default result writer after options are applied so that
	// WithLogger affects it, while WithResultWriter still takes precedence.
	if h.writeResult == nil {
		h.writeResult = httputil.NewHTTPResp(h.logger).SendJSON
	}

	warnInvalidChecks(h.logger, h.checks)

	return h
}

// warnInvalidChecks logs a warning for misconfigured registrations: empty or
// duplicate IDs (which collapse into a single entry in the aggregated response
// map) and nil checkers (which always fail with [ErrNoChecker]). Every check is
// still registered; the warnings surface the problem at startup.
func warnInvalidChecks(logger *slog.Logger, checks []HealthCheck) {
	seen := make(map[string]struct{}, len(checks))

	for _, hc := range checks {
		if hc.ID == "" {
			logger.Warn("healthcheck registered with an empty ID")
		} else if _, dup := seen[hc.ID]; dup {
			logger.Warn("healthcheck registered with a duplicate ID", slog.String("id", hc.ID))
		}

		if hc.Checker == nil {
			logger.Warn("healthcheck registered with a nil checker", slog.String("id", hc.ID))
		}

		seen[hc.ID] = struct{}{}
	}
}

// ServeHTTP executes all configured checks in parallel and writes aggregated output.
//
// The response status is 200 when all checks pass, otherwise 503. The response
// body maps check IDs to "OK" or an error message; a check with the same ID as
// another is reported as failed if any of them fail. When [WithTimeout] is set,
// checks that do not return in time are reported with [ErrCheckTimeout].
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	results := h.runChecks(ctx)

	status := http.StatusOK
	data := make(map[string]string, len(results))

	for _, res := range results {
		if res.err != nil {
			status = http.StatusServiceUnavailable
			data[res.id] = res.err.Error()

			continue
		}

		// Do not overwrite a failure recorded under the same ID with "OK".
		if _, recorded := data[res.id]; !recorded {
			data[res.id] = StatusOK
		}
	}

	// Write with the original request context: the response must be produced even
	// when the check context has already timed out.
	h.writeResult(r.Context(), w, status, data)
}

// checkResult is the outcome of a single check, kept in registration order.
type checkResult struct {
	id  string
	err error
}

// indexedResult carries a check outcome back to the request goroutine together
// with the slot it belongs to, so only that goroutine writes the results slice.
type indexedResult struct {
	index int
	err   error
}

// runChecks launches every check concurrently and collects their results in
// registration order. Checks that do not report before ctx is done keep their
// pre-seeded [ErrCheckTimeout] outcome.
func (h *Handler) runChecks(ctx context.Context) []checkResult {
	n := len(h.checks)

	results := make([]checkResult, n)
	for i := range results {
		results[i] = checkResult{id: h.checks[i].ID, err: ErrCheckTimeout}
	}

	// Buffered to the number of checks so a check can always deliver its result
	// without blocking, even after collection has stopped on timeout.
	resCh := make(chan indexedResult, n)

	for i, hc := range h.checks {
		go h.runCheck(ctx, i, hc, resCh)
	}

	for remaining := n; remaining > 0; remaining-- {
		select {
		case res := <-resCh:
			results[res.index].err = res.err
		case <-ctx.Done():
			// On deadline or cancellation, still record any results that already
			// arrived, so checks that completed in time are not misreported as
			// timed out. Checks that have not reported keep their seeded outcome.
			drainResults(results, resCh)

			return results
		}
	}

	return results
}

// drainResults records every result currently buffered in resCh without
// blocking, leaving not-yet-reported checks with their pre-seeded outcome.
func drainResults(results []checkResult, resCh <-chan indexedResult) {
	for {
		select {
		case res := <-resCh:
			results[res.index].err = res.err
		default:
			return
		}
	}
}

// runCheck executes a single check and reports its outcome on resCh, converting
// a panic into a regular failure. An unrecovered panic in this child goroutine
// would crash the whole process, since net/http panic recovery only covers the
// request goroutine.
func (h *Handler) runCheck(ctx context.Context, index int, hc HealthCheck, resCh chan<- indexedResult) {
	defer func() {
		if p := recover(); p != nil {
			h.logger.ErrorContext(ctx, "healthcheck checker panicked",
				slog.String("id", hc.ID),
				slog.Any("panic", p),
				slog.String("stack", string(debug.Stack())),
			)

			resCh <- indexedResult{index: index, err: ErrCheckPanic}
		}
	}()

	// Guard a nil checker so it fails cleanly instead of triggering (and logging)
	// a recovered nil-pointer panic on every probe.
	if hc.Checker == nil {
		resCh <- indexedResult{index: index, err: ErrNoChecker}

		return
	}

	resCh <- indexedResult{index: index, err: hc.Checker.HealthCheck(ctx)}
}
