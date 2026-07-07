package mysqllock

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

				mock.ExpectQuery(sqlReleaseLock).
					WithArgs("key", resLockError).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
			},
			wantErr: false,
		},
		{
			name: "error executing get lock",
			setupMocks: func(mock sqlmock.Sqlmock) {
				// A non-context error must NOT trigger a best-effort release.
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

				mock.ExpectQuery(sqlReleaseLock).
					WillReturnError(errors.New("db error"))
			},
			wantErr:        false,
			wantReleaseErr: true,
		},
		{
			name: "release reports lost lock",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sqlGetLock).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

				// RELEASE_LOCK returns -1 (COALESCE of NULL): the lock was already gone.
				mock.ExpectQuery(sqlReleaseLock).
					WithArgs("key", resLockError).
					WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(resLockError))
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

func TestDB_Acquire_validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		timeout time.Duration
		withDB  bool
		wantErr error
	}{
		{name: "nil db", key: "key", timeout: time.Second, withDB: false, wantErr: ErrNilDB},
		{name: "empty key", key: "", timeout: time.Second, withDB: true, wantErr: ErrInvalidKey},
		{name: "too long key", key: strings.Repeat("a", maxKeyLen+1), timeout: time.Second, withDB: true, wantErr: ErrInvalidKey},
		{name: "zero timeout", key: "key", timeout: 0, withDB: true, wantErr: ErrInvalidTimeout},
		{name: "negative timeout", key: "key", timeout: -time.Second, withDB: true, wantErr: ErrInvalidTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var db *sql.DB

			if tt.withDB {
				mockDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
				require.NoError(t, err)

				defer func() { _ = mockDB.Close() }()

				db = mockDB
			}

			locker := New(db)

			release, err := locker.Acquire(t.Context(), tt.key, tt.timeout)

			require.Nil(t, release)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// Test_Acquire_bestEffortReleaseOnContextError verifies that when GET_LOCK is
// interrupted by context cancellation (the lock may have been granted
// server-side), the acquire path issues a best-effort RELEASE_LOCK so the lock
// does not leak on the pooled connection.
func Test_Acquire_bestEffortReleaseOnContextError(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnError(context.Canceled)
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	locker := New(mockDB)

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second)

	require.Nil(t, release)
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test_Acquire_bestEffortReleaseBounded verifies that the acquire-path
// best-effort release is time-bounded, so a canceled acquisition returns
// promptly instead of blocking on an unresponsive RELEASE_LOCK.
func Test_Acquire_bestEffortReleaseBounded(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnError(context.DeadlineExceeded)
	// The best-effort RELEASE_LOCK stalls; it must be canceled by the bounded
	// timeout rather than run to completion.
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillDelayFor(time.Minute).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	locker := New(mockDB, WithReleaseTimeout(20*time.Millisecond))

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second)

	require.Nil(t, release)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test_Acquire_keepAliveOrdering verifies that the keep-alive goroutine and the
// release function never use the same *sql.Conn concurrently: the release path
// must cancel the keep-alive context and wait for the goroutine to exit before
// running the RELEASE_LOCK query. It also asserts that canceling an in-flight
// keep-alive query at release time is treated as a normal shutdown and does not
// spuriously report a lost lock. Run under -race this would flag the concurrent
// connection use the ordering fix prevents.
func Test_Acquire_keepAliveOrdering(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	// The keep-alive query fires once and blocks, so it is still in flight when
	// release cancels the keep-alive context.
	mock.ExpectExec(keepAliveSQLQuery).
		WillDelayFor(time.Second).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	var handlerErr error

	locker := New(mockDB,
		WithKeepAliveErrorHandler(func(e error) { handlerErr = e }),
		WithKeepAliveInterval(5*time.Millisecond),
	)

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second)
	require.NoError(t, err)
	require.NotNil(t, release)

	// Let the keep-alive ticker fire so its query is in flight before release.
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, release())
	require.NoError(t, handlerErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test_Acquire_lostLockNotification verifies that a keep-alive failure is
// reported to both the instance-wide keep-alive handler and the per-acquisition
// lost-lock handler, each receiving an error wrapping ErrLockLost and naming the
// lock key.
func Test_Acquire_lostLockNotification(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(sqlGetLock).
		WithArgs("mykey", 2.0, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectExec(keepAliveSQLQuery).
		WillReturnError(errors.New("connection reset"))
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("mykey", resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(resLockError))

	instCh := make(chan error, 1)
	lostCh := make(chan error, 1)

	locker := New(mockDB,
		WithKeepAliveErrorHandler(func(e error) { instCh <- e }),
		WithKeepAliveInterval(5*time.Millisecond),
	)

	release, err := locker.Acquire(t.Context(), "mykey", 2*time.Second,
		WithLostLockHandler(func(e error) { lostCh <- e }))
	require.NoError(t, err)

	instErr := waitErr(t, instCh)
	lostErr := waitErr(t, lostCh)

	require.ErrorIs(t, instErr, ErrLockLost)
	require.ErrorIs(t, lostErr, ErrLockLost)
	require.ErrorContains(t, lostErr, "mykey")

	// The lock was already lost, so release surfaces ErrLockLost too.
	require.ErrorIs(t, release(), ErrLockLost)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test_Acquire_handlerPanicRecovered verifies that a panic in one lock-loss
// handler neither crashes the process nor prevents the sibling handler from
// receiving the error.
func Test_Acquire_handlerPanicRecovered(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectExec(keepAliveSQLQuery).
		WillReturnError(errors.New("connection reset"))
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(resLockError))

	lostCh := make(chan error, 1)

	locker := New(mockDB,
		WithKeepAliveErrorHandler(func(error) { panic("handler blew up") }),
		WithKeepAliveInterval(5*time.Millisecond),
	)

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second,
		WithLostLockHandler(func(e error) { lostCh <- e }))
	require.NoError(t, err)

	lostErr := waitErr(t, lostCh)
	require.ErrorIs(t, lostErr, ErrLockLost)

	require.ErrorIs(t, release(), ErrLockLost)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test_Acquire_releaseFromLostLockHandler verifies that calling the release
// function from within the lock-loss handler completes instead of deadlocking:
// the keep-alive goroutine must close done before invoking the handler.
func Test_Acquire_releaseFromLostLockHandler(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectExec(keepAliveSQLQuery).
		WillReturnError(errors.New("connection reset"))
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(resLockError))

	// relCh publishes the release function to the handler with a happens-before
	// edge, avoiding a data race on the release variable.
	relCh := make(chan ReleaseFunc, 1)
	released := make(chan error, 1)

	locker := New(mockDB, WithKeepAliveInterval(5*time.Millisecond))

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second,
		WithLostLockHandler(func(error) {
			released <- (<-relCh)()
		}))
	require.NoError(t, err)

	relCh <- release

	select {
	case relErr := <-released:
		require.ErrorIs(t, relErr, ErrLockLost)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "release() from lost-lock handler deadlocked")
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

// Test_Acquire_releaseTimeout verifies that releasing the lock is bounded by the
// configured release timeout, so a wedged RELEASE_LOCK cannot block forever.
func Test_Acquire_releaseTimeout(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillDelayFor(500 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	locker := New(mockDB, WithReleaseTimeout(20*time.Millisecond))

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second)
	require.NoError(t, err)

	// Without the bounded release context, release would block for the full
	// 500ms delay and then succeed (nil); with it, the RELEASE_LOCK query is
	// canceled and release returns the wrapped error promptly.
	relErr := release()
	require.Error(t, relErr)
	require.ErrorContains(t, relErr, "unable to release mysql lock")
}

// Test_Acquire_releaseIdempotent verifies that calling the release function more
// than once runs RELEASE_LOCK only once and returns the same result.
func Test_Acquire_releaseIdempotent(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	// A single RELEASE_LOCK expectation: a second release must not query again.
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	locker := New(mockDB)

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second)
	require.NoError(t, err)

	require.NoError(t, release())
	require.NoError(t, release())
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test_Acquire_releaseConcurrent verifies that concurrent calls to the release
// function run RELEASE_LOCK exactly once and all observe the same result. Run
// under -race this exercises the sync.Once concurrency contract.
func Test_Acquire_releaseConcurrent(t *testing.T) {
	t.Parallel()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	defer func() { _ = mockDB.Close() }()

	mock.ExpectQuery(sqlGetLock).
		WithArgs("key", 2.0, resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectQuery(sqlReleaseLock).
		WithArgs("key", resLockError).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	locker := New(mockDB)

	release, err := locker.Acquire(t.Context(), "key", 2*time.Second)
	require.NoError(t, err)

	const n = 8

	var wg sync.WaitGroup

	errs := make([]error, n)

	wg.Add(n)

	for i := range errs {
		go func(i int) {
			defer wg.Done()

			errs[i] = release()
		}(i)
	}

	wg.Wait()

	for _, e := range errs {
		require.NoError(t, e)
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func Test_safeInvoke(t *testing.T) {
	t.Parallel()

	// A nil handler is a no-op.
	require.NotPanics(t, func() { safeInvoke(nil, errors.New("x")) })

	// A panicking handler is recovered.
	require.NotPanics(t, func() { safeInvoke(func(error) { panic("boom") }, errors.New("x")) })

	// A normal handler receives the error.
	var got error

	safeInvoke(func(e error) { got = e }, ErrLockLost)
	require.ErrorIs(t, got, ErrLockLost)
}

func Test_makeLostNotifier(t *testing.T) {
	t.Parallel()

	// With no handlers the notifier is a no-op (and does not build an error).
	noop := New(nil).makeLostNotifier("key", nil)

	require.NotPanics(t, func() { noop(errors.New("boom")) })

	// With a handler the wrapped error reaches it, tagged with ErrLockLost.
	var got error

	notify := New(nil).makeLostNotifier("mykey", func(e error) { got = e })
	notify(errors.New("boom"))

	require.ErrorIs(t, got, ErrLockLost)
	require.ErrorContains(t, got, "mykey")
}

type keepConnectionAliveTest struct {
	name        string
	setupMocks  func(mock sqlmock.Sqlmock)
	ctxTimeout  time.Duration
	interval    time.Duration
	pingTimeout time.Duration
	wantHandler bool
}

func Test_keepConnectionAlive(t *testing.T) {
	t.Parallel()

	tests := []keepConnectionAliveTest{
		{
			name: "keep alive query error reported to handler",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(keepAliveSQLQuery).
					WillReturnError(errors.New("can't execute keep alive query at this time"))
			},
			ctxTimeout:  200 * time.Millisecond,
			interval:    20 * time.Millisecond,
			pingTimeout: 200 * time.Millisecond,
			wantHandler: true,
		},
		{
			name: "release cancellation is not reported",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(keepAliveSQLQuery).
					WillDelayFor(time.Second).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			ctxTimeout:  40 * time.Millisecond,
			interval:    20 * time.Millisecond,
			pingTimeout: 500 * time.Millisecond,
			wantHandler: false,
		},
		{
			name: "hung ping times out and is reported",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(keepAliveSQLQuery).
					WillDelayFor(time.Second).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			ctxTimeout:  time.Second,
			interval:    20 * time.Millisecond,
			pingTimeout: 20 * time.Millisecond,
			wantHandler: true,
		},
		{
			name: "successful ping then context done",
			setupMocks: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(keepAliveSQLQuery).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			ctxTimeout:  30 * time.Millisecond,
			interval:    20 * time.Millisecond,
			pingTimeout: 200 * time.Millisecond,
			wantHandler: false,
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

	ctx, cancel := context.WithTimeout(t.Context(), tt.ctxTimeout)
	defer cancel()

	conn, err := mockDB.Conn(ctx)
	require.NoError(t, err)

	var (
		mu          sync.Mutex
		handlerErrs []error
	)

	handler := func(e error) {
		mu.Lock()
		defer mu.Unlock()

		handlerErrs = append(handlerErrs, e)
	}

	done := make(chan struct{})

	keepConnectionAlive(ctx, conn, tt.interval, tt.pingTimeout, handler, done)

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

// waitErr returns the first error delivered on ch, failing the test if none
// arrives promptly.
func waitErr(t *testing.T, ch <-chan error) error {
	t.Helper()

	select {
	case e := <-ch:
		return e
	case <-time.After(2 * time.Second):
		require.FailNow(t, "expected an error but none was delivered")

		return nil
	}
}
