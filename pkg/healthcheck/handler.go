package healthcheck

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/tecnickcom/gogen/pkg/httputil"
)

const (
	// StatusOK is the canonical success value used in per-check results.
	StatusOK = "OK"
)

// ResultWriter writes aggregated healthcheck output to an HTTP response.
type ResultWriter func(ctx context.Context, w http.ResponseWriter, statusCode int, data any)

// Handler aggregates and serves healthcheck results over HTTP.
type Handler struct {
	checks      []HealthCheck
	checksCount int
	writeResult ResultWriter
	logger      *slog.Logger
}

// NewHandler builds an HTTP healthcheck aggregator handler.
//
// It executes registered checks concurrently and writes a combined response via
// the configured ResultWriter.
func NewHandler(checks []HealthCheck, opts ...HandlerOption) *Handler {
	h := &Handler{
		checks:      checks,
		checksCount: len(checks),
		logger:      slog.Default(),
	}

	h.writeResult = httputil.NewHTTPResp(h.logger).SendJSON

	for _, apply := range opts {
		apply(h)
	}

	return h
}

// ServeHTTP executes all configured checks in parallel and writes aggregated output.
//
// The response status is 200 when all checks pass, otherwise 503. The response
// body maps check IDs to "OK" or error messages.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type checkResult struct {
		id  string
		err error
	}

	resCh := make(chan checkResult, h.checksCount)
	defer close(resCh)

	var wg sync.WaitGroup

	wg.Add(h.checksCount)

	for _, hc := range h.checks {
		go func() { //nolint:contextcheck
			defer wg.Done()

			resCh <- checkResult{
				id:  hc.ID,
				err: hc.Checker.HealthCheck(r.Context()),
			}
		}()
	}

	wg.Wait()

	status := http.StatusOK
	data := make(map[string]string, h.checksCount)

	for len(resCh) > 0 {
		r := <-resCh
		data[r.id] = StatusOK

		if r.err != nil {
			status = http.StatusServiceUnavailable
			data[r.id] = r.err.Error()
		}
	}

	h.writeResult(r.Context(), w, status, data)
}
