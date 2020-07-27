// Package sqlconn provides a simple way to manage a database connection
package sqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/nexmoinc/gosrvlib/pkg/healthcheck"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

// ConnectFunc is the function called to perform the actual DB connection
type ConnectFunc func(ctx context.Context, cfg *config) (*sql.DB, error)

// CheckConnectionFunc is the function called to perform a DB connection check
type CheckConnectionFunc func(ctx context.Context, db *sql.DB) error

// SQLOpenFunc is the function called to open the DB. (Only for monkey patch testing)
type SQLOpenFunc func(driverName, dataSourceName string) (*sql.DB, error)

// Connect attempts to connect to a SQL database
func Connect(ctx context.Context, driver, dsn string, opts ...Option) (*SQLConn, error) {
	cfg := defaultConfig(driver, dsn)

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	db, err := cfg.connectFunc(ctx, cfg)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cfg.connMaxIdle)
	db.SetConnMaxLifetime(cfg.connMaxLifetime)
	db.SetMaxOpenConns(cfg.connMaxOpen)

	c := SQLConn{
		cfg: cfg,
		ctx: ctx,
		db:  db,
	}

	// disconnect client when the context is canceled
	go func() {
		<-ctx.Done()
		c.disconnect()
	}()

	return &c, nil
}

// SQLConn is the structure that helps to manage a SQL DB connection
type SQLConn struct {
	cfg    *config
	ctx    context.Context
	db     *sql.DB
	dbLock sync.RWMutex
}

// DB returns a database connection from the pool
func (c *SQLConn) DB() *sql.DB {
	c.dbLock.RLock()
	defer c.dbLock.RUnlock()

	return c.db
}

// HealthCheck performs a health check of the database connection
func (c *SQLConn) HealthCheck(ctx context.Context) healthcheck.Result {
	c.dbLock.RLock()
	defer c.dbLock.RUnlock()

	if c.db == nil {
		return healthcheck.Result{
			Status: healthcheck.Unavailable,
		}
	}

	if err := c.cfg.checkConnectionFunc(ctx, c.db); err != nil {
		return healthcheck.Result{
			Status: healthcheck.Err,
			Error:  err,
		}
	}

	return healthcheck.Result{
		Status: healthcheck.OK,
	}
}

func (c *SQLConn) disconnect() {
	c.dbLock.Lock()
	defer c.dbLock.Unlock()

	if err := c.db.Close(); err != nil {
		logging.FromContext(c.ctx).Error("failed closing database connection", zap.Error(err))
	}
	c.db = nil
}

func checkConnection(ctx context.Context, db *sql.DB) error {
	var err error

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed ping on database: %w", err)
	}

	// nolint:rowserrcheck
	if _, err = db.QueryContext(ctx, "SELECT 1"); err != nil {
		return fmt.Errorf("failed running check query on database: %w", err)
	}

	return nil
}

func connectWithBackoff(ctx context.Context, cfg *config) (*sql.DB, error) {
	var err error

	db, err := cfg.sqlOpenFunc(cfg.driver, cfg.dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening database connection: %w", err)
	}

	if err = cfg.checkConnectionFunc(ctx, db); err != nil {
		return nil, fmt.Errorf("failed checking database connection: %w", err)
	}

	return db, nil
}
