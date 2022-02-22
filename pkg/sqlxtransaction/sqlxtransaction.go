// Package sqlxtransaction helps executing a function inside an SQLX transaction.
package sqlxtransaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

// ExecFunc is the type of the function to be executed inside a SQL Transaction.
type ExecFunc func(ctx context.Context, tx *sqlx.Tx) error

// Exec executes the specified function inside a SQL transaction.
func Exec(ctx context.Context, db *sqlx.DB, run ExecFunc) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to start an SQLX transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			logging.FromContext(ctx).Error("failed rolling back SQLX transaction", zap.Error(err))
		}
	}()

	if err = run(ctx, tx); err != nil {
		return fmt.Errorf("failed executing a function inside an SQLX transaction: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit an SQL transaction: %w", err)
	}

	return nil
}
