/*
Package sqlxtransaction solves a common reliability problem in Go database
code: correctly handling begin/commit/rollback control flow around business
logic executed inside a sqlx transaction.

# Problem

Manual transaction handling is repetitive and easy to get wrong. Typical bugs
include forgetting rollback on error paths, attempting double rollback,
committing after a failed operation, or returning partial error context. These
issues can silently leak inconsistent state and make production incidents harder
to debug.

This package provides a small execution wrapper that centralizes the
transaction lifecycle and enforces a safe default pattern.

# How It Works

[Exec] and [ExecWithOptions] accept a caller-provided [ExecFunc] and run it
inside a sqlx transaction:

 1. Start a transaction with [DB.BeginTxx].
 2. Execute the provided function with the transaction object.
 3. Commit if execution succeeds.
 4. Roll back automatically if execution fails or commit is not reached.

Rollback behavior is deferred and guarded:

  - rollback is skipped after a successful commit,
  - `sql.ErrTxDone` is ignored during deferred rollback,
  - rollback failures are joined with the current error so diagnostics are not
    lost.

# Key Features

  - Minimal API: call [Exec] for default transaction options, or
    [ExecWithOptions] for custom [sql.TxOptions] (isolation level, read-only,
    etc.).
  - Testability by interface: the [DB] interface abstracts `BeginTxx`, making
    the transaction entrypoint easy to mock.
  - Strong error context: begin, run, commit, and rollback failures are
    wrapped with actionable messages.
  - Safe-by-default lifecycle: commit happens only on successful function
    completion; otherwise rollback is guaranteed.

# Usage

	err := sqlxtransaction.Exec(ctx, db, func(ctx context.Context, tx *sqlx.Tx) error {
	    // Execute all related SQL operations using tx.
	    // Return an error to trigger rollback.
	    return nil
	})
	if err != nil {
	    return err
	}

For a similar helper based on the standard database/sql package (instead of
github.com/jmoiron/sqlx), see:
github.com/tecnickcom/gogen/pkg/sqltransaction
*/
package sqlxtransaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ExecFunc is the type of the function to be executed inside a SQL Transaction.
type ExecFunc func(ctx context.Context, tx *sqlx.Tx) error

// DB is the interface which represents the database driver.
type DB interface {
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

// Exec executes the specified function inside a SQL transaction.
func Exec(ctx context.Context, db DB, run ExecFunc) error {
	return ExecWithOptions(ctx, db, run, nil)
}

// ExecWithOptions executes the specified function inside a SQL transaction.
func ExecWithOptions(ctx context.Context, db DB, run ExecFunc, opts *sql.TxOptions) (err error) {
	var committed bool

	tx, berr := db.BeginTxx(ctx, opts)
	if berr != nil {
		return fmt.Errorf("unable to start SQLX transaction: %w", berr)
	}

	defer func() {
		if committed {
			return
		}

		kerr := tx.Rollback()
		if kerr != nil && !errors.Is(kerr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("failed rolling back SQL transaction: %w", kerr))
		}
	}()

	rerr := run(ctx, tx)
	if rerr != nil {
		return fmt.Errorf("failed executing a function inside SQLX transaction: %w", rerr)
	}

	cerr := tx.Commit()
	if cerr != nil {
		return fmt.Errorf("unable to commit SQL transaction: %w", cerr)
	}

	committed = true

	return nil
}
