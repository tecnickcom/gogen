package sqlconn

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Option configures SQL connection behavior.
type Option func(*config)

// WithConnectFunc replaces default connection function (e.g., for testing).
// The connection manager applies the pool settings (max idle/open, idle time,
// lifetime) to the returned handle after this function returns, so those limits
// take precedence over any pool tuning the custom function applied itself.
func WithConnectFunc(fn ConnectFunc) Option {
	return func(cfg *config) {
		cfg.connectFunc = fn
	}
}

// WithCheckConnectionFunc replaces default connection verification function.
// To change only the validation query of the built-in check (rather than replace
// the whole check), use [WithValidationQuery] instead.
func WithCheckConnectionFunc(fn CheckConnectionFunc) Option {
	return func(cfg *config) {
		cfg.checkConnectionFunc = fn
	}
}

// WithSQLOpenFunc replaces default sql.Open wrapper (for testing only).
func WithSQLOpenFunc(fn SQLOpenFunc) Option {
	return func(cfg *config) {
		cfg.sqlOpenFunc = fn
	}
}

// WithConnMaxIdleCount sets the maximum number of idle database connections in
// the pool. Zero disables the idle connection pool. When greater than the max
// open count, database/sql silently caps it to the max open count.
func WithConnMaxIdleCount(maxIdle int) Option {
	return func(cfg *config) {
		cfg.connMaxIdleCount = maxIdle
	}
}

// WithConnMaxIdleTime sets the maximum time a connection may remain idle before
// being closed. Zero means connections are never closed due to idle time.
func WithConnMaxIdleTime(t time.Duration) Option {
	return func(cfg *config) {
		cfg.connMaxIdleTime = t
	}
}

// WithConnMaxLifetime sets the maximum time a connection may be reused before
// being closed. Zero means connections are never closed due to their age.
func WithConnMaxLifetime(t time.Duration) Option {
	return func(cfg *config) {
		cfg.connMaxLifetime = t
	}
}

// WithConnMaxOpen sets the maximum number of open connections in the pool.
// Zero means there is no limit on the number of open connections.
func WithConnMaxOpen(maxOpen int) Option {
	return func(cfg *config) {
		cfg.connMaxOpenCount = maxOpen
	}
}

// WithDefaultDriver sets fallback driver if not included in DSN.
func WithDefaultDriver(driver string) Option {
	return func(cfg *config) {
		if cfg.driver == "" {
			cfg.driver = driver
		}
	}
}

// WithPingTimeout sets context timeout for health check ping operations.
func WithPingTimeout(t time.Duration) Option {
	return func(cfg *config) {
		cfg.pingTimeout = t
	}
}

// WithValidationQuery overrides the query the built-in health check runs after
// the ping (default "SELECT 1"). Use it for engines where "SELECT 1" is invalid,
// e.g. Oracle requires "SELECT 1 FROM DUAL". The query must return at least one
// row with a single scannable column. It has no effect when the whole check is
// replaced via [WithCheckConnectionFunc].
func WithValidationQuery(query string) Option {
	return func(cfg *config) {
		cfg.validationQuery = query
	}
}

// WithLogger overrides default logger for connection lifecycle events.
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *config) {
		cfg.logger = logger
	}
}

// WithShutdownWaitGroup sets external wait group to signal when connection closes.
func WithShutdownWaitGroup(wg *sync.WaitGroup) Option {
	return func(cfg *config) {
		cfg.shutdownWaitGroup = wg
	}
}

// WithShutdownSignalChan sets channel to trigger graceful shutdown of connection.
func WithShutdownSignalChan(ch chan struct{}) Option {
	return func(cfg *config) {
		cfg.shutdownSignalChan = ch
	}
}

// WithLifetimeContext sets the context whose cancellation triggers a graceful
// shutdown of the connection. It is independent from the context passed to
// New/Connect, which bounds only connection establishment (dialing and the
// initial health check). When unset, the connection lifetime is governed solely
// by the shutdown signal channel (WithShutdownSignalChan) and explicit Shutdown
// calls, and a short-lived establishment context cannot tear the pool down.
func WithLifetimeContext(ctx context.Context) Option {
	return func(cfg *config) {
		cfg.lifetimeCtx = ctx
	}
}
