/*
Package periodic schedules a task function to run repeatedly at a fixed
interval, with optional random jitter and a per-invocation context timeout.

# Problem

Background workers that poll an external resource, flush a cache, or emit
heartbeats are common in production services. Implementing them correctly
requires handling context cancellation, per-call timeouts, graceful shutdown,
and the thundering-herd problem when many instances restart simultaneously.
Writing this loop from scratch every time is repetitive and error-prone.

# Solution

[New] constructs a [Periodic] scheduler from an interval, a jitter ceiling, a
per-call timeout, and a [TaskFn]. [Periodic.Start] runs the task in a dedicated
goroutine and [Periodic.Stop] shuts it down cleanly:

	p, err := periodic.New(
		30*time.Second,  // run every 30 s
		5*time.Second,   // add up to 5 s of random jitter
		10*time.Second,  // each call gets a 10 s deadline
		myTask,
	)
	if err != nil {
		log.Fatal(err)
	}

	p.Start(ctx)
	defer p.Stop()

# Features

  - Fixed interval with random jitter: the actual pause between calls is
    interval + rand(0, jitter), spreading load across a fleet and avoiding
    the thundering-herd problem
    (https://en.wikipedia.org/wiki/Thundering_herd_problem).
  - Per-call timeout: each [TaskFn] invocation receives a [context.Context]
    derived from the parent with an independent deadline, preventing a single
    slow call from blocking the scheduler.
  - Context-aware shutdown: [Periodic.Start] accepts a parent context;
    canceling it (or calling [Periodic.Stop]) stops the loop after the
    current task invocation returns — no goroutine leaks.
  - Eager first execution: the first call fires after ~1 ns so the task runs
    immediately on start rather than waiting for the first full interval.
  - Simple API: three methods ([New], [Periodic.Start], [Periodic.Stop]) and
    one function type ([TaskFn]) cover the entire surface area.

# Constraints

  - interval must be > 0.
  - jitter must be >= 0 (pass 0 to disable jitter entirely).
  - timeout must be > 0.
  - task must not be nil.

# Benefits

This package eliminates the recurring boilerplate of ticker-based background
loops in Go services, providing built-in jitter, cancellation, and timeout
handling in a minimal, dependency-free implementation.
*/
package periodic

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// TaskFn is the signature of the function executed on each tick.
// The context passed in carries the deadline configured via the timeout
// parameter of [New]. Implementations should respect context cancellation
// and return promptly when ctx.Done() is closed.
type TaskFn func(context.Context)

// Periodic schedules a [TaskFn] to run repeatedly at a configurable interval.
// Create one with [New], start it with [Periodic.Start], and stop it with
// [Periodic.Stop]. The zero value is not usable; always use [New].
type Periodic struct {
	interval   int64         // Time in nanoseconds between two successive calls.
	jitter     int64         // Maximum random Jitter time between each function call.
	timeout    time.Duration // Timeout applied to each function call via context.
	task       TaskFn        // Function to be periodically executed. It should return within the context's timeout.
	timer      *time.Timer
	resetTimer chan time.Duration
	cancel     context.CancelFunc
}

// New creates and validates a new [Periodic] scheduler.
//
// Parameters:
//   - interval: the base duration between successive task invocations; must be > 0.
//   - jitter:   upper bound of a uniformly distributed random delay added to
//     interval after each call; must be >= 0. Pass 0 to disable jitter.
//   - timeout:  deadline applied to each individual task call via a derived
//     [context.Context]; must be > 0.
//   - task:     the [TaskFn] to execute on each tick; must not be nil.
//
// New returns an error if any parameter fails its constraint. Call
// [Periodic.Start] on the returned instance to begin execution.
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

// Start begins the periodic execution loop in a new goroutine.
//
// The first task invocation fires almost immediately (after ~1 ns). Subsequent
// invocations are scheduled at interval + rand(0, jitter) after each call
// completes. The loop exits when ctx is canceled or [Periodic.Stop] is called.
// Start must not be called more than once on the same [Periodic] instance.
func (p *Periodic) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	go p.loop(ctx)
}

// Stop cancels the execution loop.
//
// It signals the goroutine started by [Periodic.Start] to exit and may block
// briefly until the currently running task invocation returns. It is safe to
// call Stop multiple times or before Start.
func (p *Periodic) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

// loop runs the main periodic execution loop.
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

// setTimer sets the timer to the given duration.
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

// run executes the task function with a timeout context and resets the timer.
func (p *Periodic) run(ctx context.Context) {
	tctx, cancel := context.WithTimeout(ctx, p.timeout)
	p.task(tctx)
	cancel()

	p.resetTimer <- time.Duration(p.interval + rand.Int63n(p.jitter)) //nolint:gosec
}
