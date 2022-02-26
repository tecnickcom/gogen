// Package sqltransaction helps executing a function inside an SQL transaction.
package sqltransaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

// ExecFunc is the type of the function to be executed inside a SQL Transaction.
type ExecFunc func(ctx context.Context, tx *sql.Tx) error

// Exec executes the specified function inside a SQL transaction.
func Exec(ctx context.Context, db *sql.DB, run ExecFunc) error {
	var committed bool

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to start an SQL transaction: %w", err)
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
		return fmt.Errorf("failed executing a function inside an SQL transaction: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit an SQL transaction: %w", err)
	}

	committed = true

	return nil
}
