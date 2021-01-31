package sqlconn

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	defaultConnMaxIdle          = 2                // Maximum number of idle connections (0 = unlimited)
	defaultConnMaxLifetime      = time.Duration(0) // Maximum connection lifetime in seconds (0 = unlimited reuse)
	defaultConnMaxOpen          = 2                // Maximum number of open connections (0 = unlimited connections)
	defaultConnectMaxRetry      = 1                // Number of maximum connection retries
	defaultConnectRetryInterval = 3 * time.Second  // Connection retry time in seconds
)

func defaultConfig(driver, dsn string) *config {
	return &config{
		checkConnectionFunc:  checkConnection,
		connectMaxRetry:      defaultConnectMaxRetry,
		connectRetryInterval: defaultConnectRetryInterval,
		connectFunc:          connectWithBackoff,
		driver:               driver,
		dsn:                  dsn,
		sqlOpenFunc:          sql.Open,
		connMaxIdle:          defaultConnMaxIdle,
		connMaxLifetime:      defaultConnMaxLifetime,
		connMaxOpen:          defaultConnMaxOpen,
	}
}

type config struct {
	checkConnectionFunc  CheckConnectionFunc
	connectMaxRetry      int
	connectRetryInterval time.Duration
	connectFunc          ConnectFunc
	connMaxIdle          int
	connMaxLifetime      time.Duration
	connMaxOpen          int
	driver               string
	dsn                  string
	sqlOpenFunc          SQLOpenFunc
}

func (c *config) validate() error {
	if strings.TrimSpace(c.driver) == "" {
		return fmt.Errorf("database driver must be set")
	}

	if strings.TrimSpace(c.dsn) == "" {
		return fmt.Errorf("database DSN must be set")
	}

	if c.connectMaxRetry < 1 {
		return fmt.Errorf("database connect max retry must be greater than 0")
	}

	if c.connectRetryInterval < 1*time.Second {
		return fmt.Errorf("database connect retry interval must be greater than 1s")
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

	if c.connMaxIdle < 1 {
		return fmt.Errorf("database pool max idle connections must be greater than 0")
	}

	return nil
}
