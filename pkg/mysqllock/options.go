package mysqllock

import "time"

// Option configures a [MySQLLock] at construction time.
//
// Options are additive; passing none preserves the default behavior.
type Option func(*MySQLLock)

// WithKeepAliveErrorHandler sets a handler that receives errors produced by the
// keep-alive goroutine (failures of the periodic keep-alive query) while a lock
// is held. The error wraps [ErrLockLost] and names the affected lock key.
//
// The handler is called from the keep-alive goroutine, so it must be safe for
// concurrent use and should not block for long. A nil handler is ignored, and a
// panic in the handler is recovered rather than crashing the process. For
// aborting a specific critical section prefer the per-acquisition
// [WithLostLockHandler].
func WithKeepAliveErrorHandler(handler func(error)) Option {
	return func(l *MySQLLock) {
		l.keepAliveErrHandler = handler
	}
}

// WithKeepAliveInterval sets the period between keep-alive queries that keep the
// lock-owning connection active. Use a value shorter than the shortest idle
// timeout in the connection path (MySQL wait_timeout or any intervening proxy).
// A non-positive interval is ignored and the default (30s) is kept.
func WithKeepAliveInterval(interval time.Duration) Option {
	return func(l *MySQLLock) {
		if interval > 0 {
			l.keepAliveInterval = interval
		}
	}
}

// WithKeepAlivePingTimeout bounds how long a single keep-alive query may run
// before it is treated as a keep-alive failure (and thus a lost lock). This lets
// a hung connection be detected instead of stalling the keep-alive goroutine
// indefinitely. Choose a value smaller than the keep-alive interval. A
// non-positive timeout is ignored and the default (10s) is kept.
func WithKeepAlivePingTimeout(timeout time.Duration) Option {
	return func(l *MySQLLock) {
		if timeout > 0 {
			l.keepAlivePingTimeout = timeout
		}
	}
}

// WithReleaseTimeout bounds how long the RELEASE_LOCK query invoked by the
// [ReleaseFunc] may run, so a wedged connection cannot block the caller
// indefinitely. A non-positive timeout is ignored and the default (10s) is kept.
func WithReleaseTimeout(timeout time.Duration) Option {
	return func(l *MySQLLock) {
		if timeout > 0 {
			l.releaseTimeout = timeout
		}
	}
}

// AcquireOption configures a single [MySQLLock.Acquire] call.
type AcquireOption func(*acquireConfig)

// WithLostLockHandler sets a handler invoked when this specific lock is lost
// while held (the keep-alive query failed). The handler receives an error
// wrapping [ErrLockLost] and naming the lock key, giving the caller a chance to
// abort the critical section.
//
// The handler is called from the keep-alive goroutine, so it must be safe for
// concurrent use and should not block for long. A nil handler is ignored, and a
// panic in the handler is recovered rather than crashing the process. The
// handler may call the [ReleaseFunc] returned by [MySQLLock.Acquire] without
// deadlocking, though the usual pattern is to signal the critical section to
// stop and let its own deferred release run.
func WithLostLockHandler(handler func(error)) AcquireOption {
	return func(c *acquireConfig) {
		c.onLost = handler
	}
}
