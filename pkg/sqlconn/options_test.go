// +build unit

package sqlconn

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithConnectFunc(t *testing.T) {
	t.Parallel()

	v := func(ctx context.Context, cfg *config) (db *sql.DB, err error) {
		// mock function
		return nil, nil
	}
	cfg := &config{}
	WithConnectFunc(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.connectFunc).Pointer())
}

func TestWithCheckConnectionFunc(t *testing.T) {
	t.Parallel()

	v := func(ctx context.Context, db *sql.DB) error {
		// mock function
		return nil
	}
	cfg := &config{}
	WithCheckConnectionFunc(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.checkConnectionFunc).Pointer())
}

func TestWithSQLOpenFunc(t *testing.T) {
	t.Parallel()

	v := func(a, b string) (*sql.DB, error) {
		// mock function
		return nil, nil
	}
	cfg := &config{}
	WithSQLOpenFunc(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.sqlOpenFunc).Pointer())
}

func TestWithConnectMaxRetry(t *testing.T) {
	t.Parallel()

	v := 12345
	cfg := &config{}
	WithConnectMaxRetry(v)(cfg)
	require.Equal(t, v, cfg.connectMaxRetry)
}

func TestWithConnectRetryInterval(t *testing.T) {
	t.Parallel()

	v := 99 * time.Second
	cfg := &config{}
	WithConnectRetryInterval(v)(cfg)
	require.Equal(t, v, cfg.connectRetryInterval)
}

func TestWithConnMaxOpen(t *testing.T) {
	t.Parallel()

	v := 12345
	cfg := &config{}
	WithConnMaxOpen(v)(cfg)
	require.Equal(t, v, cfg.connMaxOpen)
}

func TestWithConnMaxIdle(t *testing.T) {
	t.Parallel()

	v := 12345
	cfg := &config{}
	WithConnMaxIdle(v)(cfg)
	require.Equal(t, v, cfg.connMaxIdle)
}

func TestWithConnMaxLifetime(t *testing.T) {
	t.Parallel()

	v := 99 * time.Second
	cfg := &config{}
	WithConnMaxLifetime(v)(cfg)
	require.Equal(t, v, cfg.connMaxLifetime)
}
