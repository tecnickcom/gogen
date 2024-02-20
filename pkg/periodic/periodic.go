// Package periodic allow to execute a specified function periodically.
package periodic

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// TaskFn is the type of function to be periodically executed.
type TaskFn func(context.Context)

// Periodic instance.
type Periodic struct {
	interval   int64         // Time in nanoseconds between two successive calls.
	jitter     int64         // Maximum random Jitter time between each function call.
	timeout    time.Duration // Timeout applied to each function call via context.
	task       TaskFn        // Function to be periodically executed. It should return within the context's timeout.
	timer      *time.Timer
	resetTimer chan time.Duration
	cancel     context.CancelFunc
}

// New creates a new Periodic instance.
// The jitter parameter is the maximum random Jitter time between each function call.
// This is useful to avoid the Thundering herd problem (https://en.wikipedia.org/wiki/Thundering_herd_problem).
func New(interval time.Duration, jitter time.Duration, timeout time.Duration, task TaskFn) (*Periodic, error) {
	intervalNs := int64(interval)
	if intervalNs < 1 {
		return nil, errors.New("interval must be positive")
	}

	jitterNs := int64(jitter)
	if jitterNs < 0 {
		return nil, errors.New("jitter must be positive")
	}

	if int64(timeout) < 1 {
		return nil, errors.New("timeout must be positive")
	}

	if task == nil {
		return nil, errors.New("nil task")
	}

	return &Periodic{
		interval:   intervalNs,
		jitter:     jitterNs,
		timeout:    timeout,
		task:       task,
		resetTimer: make(chan time.Duration, 1),
	}, nil
}

// Start the periodic execution.
func (p *Periodic) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	go p.loop(ctx)
}

// Stop the periodic execution.
// It may block until the last running function returns.
func (p *Periodic) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *Periodic) loop(ctx context.Context) {
	defer p.cancel()

	p.timer = time.NewTimer(1 * time.Nanosecond)

	for {
		select {
		case <-ctx.Done():
			return
		case d := <-p.resetTimer:
			p.setTimer(d)
		case <-p.timer.C:
			p.run(ctx)
		}
	}
}

func (p *Periodic) setTimer(d time.Duration) {
	if !p.timer.Stop() {
		// make sure to drain timer channel before reset
		select {
		case <-p.timer.C:
		default:
		}
	}

	p.timer.Reset(d)
}

func (p *Periodic) run(ctx context.Context) {
	tctx, cancel := context.WithTimeout(ctx, p.timeout)
	p.task(tctx)
	cancel()

	p.resetTimer <- time.Duration(p.interval + rand.Int63n(p.jitter)) //nolint:gosec
}
