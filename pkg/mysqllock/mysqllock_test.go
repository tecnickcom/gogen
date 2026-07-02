package mysqllock

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
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
					WithArgs("key", 2.0, resLockError).
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

			release, err := locker.Acquire(t.Context(), "key", 2*time.Second)

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

// Test_Acquire_keepAliveOrdering verifies that the keep-alive goroutine and the
// release function never use the same *sql.Conn concurrently: the release path
// must cancel the keep-alive context and wait for the goroutine to exit before
// running the RELEASE_LOCK exec. Run under -race this would fail without the fix.
func Test_Acquire_keepAliveOrdering(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 0.05, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	// The keep-alive query may run zero or more times before release; allow it.
	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(keepAliveSQLQuery).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectExec(sqlReleaseLock).
		WithArgs("key").
		WillReturnResult(sqlmock.NewResult(0, 0))

	var handlerErr error

	locker := New(mockDB, WithKeepAliveErrorHandler(func(e error) { handlerErr = e }))

	// Use a tiny interval so the keep-alive ticker fires before release.
	release, err := locker.Acquire(t.Context(), "key", 50*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, release)

	// Give the keep-alive goroutine time to run at least one iteration.
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, release())
	require.NoError(t, handlerErr)
}

func TestWithKeepAliveErrorHandler(t *testing.T) {
	t.Parallel()

	called := false
	locker := New(nil, WithKeepAliveErrorHandler(func(error) { called = true }))

	require.NotNil(t, locker.keepAliveErrHandler)

	locker.keepAliveErrHandler(errors.New("boom"))
	require.True(t, called)
}

type keepConnectionAliveTest struct {
	name        string
	setupMocks  func(mock sqlmock.Sqlmock)
	ctxFunc     func() (context.Context, context.CancelFunc)
	interval    time.Duration
	useHandler  bool
	wantHandler bool
}

func Test_keepConnectionAlive(t *testing.T) {
	t.Parallel()

	const (
		intervalFullTime = 40 * time.Millisecond
		intervalHalfTime = 20 * time.Millisecond
	)

	tests := []keepConnectionAliveTest{
		{
			name: "context done before keep alive query runs",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(t.Context(), intervalHalfTime)
			},
			interval: intervalFullTime,
		},
		{
			name: "error while executing keep alive query reported to handler",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnError(errors.New("can't execute keep alive query at this time"))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(t.Context(), intervalFullTime)
			},
			interval:    intervalHalfTime,
			useHandler:  true,
			wantHandler: true,
		},
		{
			name: "error while executing keep alive query without handler",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnError(errors.New("can't execute keep alive query at this time"))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(t.Context(), intervalFullTime)
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
				return context.WithTimeout(t.Context(), intervalFullTime)
			},
			interval: intervalHalfTime,
		},
		{
			name: "close rows error reported to handler",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(keepAliveSQLQuery).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1).CloseError(errors.New("close error")))
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(t.Context(), intervalFullTime)
			},
			interval:    intervalHalfTime,
			useHandler:  true,
			wantHandler: true,
		},
	}
	for _, tt := range tests {
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

	var (
		mu          sync.Mutex
		handlerErrs []error
		handler     func(error)
	)

	if tt.useHandler {
		handler = func(e error) {
			mu.Lock()
			defer mu.Unlock()

			handlerErrs = append(handlerErrs, e)
		}
	}

	done := make(chan struct{})

	keepConnectionAlive(ctx, conn, tt.interval, handler, done)

	// done must be closed when keepConnectionAlive returns.
	select {
	case <-done:
	default:
		assert.Fail(t, "done channel was not closed on return")
	}

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, tt.wantHandler, len(handlerErrs) > 0, "handler invocation mismatch: %v", handlerErrs)
}

func Test_closeConnection(t *testing.T) {
	t.Parallel()

	mockDB, _, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))

	defer func() { _ = mockDB.Close() }()

	conn, err := mockDB.Conn(t.Context())
	require.NoError(t, err)

	var closeErr error

	closeConnection(conn, &closeErr)

	require.NoError(t, closeErr)

	// A successful close must leave a pre-existing sentinel error untouched so
	// its identity is preserved (e.g. err == ErrTimeout keeps working).
	conn2, err := mockDB.Conn(t.Context())
	require.NoError(t, err)

	sentinelErr := ErrTimeout

	closeConnection(conn2, &sentinelErr)

	require.Equal(t, ErrTimeout, sentinelErr)

	// Close the connection to simulate an error on close.
	_ = conn.Close()

	closeConnection(conn, &closeErr)

	require.Error(t, closeErr)

	// A failing close must join the close error into a pre-existing error, so
	// both errors reach the caller (as in the Acquire error paths).
	joinedErr := ErrTimeout

	closeConnection(conn, &joinedErr)

	require.ErrorIs(t, joinedErr, ErrTimeout)
	require.ErrorIs(t, joinedErr, sql.ErrConnDone)
}
