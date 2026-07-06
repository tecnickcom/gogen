package sqlconn

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Exported sentinel errors returned by configuration validation and health
// checks so callers can match them with errors.Is.
var (
	// ErrDriverRequired is returned when no database driver is configured.
	ErrDriverRequired = errors.New("database driver must be set")

	// ErrDSNRequired is returned when no database DSN is configured.
	ErrDSNRequired = errors.New("database DSN must be set")

	// ErrNilConnectFunc is returned when the connect function is nil.
	ErrNilConnectFunc = errors.New("database connect function must be set")

	// ErrNilCheckConnectionFunc is returned when the check connection function is nil.
	ErrNilCheckConnectionFunc = errors.New("check connection function must be set")

	// ErrNilSQLOpenFunc is returned when the sql open function is nil.
	ErrNilSQLOpenFunc = errors.New("sql open function must be set")

	// ErrInvalidMaxIdleCount is returned when the pool max idle connection count is negative.
	ErrInvalidMaxIdleCount = errors.New("database pool max idle connections must not be negative")

	// ErrInvalidMaxIdleTime is returned when the connection max idle time is negative.
	ErrInvalidMaxIdleTime = errors.New("database connection max idle time must not be negative")

	// ErrInvalidMaxLifetime is returned when the connection max lifetime is negative.
	ErrInvalidMaxLifetime = errors.New("database connection max lifetime must not be negative")

	// ErrInvalidMaxOpenCount is returned when the pool max open connection count is negative.
	ErrInvalidMaxOpenCount = errors.New("database pool max open connections must not be negative")

	// ErrInvalidPingTimeout is returned when the ping timeout is below one second.
	ErrInvalidPingTimeout = errors.New("database ping timeout must be at least 1 second")

	// ErrNilLogger is returned when the logger is nil.
	ErrNilLogger = errors.New("logger is required")

	// ErrNilShutdownWaitGroup is returned when the shutdown wait group is nil.
	ErrNilShutdownWaitGroup = errors.New("shutdownWaitGroup is required")

	// ErrNilShutdownSignalChan is returned when the shutdown signal channel is nil.
	ErrNilShutdownSignalChan = errors.New("shutdownSignalChan is required")

	// ErrEmptyValidationQuery is returned when the health-check validation query is blank.
	ErrEmptyValidationQuery = errors.New("validation query must not be empty")
)

// Default configuration values.
const (
	defaultConnMaxIdleCount = 2               // Maximum number of idle connections (0 disables the idle pool)
	defaultConnMaxIdleTime  = 1 * time.Minute // Maximum time a connection may be idle before being closed (0 = no limit)
	defaultConnMaxLifetime  = 1 * time.Hour   // Maximum time a connection may be reused before being closed (0 = no limit)
	defaultConnMaxOpenCount = 5               // Maximum number of open connections (0 = unlimited)
	defaultPingTimeout      = 5 * time.Second // Healthcheck ping timeout (must be at least 1 second)
	defaultValidationQuery  = "SELECT 1"      // Healthcheck validation query (override per engine, e.g. Oracle needs FROM DUAL)
)

// config holds configuration settings for the SQL connection.
type config struct {
	checkConnectionFunc CheckConnectionFunc
	sqlOpenFunc         SQLOpenFunc
	connectFunc         ConnectFunc
	connMaxIdleTime     time.Duration
	connMaxLifetime     time.Duration
	connMaxIdleCount    int
	connMaxOpenCount    int
	driver              string
	dsn                 string
	pingTimeout         time.Duration
	validationQuery     string
	logger              *slog.Logger
	shutdownWaitGroup   *sync.WaitGroup
	shutdownSignalChan  chan struct{}
	lifetimeCtx         context.Context //nolint:containedctx
}

// defaultConfig returns a config struct initialized with default values.
func defaultConfig(driver, dsn string) *config {
	cfg := &config{
		sqlOpenFunc:        sql.Open,
		connectFunc:        connectOnce,
		connMaxIdleCount:   defaultConnMaxIdleCount,
		connMaxIdleTime:    defaultConnMaxIdleTime,
		connMaxLifetime:    defaultConnMaxLifetime,
		connMaxOpenCount:   defaultConnMaxOpenCount,
		driver:             driver,
		dsn:                dsn,
		pingTimeout:        defaultPingTimeout,
		validationQuery:    defaultValidationQuery,
		logger:             slog.Default(),
		shutdownWaitGroup:  &sync.WaitGroup{},
		shutdownSignalChan: make(chan struct{}),
	}

	// Bind the default check as a method value so it reads validationQuery at call
	// time, honoring a later WithValidationQuery on the same config.
	cfg.checkConnectionFunc = cfg.checkConnection

	return cfg
}

// newConfig builds and validates a config from defaults plus options.
func newConfig(driver, dsn string, opts ...Option) (*config, error) {
	cfg := defaultConfig(driver, dsn)

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	// Trim once so stray whitespace (e.g. a trailing newline from a secret file
	// or env var) never reaches sql.Open and fails with an opaque driver error.
	cfg.driver = strings.TrimSpace(cfg.driver)
	cfg.dsn = strings.TrimSpace(cfg.dsn)

	err := cfg.validate()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks the configuration for required fields and valid values.
//
//nolint:gocyclo,cyclop,gocognit
func (c *config) validate() error {
	if strings.TrimSpace(c.driver) == "" {
		return ErrDriverRequired
	}

	if strings.TrimSpace(c.dsn) == "" {
		return ErrDSNRequired
	}

	if c.connectFunc == nil {
		return ErrNilConnectFunc
	}

	if c.checkConnectionFunc == nil {
		return ErrNilCheckConnectionFunc
	}

	if c.sqlOpenFunc == nil {
		return ErrNilSQLOpenFunc
	}

	// Pool limits mirror database/sql semantics: a negative value is invalid,
	// while zero is a legitimate configuration (disabled idle pool, no idle/age
	// expiry, or unlimited open connections respectively).
	if c.connMaxIdleCount < 0 {
		return ErrInvalidMaxIdleCount
	}

	if c.connMaxIdleTime < 0 {
		return ErrInvalidMaxIdleTime
	}

	if c.connMaxLifetime < 0 {
		return ErrInvalidMaxLifetime
	}

	if c.connMaxOpenCount < 0 {
		return ErrInvalidMaxOpenCount
	}

	if c.pingTimeout < 1*time.Second {
		return ErrInvalidPingTimeout
	}

	if strings.TrimSpace(c.validationQuery) == "" {
		return ErrEmptyValidationQuery
	}

	if c.logger == nil {
		return ErrNilLogger
	}

	if c.shutdownWaitGroup == nil {
		return ErrNilShutdownWaitGroup
	}

	if c.shutdownSignalChan == nil {
		return ErrNilShutdownSignalChan
	}

	return nil
}
