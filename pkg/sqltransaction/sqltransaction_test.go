package sqltransaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func Test_Exec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(mock sqlmock.Sqlmock)
		run        func(ctx context.Context, tx *sql.Tx) error
		wantErr    bool
	}{
		{
			name: "success",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			run: func(_ context.Context, _ *sql.Tx) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "rollback transaction",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			run: func(_ context.Context, _ *sql.Tx) error {
				return errors.New("db error")
			},
			wantErr: true,
		},
		{
			name: "begin error",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("begin error"))
			},
			run: func(_ context.Context, _ *sql.Tx) error {
				return nil
			},
			wantErr: true,
		},
		{
			name: "commit error",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			run: func(_ context.Context, _ *sql.Tx) error {
				return nil
			},
			wantErr: true,
		},
		{
			name: "rollback error",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback().WillReturnError(errors.New("rollback error"))
			},
			run: func(_ context.Context, _ *sql.Tx) error {
				return errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err)

			defer func() { _ = mockDB.Close() }()

			if tt.setupMocks != nil {
				tt.setupMocks(mock)
			}

			err = Exec(testutil.Context(), mockDB, tt.run)
			require.Equal(t, tt.wantErr, err != nil, "Exec() error = %v, wantErr %v", err, tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

type dbMock struct {
	*sql.DB
	givenOptions *sql.TxOptions
}

func (d *dbMock) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	d.givenOptions = opts
	return d.DB.BeginTx(ctx, opts) //nolint:wrapcheck
}

func Test_ExecWithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options *sql.TxOptions
	}{
		{
			name:    "without options",
			options: nil,
		},
		{
			name: "with READ_COMMITTED isolation level",
			options: &sql.TxOptions{
				Isolation: sql.LevelReadCommitted,
			},
		},
		{
			name: "with ReadOnly",
			options: &sql.TxOptions{
				ReadOnly: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err)

			defer func() { _ = mockDB.Close() }()

			mock.ExpectBegin()
			mock.ExpectCommit()

			db := &dbMock{DB: mockDB}
			err = ExecWithOptions(testutil.Context(), db, func(_ context.Context, _ *sql.Tx) error { return nil }, tt.options)
			require.NoError(t, err)
			require.Equal(t, tt.options, db.givenOptions)
		})
	}
}
