/*
Package mysqllock provides process-distributed mutual exclusion using MySQL's
named lock primitives GET_LOCK and RELEASE_LOCK.

# Problem

When multiple application instances (or background workers) run concurrently,
some operations must execute at most once cluster-wide: scheduled jobs,
idempotent migrations, reconciliation tasks, or cache rebuilds. Coordinating
this with in-memory mutexes is impossible across processes, and introducing a
separate lock service can add operational complexity.

# Solution

This package uses MySQL's built-in named locks to provide a simple distributed
lock API around an existing database dependency. [MySQLLock.Acquire] requests a
lock by key and returns a [ReleaseFunc] that must be called to release it.

Usage:

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	locker := mysqllock.New(db)
	release, err := locker.Acquire(ctx, "daily-reconciliation", 10*time.Second)
	if err != nil {
		if errors.Is(err, mysqllock.ErrTimeout) {
			// Another instance is holding the lock.
			return
		}

		log.Fatal(err)
	}
	defer func() {
		if err := release(); err != nil {
			log.Printf("failed to release lock: %v", err)
		}
	}()

	// Perform the critical section while lock is held.

# Features

  - Single-call acquisition API: [MySQLLock.Acquire] returns a release closure,
    so lock lifetime is easy to scope with defer.
  - Explicit timeout handling: [ErrTimeout] is returned when GET_LOCK does not
    acquire the lock within the requested timeout.
  - Dedicated lock connection: each successful lock acquisition is tied to a
    dedicated SQL connection, matching MySQL's lock semantics.
  - Connection keep-alive: a periodic query keeps the lock-owning connection
    active for long-running critical sections.
  - Context-aware acquisition: caller context controls acquisition cancellation.
  - Zero external dependencies at runtime: relies only on database/sql and
    MySQL lock functions.

# Behavior Notes

The lock key namespace is per MySQL server instance. Use stable, descriptive
keys (for example, "service:job:daily-reconciliation"). Always call the
returned [ReleaseFunc], ideally with defer, to avoid holding locks longer than
intended.
*/
package mysqllock

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ReleaseFunc releases an acquired lock and its associated SQL connection.
//
// It is returned by [MySQLLock.Acquire]. Callers should invoke it exactly once,
// typically using defer immediately after successful acquisition.
type ReleaseFunc func() error

var (
	// ErrTimeout indicates that GET_LOCK did not acquire the lock before timeout.
	ErrTimeout = errors.New("acquire lock timeout")

	// ErrFailed indicates a non-timeout lock acquisition failure.
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

// MySQLLock acquires and releases MySQL named locks through a sql.DB pool.
//
// Create instances with [New].
type MySQLLock struct {
	db *sql.DB
}

// New constructs distributed lock manager using MySQL named locks on provided database connection.
func New(db *sql.DB) *MySQLLock {
	return &MySQLLock{db: db}
}

// Acquire acquires named lock with timeout, returning release function and keep-alive management; returns ErrTimeout if acquisition fails due to timeout.
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
				*err = errors.Join(*err, rows.Close())
			}()

		case <-ctx.Done():
			return
		}
	}
}

// closeConnection closes the given SQL connection and logs any error.
func closeConnection(conn *sql.Conn, err *error) {
	*err = errors.Join(*err, conn.Close())
}
