package mysqllock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestDB_Acquire(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupMocks     func(mock sqlmock.Sqlmock)
		closeConn      bool
		wantErr        bool
		wantReleaseErr bool
	}{
		{
			name: "success",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sqlGetLock).
					WithArgs("key", 2, resLockError).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

				mock.ExpectExec(sqlReleaseLock).
					WithArgs("key").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "error executing get lock",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sqlGetLock).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "error lock timeout",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sqlGetLock).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(0))
			},
			wantErr: true,
		},
		{
			name: "error lock acquire error",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sqlGetLock).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(2))
			},
			wantErr: true,
		},
		{
			name: "error releasing lock",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sqlGetLock).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

				mock.ExpectExec(sqlReleaseLock).
					WillReturnError(errors.New("db error"))
			},
			wantErr:        false,
			wantReleaseErr: true,
		},
		{
			name:           "error acquiring db connection",
			closeConn:      true,
			wantErr:        true,
			wantReleaseErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err, "AcquireLock() Unexpected error while creating sqlmock", err)

			defer func() { _ = mockDB.Close() }()

			if tt.closeConn {
				_ = mockDB.Close()
			}

			locker := New(mockDB)

			require.NoError(t, err, "failed to create db conn")

			if tt.setupMocks != nil {
				tt.setupMocks(mock)
			}

			release, err := locker.Acquire(testutil.Context(), "key", 2*time.Second)

			var releaseErr error

			if release != nil {
				releaseErr = release()
			}

			require.Equal(t, tt.wantErr, err != nil, "Acquire() error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, tt.wantReleaseErr, releaseErr != nil, "releaseLock() releaseError = %v, wantReleaseErr %v", releaseErr, tt.wantReleaseErr)

			require.NoError(t, mock.ExpectationsWereMet(), "DB expectations not met")
		})
	}
}

type keepConnectionAliveTest struct {
	name       string
	setupMocks func(mock sqlmock.Sqlmock)
	ctxFunc    func() (context.Context, context.CancelFunc)
	interval   time.Duration
}

func Test_keepConnectionAlive(t *testing.T) {
	t.Parallel()

	const (
		intervalFullTime = 20 * time.Millisecond
		intervalHalfTime = 10 * time.Millisecond
	)

	tests := []keepConnectionAliveTest{
		{
			name: "context done while executing keep alive query",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), intervalFullTime)
			},
			interval: intervalFullTime,
		},
		{
			name: "error while executing keep alive query",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnError(errors.New("can't execute keep alive query at this time"))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), intervalFullTime)
			},
			interval: intervalHalfTime,
		},
		{
			name: "successfully keeping the connection alive",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), intervalFullTime)
			},
			interval: intervalHalfTime,
		},
		{
			name: "context done before even getting a change of trying to keep the connection alive",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), intervalFullTime)
			},
			interval: intervalHalfTime,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runKeepConnectionAlive(t, tt)
		})
	}
}

func runKeepConnectionAlive(t *testing.T, tt keepConnectionAliveTest) {
	t.Helper()

	mockDB, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer func() { _ = mockDB.Close() }()

	if tt.setupMocks != nil {
		tt.setupMocks(mock)
	}

	ctx, cancel := tt.ctxFunc()
	defer cancel()

	conn, err := mockDB.Conn(ctx)
	require.NoError(t, err)

	keepConnectionAlive(ctx, conn, tt.interval)
}
