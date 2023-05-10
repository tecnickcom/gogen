package sqlconn

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultConnMaxIdleCount = 2               // Maximum number of idle connections (0 = unlimited)
	defaultConnMaxIdleTime  = 1 * time.Minute // Maximum amount of time a connection may be idle before being closed
	defaultConnMaxLifetime  = 1 * time.Hour   // Maximum amount of time a connection may be reused (0 = unlimited reuse)
	defaultConnMaxOpenCount = 5               // Maximum number of open connections (0 = unlimited connections)
	defaultPingTimeout      = 5 * time.Second // Healthcheck ping timeout
)

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
	shutdownWaitGroup   *sync.WaitGroup
	shutdownSignalChan  chan struct{}
}

func defaultConfig(driver, dsn string) *config {
	return &config{
		checkConnectionFunc: checkConnection,
		sqlOpenFunc:         sql.Open,
		connectFunc:         connectWithBackoff,
		connMaxIdleCount:    defaultConnMaxIdleCount,
		connMaxIdleTime:     defaultConnMaxIdleTime,
		connMaxLifetime:     defaultConnMaxLifetime,
		connMaxOpenCount:    defaultConnMaxOpenCount,
		driver:              driver,
		dsn:                 dsn,
		pingTimeout:         defaultPingTimeout,
		shutdownWaitGroup:   &sync.WaitGroup{},
		shutdownSignalChan:  make(chan struct{}),
	}
}

//nolint:gocyclo,gocognit
func (c *config) validate() error {
	if strings.TrimSpace(c.driver) == "" {
		return fmt.Errorf("database driver must be set")
	}

	if strings.TrimSpace(c.dsn) == "" {
		return fmt.Errorf("database DSN must be set")
	}

	if c.connectFunc == nil {
		return fmt.Errorf("database connect function must be set")
	}

	if c.checkConnectionFunc == nil {
		return fmt.Errorf("check connection function must be set")
	}

	if c.sqlOpenFunc == nil {
		return fmt.Errorf("sql open function must be set")
	}

	if c.connMaxIdleCount < 1 {
		return fmt.Errorf("database pool max idle connections must be greater than 0")
	}

	if c.connMaxIdleTime < 1*time.Second {
		return fmt.Errorf("database connect retry interval must be at least 1 second")
	}

	if c.connMaxLifetime < 1*time.Second {
		return fmt.Errorf("database connection max lifetime must be at least 1 second")
	}

	if c.connMaxOpenCount < 1 {
		return fmt.Errorf("database pool max open connections must be greater than 0")
	}

	if c.pingTimeout < 1*time.Second {
		return fmt.Errorf("database ping timeout must be at least 1 second")
	}

	if c.shutdownWaitGroup == nil {
		return fmt.Errorf("shutdownWaitGroup is required")
	}

	if c.shutdownSignalChan == nil {
		return fmt.Errorf("shutdownSignalChan is required")
	}

	return nil
}
