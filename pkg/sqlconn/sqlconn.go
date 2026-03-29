/*
Package sqlconn solves the boilerplate and lifecycle complexity of managing a
database/sql connection in long-running Go services.

# Problem

Opening a SQL connection in production is more than calling sql.Open: services
must apply pool limits, verify connectivity, expose health checks, and close
the connection gracefully on shutdown signals. When every service reimplements
this flow, behavior drifts and shutdown/health edge cases become hard to reason
about.

sqlconn provides a small, configurable connection manager that standardizes
this pattern.

# How It Works

  - [New] creates a connection from explicit `driver` and `dsn` values.
  - [Connect] accepts a URL-like string in the form `<DRIVER>://<DSN>` and
    delegates to [New]. If only the DSN is provided, a driver can be supplied
    via [WithDefaultDriver].
  - On successful connect, pool settings are applied (`max idle/open`, idle
    time, lifetime), and a goroutine waits for either context cancellation or a
    shutdown signal channel.
  - When shutdown is triggered, [SQLConn.Shutdown] closes the underlying
    database handle, updates the shared shutdown wait group, and prevents
    further use by setting the internal DB pointer to nil.

# Key Features

  - Configurable connection pipeline via options:
    [WithConnectFunc], [WithCheckConnectionFunc], [WithSQLOpenFunc].
  - Pool tuning support:
    [WithConnMaxIdleCount], [WithConnMaxIdleTime], [WithConnMaxLifetime],
    [WithConnMaxOpen].
  - Built-in health check through [SQLConn.HealthCheck], using a ping timeout
    ([WithPingTimeout]) and a basic validation query (`SELECT 1`).
  - Graceful shutdown integration for application lifecycles with
    [WithShutdownSignalChan] and [WithShutdownWaitGroup].
  - Logger integration with [WithLogger] for lifecycle diagnostics.

# Benefits

  - Consistent SQL connection behavior across services.
  - Safer startup/shutdown handling with fewer resource-leak risks.
  - Better testability through injectable open/connect/check functions.
  - Clear integration points for health endpoints and service orchestration.

# Usage

	c, err := sqlconn.Connect(
	    ctx,
	    "mysql://user:pass@tcp(localhost:3306)/appdb",
	    sqlconn.WithDefaultDriver("mysql"),
	)
	if err != nil {
	    return err
	}

	if err := c.HealthCheck(ctx); err != nil {
	    return err
	}

	defer c.Shutdown(ctx)

This package is ideal for Go applications that need a pragmatic, reusable
database/sql connection lifecycle abstraction with health and shutdown support.
*/
package sqlconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// ConnectFunc is the type of function called to perform the actual DB connection.
type ConnectFunc func(ctx context.Context, cfg *config) (*sql.DB, error)

// CheckConnectionFunc is the type of function called to perform a DB connection check.
type CheckConnectionFunc func(ctx context.Context, db *sql.DB) error

// SQLOpenFunc is the type of function called to open the DB. (Only for monkey patch testing).
type SQLOpenFunc func(driverName, dataSourceName string) (*sql.DB, error)

// SQLConn is the structure that helps to manage a SQL DB connection.
type SQLConn struct {
	cfg    *config
	db     *sql.DB
	dbLock sync.RWMutex
}

// connect attempts to connect to a SQL database.
func (cfg *config) connect(ctx context.Context) (*SQLConn, error) {
	db, err := cfg.connectFunc(ctx, cfg)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(cfg.connMaxIdleTime)
	db.SetConnMaxLifetime(cfg.connMaxLifetime)
	db.SetMaxIdleConns(cfg.connMaxIdleCount)
	db.SetMaxOpenConns(cfg.connMaxOpenCount)

	c := SQLConn{
		cfg: cfg,
		db:  db,
	}

	// wait for shutdown signal or context cancelation
	go func() {
		select {
		case <-cfg.shutdownSignalChan:
			cfg.logger.Debug("sqlconn shutdown signal received")
		case <-ctx.Done():
			cfg.logger.Warn("sqlconn context canceled")
		}

		_ = c.Shutdown(ctx)
	}()

	cfg.shutdownWaitGroup.Add(1)

	return &c, nil
}

// New constructs SQL connection with connection pool tuning, health checks, and graceful shutdown orchestration.
func New(ctx context.Context, driver, dsn string, opts ...Option) (*SQLConn, error) {
	cfg, err := newConfig(driver, dsn, opts...)
	if err != nil {
		return nil, err
	}

	return cfg.connect(ctx)
}

// Connect parses URL format "<DRIVER>://<DSN>" and delegates to New for connection creation.
func Connect(ctx context.Context, url string, opts ...Option) (*SQLConn, error) {
	driver, dsn := parseConnectionURL(url)

	return New(ctx, driver, dsn, opts...)
}

// DB returns current database connection from pool; may be nil after Shutdown().
func (c *SQLConn) DB() *sql.DB {
	c.dbLock.RLock()
	defer c.dbLock.RUnlock()

	return c.db
}

// HealthCheck verifies database connectivity with ping and validation query, respecting ping timeout.
func (c *SQLConn) HealthCheck(ctx context.Context) error {
	c.dbLock.RLock()
	defer c.dbLock.RUnlock()

	if c.db == nil {
		return errors.New("database not unavailable")
	}

	ctx, cancel := context.WithTimeout(ctx, c.cfg.pingTimeout)
	defer cancel()

	return c.cfg.checkConnectionFunc(ctx, c.db)
}

// Shutdown gracefully closes database connection, preventing new queries and updating shutdown wait group.
func (c *SQLConn) Shutdown(_ context.Context) error {
	c.cfg.logger.Debug("shutting down sql connection")

	c.dbLock.Lock()
	defer c.dbLock.Unlock()

	err := c.db.Close()

	c.db = nil
	c.cfg.shutdownWaitGroup.Add(-1)

	c.cfg.logger.With(slog.Any("error", err)).Debug("sql connection shutdown complete")

	return err //nolint:wrapcheck
}

// checkConnection performs a simple ping and query to verify the database connection is alive.
func checkConnection(ctx context.Context, db *sql.DB) (err error) {
	perr := db.PingContext(ctx)
	if perr != nil {
		return fmt.Errorf("failed ping on database: %w", perr)
	}

	//nolint:rowserrcheck
	rows, rerr := db.QueryContext(ctx, "SELECT 1")
	if rerr != nil {
		return fmt.Errorf("failed running check query on database: %w", rerr)
	}

	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	return nil
}

// connectWithBackoff attempts to open a database connection and perform a health check.
func connectWithBackoff(ctx context.Context, cfg *config) (*sql.DB, error) {
	db, err := cfg.sqlOpenFunc(cfg.driver, cfg.dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.pingTimeout)
	defer cancel()

	err = cfg.checkConnectionFunc(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed checking database connection: %w", err)
	}

	return db, nil
}

// parseConnectionURL attempts to extract the driver/dsn pair from a string in the format <DRIVER>://<DSN>
// if only the DSN part is set, the driver will need to be specified via a configuration option.
// Examples:
// - mysql://user:pass@tcp(host:3306)/database
// - pgx://postgres://user:pass@host:5432/database?sslmode=disable
func parseConnectionURL(url string) (string, string) {
	if strings.TrimSpace(url) == "" {
		return "", ""
	}

	const sep = "://"

	before, after, found := strings.Cut(url, sep)

	if found {
		return before, after
	}

	return "", before
}
