/*
Package sqltransaction provides a simple way to execute a function inside an SQL
transaction. The function to be executed is passed as an argument to the Exec
function.

For a similar functionality using the github.com/jmoiron/sqlx package instead of
the standard database/sql one, see the
github.com/tecnickcom/gogen/pkg/sqlxtransaction package.
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
