package healthcheck

import (
	"context"
	"net/http"
	"sync"

	"github.com/Vonage/gosrvlib/pkg/httputil"
)

const (
	// StatusOK represents an OK status.
	StatusOK = "OK"
)

// ResultWriter is a type alias for a function in charge of writing the result of the health checks.
type ResultWriter func(ctx context.Context, w http.ResponseWriter, statusCode int, data interface{})

// NewHandler creates a new instance of the healthcheck handler.
func NewHandler(checks []HealthCheck, opts ...HandlerOption) *Handler {
	h := &Handler{
		checks:      checks,
		checksCount: len(checks),
		writeResult: httputil.SendJSON,
	}

	for _, apply := range opts {
		apply(h)
	}

	return h
}

// Handler is the struct containng the HTTP handler function that performs the healthchecks.
type Handler struct {
	checks      []HealthCheck
	checksCount int
	writeResult ResultWriter
}

// ServeHTTP runs the configured health checks in parallel and collects their results.
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
		hc := hc

		go func() {
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
