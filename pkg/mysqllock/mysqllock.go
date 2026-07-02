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
  - Observable keep-alive: keep-alive failures can be surfaced through an
    optional handler configured with [WithKeepAliveErrorHandler].
  - Context-aware acquisition: caller context controls acquisition cancellation.
  - Zero external dependencies at runtime: relies only on database/sql and
    MySQL lock functions.

# Behavior Notes

The lock key namespace is per MySQL server instance. Use stable, descriptive
keys (for example, "service:job:daily-reconciliation"). Always call the
returned [ReleaseFunc], ideally with defer, to avoid holding locks longer than
intended.

The GET_LOCK timeout is passed to MySQL with sub-second (fractional) precision;
MySQL 5.7.5 and later accept fractional timeouts (older servers truncate to
whole seconds).
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

	// keepAliveErrHandler, when set, is invoked with any error produced by the
	// keep-alive goroutine while a lock is held.
	keepAliveErrHandler func(error)
}

// Option configures a [MySQLLock] at construction time.
//
// Options are additive; passing none preserves the historical default behavior.
type Option func(*MySQLLock)

// WithKeepAliveErrorHandler sets a handler that receives errors produced by the
// keep-alive goroutine (failures of the periodic keep-alive query or of closing
// its result set) while a lock is held.
//
// The handler is called from the keep-alive goroutine, so it must be safe for
// concurrent use and should not block for long. A nil handler is ignored.
func WithKeepAliveErrorHandler(handler func(error)) Option {
	return func(l *MySQLLock) {
		l.keepAliveErrHandler = handler
	}
}

// New constructs distributed lock manager using MySQL named locks on provided database connection.
func New(db *sql.DB, opts ...Option) *MySQLLock {
	l := &MySQLLock{db: db}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Acquire acquires named lock with timeout, returning release function and keep-alive management; returns ErrTimeout if acquisition fails due to timeout.
//
//nolint:contextcheck,nonamedreturns
func (l *MySQLLock) Acquire(ctx context.Context, key string, timeout time.Duration) (rf ReleaseFunc, err error) {
	conn, cerr := l.db.Conn(ctx)
	if cerr != nil {
		return nil, fmt.Errorf("unable to get mysql connection: %w", cerr)
	}

	row := conn.QueryRowContext(ctx, sqlGetLock, key, timeout.Seconds(), resLockError)

	var res int

	serr := row.Scan(&res)
	if serr != nil {
		err = fmt.Errorf("unable to scan mysql lock: %w", serr)
		closeConnection(conn, &err)

		return nil, err
	}

	if res != resLockAcquired {
		if res == resLockTimeout {
			err = ErrTimeout
		} else {
			err = ErrFailed
		}

		closeConnection(conn, &err)

		return nil, err
	}

	// The release context is independent from the parent context.
	releaseCtx, cancelReleaseCtx := context.WithCancel(context.Background())

	// done is closed by the keep-alive goroutine right before it returns, so the
	// release function can wait for it to stop touching conn before using conn.
	done := make(chan struct{})

	releaseFunc := func() error {
		// Stop the keep-alive goroutine and wait for it to exit BEFORE using the
		// connection, otherwise QueryContext and ExecContext could run on the same
		// *sql.Conn concurrently (a database/sql concurrent-use violation).
		cancelReleaseCtx()
		<-done

		var rerr error

		defer closeConnection(conn, &rerr)

		_, xerr := conn.ExecContext(context.Background(), sqlReleaseLock, key)
		if xerr != nil {
			rerr = errors.Join(rerr, fmt.Errorf("unable to release mysql lock: %w", xerr))
		}

		return rerr
	}

	go keepConnectionAlive(releaseCtx, conn, keepAliveInterval, l.keepAliveErrHandler, done)

	return releaseFunc, nil
}

// keepConnectionAlive periodically executes a simple query to keep the connection
// alive until ctx is canceled. It closes done right before returning so callers
// can wait for it to stop using conn. Any error is reported to errHandler when set.
func keepConnectionAlive(ctx context.Context, conn *sql.Conn, interval time.Duration, errHandler func(error), done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			kerr := pingConnection(ctx, conn)
			if kerr != nil {
				reportKeepAliveError(errHandler, kerr)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// pingConnection runs the keep-alive query once, closing its result set
// immediately (no rows are iterated, so the set is opened only to keep the
// connection active).
func pingConnection(ctx context.Context, conn *sql.Conn) (err error) {
	//nolint:rowserrcheck // the rows are not iterated; the query only refreshes the connection.
	rows, qerr := conn.QueryContext(ctx, keepAliveSQLQuery)
	if qerr != nil {
		return fmt.Errorf("error while keeping mysqllock connection alive: %w", qerr)
	}

	defer func() {
		cerr := rows.Close()
		if cerr != nil {
			err = errors.Join(err, fmt.Errorf("error while closing mysqllock keep-alive result set: %w", cerr))
		}
	}()

	return nil
}

// reportKeepAliveError forwards a keep-alive error to the handler when one is set.
func reportKeepAliveError(errHandler func(error), err error) {
	if errHandler != nil {
		errHandler(err)
	}
}

// closeConnection closes the given SQL connection, joining any close error
// into err. When the close succeeds, err is left untouched so sentinel errors
// (e.g. [ErrTimeout]) keep their identity.
func closeConnection(conn *sql.Conn, err *error) {
	cerr := conn.Close()
	if cerr != nil {
		*err = errors.Join(*err, cerr)
	}
}
