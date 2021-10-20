package sqlconn

import (
	"time"
)

// Option is a type alias for a function that configures the DB connector.
type Option func(*config)

// WithConnectFunc replaces the default connection function.
func WithConnectFunc(fn ConnectFunc) Option {
	return func(cfg *config) {
		cfg.connectFunc = fn
	}
}

// WithCheckConnectionFunc replaces the default connection check function.
func WithCheckConnectionFunc(fn CheckConnectionFunc) Option {
	return func(cfg *config) {
		cfg.checkConnectionFunc = fn
	}
}

// WithSQLOpenFunc replaces the default open database function.
func WithSQLOpenFunc(fn SQLOpenFunc) Option {
	return func(cfg *config) {
		cfg.sqlOpenFunc = fn
	}
}

// WithConnMaxIdleCount sets the maximum number of idle database connections.
func WithConnMaxIdleCount(maxIdle int) Option {
	return func(cfg *config) {
		cfg.connMaxIdleCount = maxIdle
	}
}

// WithConnMaxIdleTime sets the maximum idle time of a database connection.
func WithConnMaxIdleTime(t time.Duration) Option {
	return func(cfg *config) {
		cfg.connMaxIdleTime = t
	}
}

// WithConnMaxLifetime sets the maximum lifetime of a database connection.
func WithConnMaxLifetime(t time.Duration) Option {
	return func(cfg *config) {
		cfg.connMaxLifetime = t
	}
}

// WithConnMaxOpen sets the maximum number of open database connections.
func WithConnMaxOpen(maxOpen int) Option {
	return func(cfg *config) {
		cfg.connMaxOpenCount = maxOpen
	}
}

// WithDefaultDriver sets the default driver to use if not included in the DSN.
func WithDefaultDriver(driver string) Option {
	return func(cfg *config) {
		if cfg.driver == "" {
			cfg.driver = driver
		}
	}
}

// WithPingTimeout sets the healthcheck ping timeout.
func WithPingTimeout(t time.Duration) Option {
	return func(cfg *config) {
		cfg.pingTimeout = t
	}
}
