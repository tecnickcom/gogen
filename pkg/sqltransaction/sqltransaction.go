/*
Package sqltransaction solves a common reliability problem in Go database code:
executing business logic inside a transaction with correct begin/commit/rollback
control flow and consistent error handling.

# Problem

Manual transaction handling with database/sql is repetitive and fragile.
Developers frequently duplicate transaction scaffolding and can accidentally
forget rollback on one error path, attempt rollback after commit, or lose
important diagnostic context when multiple failures happen (for example,
operation error plus rollback error).

This package centralizes the transaction lifecycle in one helper so callers can
focus on domain logic.

# How It Works

[Exec] and [ExecWithOptions] execute a caller-provided [ExecFunc] inside a
transaction:

 1. Start transaction via [DB.BeginTx].
 2. Run the provided function with `*sql.Tx`.
 3. Commit on success.
 4. Roll back automatically if the function fails or commit is not reached.

Rollback behavior is guarded to avoid noisy false failures:

  - rollback is skipped after successful commit,
  - `sql.ErrTxDone` during rollback is ignored,
  - rollback failures are joined with the current error for full diagnostics.

# Key Features

  - Small API surface: [Exec] for default settings and [ExecWithOptions] for
    custom [sql.TxOptions] (isolation level, read-only mode, etc.).
  - Interface-driven testability: [DB] allows mocking transaction begin logic
    in unit tests.
  - Strong error context across begin/run/commit/rollback stages.
  - Safe-by-default transaction semantics with deterministic cleanup.

# Usage

	err := sqltransaction.Exec(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
	    // Execute all related SQL operations using tx.
	    // Return an error to trigger rollback.
	    return nil
	})
	if err != nil {
	    return err
	}

For a similar helper using github.com/jmoiron/sqlx instead of database/sql,
see:
github.com/tecnickcom/gogen/pkg/sqlxtransaction
*/
package sqltransaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ExecFunc is the type of the function to be executed inside a SQL Transaction.
type ExecFunc func(ctx context.Context, tx *sql.Tx) error

// DB is the interface which represents the database driver.
type DB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Exec executes the specified function inside a SQL transaction.
func Exec(ctx context.Context, db DB, run ExecFunc) error {
	return ExecWithOptions(ctx, db, run, nil)
}

// ExecWithOptions executes the specified function inside a SQL transaction.
func ExecWithOptions(ctx context.Context, db DB, run ExecFunc, opts *sql.TxOptions) (err error) {
	var committed bool

	tx, berr := db.BeginTx(ctx, opts)
	if berr != nil {
		return fmt.Errorf("unable to start SQL transaction: %w", berr)
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
		return fmt.Errorf("failed executing a function inside SQL transaction: %w", rerr)
	}

	cerr := tx.Commit()
	if cerr != nil {
		return fmt.Errorf("unable to commit SQL transaction: %w", cerr)
	}

	committed = true

	return nil
}
