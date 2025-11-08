/*
Package mysqllock provides a distributed locking mechanism that leverages
MySQL's internal functions.

This package allows you to acquire and release locks using MySQL's GET_LOCK and
RELEASE_LOCK functions. It provides a MySQLLock struct that represents a locker
and has methods for acquiring and releasing locks.

Example usage:

	// Create a new MySQLLock instance
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/database")
	if err != nil {
	    log.Fatal(err)
	}
	defer db.Close()
	lock := mysqllock.New(db)

	// Acquire a lock
	releaseFunc, err := lock.Acquire(context.Background(), "my_lock_key", 10*time.Second)
	if err != nil {
	    log.Fatal(err)
	}
	defer releaseFunc()

	// Perform locked operations

	// Release the lock
	err = releaseFunc()
	if err != nil {
	    log.Fatal(err)
	}
*/
package mysqllock

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ReleaseFunc is an alias for a release lock function.
type ReleaseFunc func() error

var (
	// ErrTimeout is an error when the lock is not acquired within the timeout.
	ErrTimeout = errors.New("acquire lock timeout")

	// ErrFailed is an error when the lock is not acquired.
	ErrFailed = errors.New("failed to acquire a lock")
)

// MySQL lock result constants and SQL queries.
const (
	resLockError    = -1
	resLockTimeout  = 0
	resLockAcquired = 1

	sqlGetLock     = "SELECT COALESCE(GET_LOCK(?, ?), ?)"
	sqlReleaseLock = "DO RELEASE_LOCK(?)"

	keepAliveInterval = 30 * time.Second
	keepAliveSQLQuery = "SELECT 1"
)

// MySQLLock represents a locker.
type MySQLLock struct {
	db *sql.DB
}

// New creates a new instance of the locker.
func New(db *sql.DB) *MySQLLock {
	return &MySQLLock{db: db}
}

// Acquire attempts to acquire a database lock.
//
//nolint:contextcheck,nonamedreturns
func (l *MySQLLock) Acquire(ctx context.Context, key string, timeout time.Duration) (rf ReleaseFunc, err error) {
	conn, cerr := l.db.Conn(ctx)
	if cerr != nil {
		return nil, fmt.Errorf("unable to get mysql connection: %w", cerr)
	}

	row := conn.QueryRowContext(ctx, sqlGetLock, key, int(timeout.Seconds()), resLockError)

	var res int

	serr := row.Scan(&res)
	if serr != nil {
		closeConnection(conn, &err)
		return nil, fmt.Errorf("unable to scan mysql lock: %w", serr)
	}

	if res != resLockAcquired {
		closeConnection(conn, &err)

		if res == resLockTimeout {
			return nil, ErrTimeout
		}

		return nil, ErrFailed
	}

	// The release context is independent from the parent context.
	releaseCtx, cancelReleaseCtx := context.WithCancel(context.Background())

	releaseFunc := func() error {
		defer closeConnection(conn, &err)
		defer cancelReleaseCtx()

		_, xerr := conn.ExecContext(releaseCtx, sqlReleaseLock, key)
		if xerr != nil {
			return fmt.Errorf("unable to release mysql lock: %w", xerr)
		}

		return nil
	}

	go keepConnectionAlive(releaseCtx, conn, keepAliveInterval, &err) //nolint:contextcheck

	return releaseFunc, nil
}

// keepConnectionAlive periodically executes a simple query to keep the connection alive.
func keepConnectionAlive(ctx context.Context, conn *sql.Conn, interval time.Duration, err *error) {
	for {
		select {
		case <-time.After(interval):
			//nolint:rowserrcheck
			rows, qerr := conn.QueryContext(ctx, keepAliveSQLQuery)
			if qerr != nil {
				*err = errors.Join(*err, fmt.Errorf("error while keeping mysqllock connection alive: %w", qerr))
				return
			}

			defer func() {
				rerr := rows.Close()
				if rerr != nil {
					*err = errors.Join(*err, fmt.Errorf("unable to close mysql rows: %w", rerr))
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

// closeConnection closes the given SQL connection and logs any error.
func closeConnection(conn *sql.Conn, err *error) {
	cerr := conn.Close()
	if cerr != nil {
		*err = errors.Join(*err, fmt.Errorf("unable to close mysql lock connection: %w", cerr))
	}
}
