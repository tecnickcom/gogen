package healthcheck

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/httputil/jsendx"
)

const (
	checkTimeout = 1 * time.Second
)

// HealthCheckerMap is a type alias for a map of healthchecker instances
type HealthCheckerMap map[string]HealthChecker

// Handler returns an HTTP handler function performing the healthcheck
// This is a basic fanout implementation, it could be smarter and run a background collection process
// independent from how many times we call the status endpoint
func Handler(checks HealthCheckerMap, appInfo *jsendx.AppInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checkCount := len(checks)

		var wg sync.WaitGroup
		wg.Add(checkCount)

		type resultWrap struct {
			id     string
			result Result
		}
		resCh := make(chan resultWrap, checkCount)
		defer close(resCh)

		for id, hc := range checks {
			go func(id string, hc HealthChecker) {
				defer wg.Done()
				resCh <- resultWrap{
					id:     id,
					result: runCheckWithTimeout(r.Context(), hc, checkTimeout),
				}
			}(id, hc)
		}

		wg.Wait()

		data := make(map[string]Result, checkCount)
		for i := 0; i < checkCount; i++ {
			r := <-resCh
			data[r.id] = r.result
		}

		if appInfo != nil {
			jsendx.Send(r.Context(), w, http.StatusOK, appInfo, data)
			return
		}
		httputil.SendJSON(r.Context(), w, http.StatusOK, data)
	}
}

func runCheckWithTimeout(ctx context.Context, c HealthChecker, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resCh := make(chan Result)
	go func() {
		resCh <- c.HealthCheck(ctx)
	}()

	select {
	case <-ctx.Done():
		return Result{
			Status: Err,
			Error:  ctx.Err(),
		}
	case r := <-resCh:
		return r
	}
}
