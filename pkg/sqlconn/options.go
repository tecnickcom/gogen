package sqlconn

import (
	"log/slog"
	"sync"
	"time"
)

// Option configures SQL connection behavior.
type Option func(*config)

// WithConnectFunc replaces default connection function (e.g., for testing).
func WithConnectFunc(fn ConnectFunc) Option {
	return func(cfg *config) {
		cfg.connectFunc = fn
	}
}

// WithCheckConnectionFunc replaces default connection verification function.
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

// WithConnMaxIdleCount sets maximum number of idle database connections in pool.
func WithConnMaxIdleCount(maxIdle int) Option {
	return func(cfg *config) {
		cfg.connMaxIdleCount = maxIdle
	}
}

// WithConnMaxIdleTime sets maximum idle time before connection is reconnected.
func WithConnMaxIdleTime(t time.Duration) Option {
	return func(cfg *config) {
		cfg.connMaxIdleTime = t
	}
}

// WithConnMaxLifetime sets maximum lifetime of a connection before it must be closed.
func WithConnMaxLifetime(t time.Duration) Option {
	return func(cfg *config) {
		cfg.connMaxLifetime = t
	}
}

// WithConnMaxOpen sets maximum number of open connections in the pool.
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
