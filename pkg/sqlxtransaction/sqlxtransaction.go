/*
Package sqlxtransaction provides a simple way to execute a function inside an
SQLX transaction. The function to be executed is passed as an argument to the
Exec function.

For a similar functionality using the standard database/sql package instead of
the github.com/jmoiron/sqlx one, see the
github.com/tecnickcom/gogen/pkg/sqltransaction package.
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
