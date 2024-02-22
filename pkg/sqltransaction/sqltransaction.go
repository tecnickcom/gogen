/*
Package sqltransaction provides a simple way to execute a function inside an SQL transaction.
The function to be executed is passed as an argument to the Exec function.

See also the sqlxtransaction package for the same functionality but using the jmoiron/sqlx package.
*/
package sqltransaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/logging"
	"go.uber.org/zap"
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
func ExecWithOptions(ctx context.Context, db DB, run ExecFunc, opts *sql.TxOptions) error {
	var committed bool

	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("unable to start SQL transaction: %w", err)
	}

	defer func() {
		if committed {
			return
		}

		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			logging.FromContext(ctx).Error("failed rolling back SQL transaction", zap.Error(err))
		}
	}()

	if err = run(ctx, tx); err != nil {
		return fmt.Errorf("failed executing a function inside SQL transaction: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit SQL transaction: %w", err)
	}

	committed = true

	return nil
}
