package sqlconn

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockConnectFunc(db *sql.DB, err error) ConnectFunc {
	return func(_ context.Context, _ *config) (*sql.DB, error) {
		return db, err
	}
}

//nolint:gocognit
func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		driver         string
		dsn            string
		connectErr     error
		nilDB          bool
		configMockFunc func(sqlmock.Sqlmock)
		wantConn       bool
		shutdownSig    bool
		wantErr        bool
		wantErrIs      error
	}{
		{
			name:      "fail with config validation error",
			driver:    "",
			dsn:       "",
			wantErr:   true,
			wantErrIs: ErrDriverRequired,
		},
		{
			name:       "fail to open DB connection",
			driver:     "testsql",
			dsn:        "user:pass@tcp(db.host.invalid:1234)/testdb",
			connectErr: errors.New("db open error"),
			wantErr:    true,
		},
		{
			name:      "fail with nil DB from connect function",
			driver:    "testsql",
			dsn:       "user:pass@tcp(db.host.invalid:1234)/testdb",
			nilDB:     true,
			wantErr:   true,
			wantErrIs: ErrNilDB,
		},
		{
			name:   "success with close error",
			driver: "testsql",
			dsn:    "user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose().WillReturnError(errors.New("close error"))
			},
			wantConn: true,
		},
		{
			name:   "success",
			driver: "testsql",
			dsn:    "user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			wantConn: true,
		},
		{
			name:   "success with shutdown signal",
			driver: "testsql",
			dsn:    "user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			shutdownSig: true,
			wantConn:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			if tt.configMockFunc != nil {
				tt.configMockFunc(mock)
			}

			shutdownWG := &sync.WaitGroup{}
			shutdownSG := make(chan struct{})

			// The lifetime context governs shutdown; the establishment context
			// passed to New must not tear the pool down.
			lifeCtx, lifeCancel := context.WithCancel(context.Background())
			defer lifeCancel()

			defer func() {
				if tt.shutdownSig {
					close(shutdownSG)
				} else {
					lifeCancel()
				}

				// Deterministically wait for the watcher's Shutdown to run: the wait
				// group returns immediately for the error cases (never incremented)
				// and blocks until Add(-1) for the connected cases.
				shutdownWG.Wait()

				require.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
			}()

			mockConnectFunc := newMockConnectFunc(db, nil)
			if tt.connectErr != nil {
				mockConnectFunc = newMockConnectFunc(nil, tt.connectErr)
			}

			if tt.nilDB {
				mockConnectFunc = newMockConnectFunc(nil, nil)
			}

			conn, err := New(
				t.Context(),
				tt.driver,
				tt.dsn,
				WithConnectFunc(mockConnectFunc),
				WithShutdownWaitGroup(shutdownWG),
				WithShutdownSignalChan(shutdownSG),
				WithLifetimeContext(lifeCtx),
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			}

			if (conn != nil) != tt.wantConn {
				t.Errorf("New() gotConn = %v, wantConn %v", conn != nil, tt.wantConn)
			}
		})
	}
}

func TestNew_nilContext(t *testing.T) {
	t.Parallel()

	// A nil context must fail fast rather than panic deep inside context helpers.
	conn, err := New(nil, "testsql", "user:pass@tcp(db.host.invalid:1234)/testdb") //nolint:staticcheck // intentionally nil to exercise the guard
	require.Nil(t, conn)
	require.ErrorIs(t, err, ErrNilContext)
}

//nolint:gocognit
func TestConnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		connectURL     string
		connectErr     error
		configMockFunc func(sqlmock.Sqlmock)
		wantConn       bool
		shutdownSig    bool
		wantErr        bool
	}{
		{
			name:       "fail with config validation error",
			connectURL: "",
			wantErr:    true,
		},
		{
			name:       "fail to open DB connection",
			connectURL: "testsql://user:pass@tcp(db.host.invalid:1234)/testdb",
			connectErr: errors.New("db open error"),
			wantErr:    true,
		},
		{
			name:       "success with close error",
			connectURL: "testsql://user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose().WillReturnError(errors.New("close error"))
			},
			wantConn: true,
		},
		{
			name:       "success",
			connectURL: "testsql://user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			wantConn: true,
		},
		{
			name:       "success with shutdown signal",
			connectURL: "testsql://user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			shutdownSig: true,
			wantConn:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			if tt.configMockFunc != nil {
				tt.configMockFunc(mock)
			}

			shutdownWG := &sync.WaitGroup{}
			shutdownSG := make(chan struct{})

			// The lifetime context governs shutdown; the establishment context
			// passed to Connect must not tear the pool down.
			lifeCtx, lifeCancel := context.WithCancel(context.Background())
			defer lifeCancel()

			defer func() {
				if tt.shutdownSig {
					close(shutdownSG)
				} else {
					lifeCancel()
				}

				// Deterministically wait for the watcher's Shutdown to run: the wait
				// group returns immediately for the error cases (never incremented)
				// and blocks until Add(-1) for the connected cases.
				shutdownWG.Wait()

				require.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
			}()

			mockConnectFunc := newMockConnectFunc(db, nil)
			if tt.connectErr != nil {
				mockConnectFunc = newMockConnectFunc(nil, tt.connectErr)
			}

			conn, err := Connect(
				t.Context(),
				tt.connectURL,
				WithConnectFunc(mockConnectFunc),
				WithShutdownWaitGroup(shutdownWG),
				WithShutdownSignalChan(shutdownSG),
				WithLifetimeContext(lifeCtx),
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (conn != nil) != tt.wantConn {
				t.Errorf("Connect() gotConn = %v, wantConn %v", conn != nil, tt.wantConn)
			}
		})
	}
}

func TestSQLConn_DB(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	mock.ExpectClose()

	mockConnectFunc := newMockConnectFunc(db, nil)
	conn, err := Connect(t.Context(), "testsql://user:pass@tcp(db.host.invalid:1234)/testdb", WithConnectFunc(mockConnectFunc))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, db, conn.DB())

	// Shut down explicitly to close the pool and stop the watcher goroutine.
	require.NoError(t, conn.Shutdown(t.Context()))
	require.Nil(t, conn.DB())

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLConn_Shutdown_idempotent(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	// Close must be expected exactly once: subsequent Shutdown calls are no-ops.
	mock.ExpectClose()

	shutdownWG := &sync.WaitGroup{}
	shutdownSG := make(chan struct{})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	mockConnectFunc := newMockConnectFunc(db, nil)

	conn, err := New(
		ctx,
		"testsql",
		"user:pass@tcp(db.host.invalid:1234)/testdb",
		WithConnectFunc(mockConnectFunc),
		WithShutdownWaitGroup(shutdownWG),
		WithShutdownSignalChan(shutdownSG),
	)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// First shutdown closes the DB and decrements the wait group.
	require.NoError(t, conn.Shutdown(ctx))
	require.Nil(t, conn.DB())

	// A second direct call must be a no-op: no panic, no double close, and the
	// wait group must not go negative (a negative wait group would panic on Wait).
	require.NoError(t, conn.Shutdown(ctx))

	// A concurrent third call (mimicking the watcher goroutine racing a deferred
	// call) must also be safe; assert is used because this runs off the test goroutine.
	done := make(chan struct{})

	go func() {
		defer close(done)

		assert.NoError(t, conn.Shutdown(ctx))
	}()

	<-done

	// Wait returns immediately without panicking only if the counter is exactly 0.
	shutdownWG.Wait()

	require.NoError(t, mock.ExpectationsWereMet())
}

// recordingHandler is a minimal slog.Handler that records emitted messages so a
// test can observe log output without a real backend.
type recordingHandler struct {
	mu   *sync.Mutex
	msgs *[]string
}

func (h recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	*h.msgs = append(*h.msgs, r.Message)

	return nil
}

func (h recordingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h recordingHandler) WithGroup(_ string) slog.Handler      { return h }

// TestSQLConn_Shutdown_stopsWatcher verifies a direct Shutdown unblocks the
// watcher goroutine (via the internal done channel) so it cannot leak when
// neither the context nor the shutdown signal channel is ever triggered.
func TestSQLConn_Shutdown_stopsWatcher(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	mock.ExpectClose()

	var (
		mu   sync.Mutex
		msgs []string
	)

	logger := slog.New(recordingHandler{mu: &mu, msgs: &msgs})

	conn, err := New(
		t.Context(),
		"testsql",
		"user:pass@tcp(db.host.invalid:1234)/testdb",
		WithConnectFunc(newMockConnectFunc(db, nil)),
		WithLogger(logger),
	)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Only the direct Shutdown below must stop the watcher.
	require.NoError(t, conn.Shutdown(t.Context()))
	require.Nil(t, conn.DB())

	// The watcher logs a distinct message when it wakes on the done channel; its
	// appearance proves the goroutine observed the shutdown and is terminating.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return slices.Contains(msgs, "sqlconn direct shutdown")
	}, time.Second, 5*time.Millisecond)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLConn_HealthCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		configOpts            []Option
		disconnectBeforeCheck bool
		wantErr               bool
	}{
		{
			name:                  "fail because unavailable",
			disconnectBeforeCheck: true,
			wantErr:               true,
		},
		{
			name: "fail with check connection error",
			configOpts: []Option{
				WithCheckConnectionFunc(func(_ context.Context, _ *sql.DB) error {
					return errors.New("check error")
				}),
			},
			wantErr: true,
		},
		{
			name: "success",
			configOpts: []Option{
				WithCheckConnectionFunc(func(_ context.Context, _ *sql.DB) error {
					return nil
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)
			mock.ExpectClose()

			mockConnectFunc := newMockConnectFunc(db, nil)

			opts := append([]Option{WithConnectFunc(mockConnectFunc)}, tt.configOpts...)

			conn, err := Connect(t.Context(), "testsql://user:pass@tcp(db.host.invalid:1234)/testdb", opts...)
			require.NoError(t, err)
			require.NotNil(t, conn)
			require.Equal(t, db, conn.DB())

			// Ensure the watcher goroutine is stopped and the pool closed.
			defer func() { _ = conn.Shutdown(context.Background()) }()

			if tt.disconnectBeforeCheck {
				require.NoError(t, conn.Shutdown(t.Context()))
				require.Nil(t, conn.DB())
			}

			err = conn.HealthCheck(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.disconnectBeforeCheck {
				require.ErrorIs(t, err, ErrUnavailable)
			}
		})
	}
}

//nolint:gocognit
func Test_checkConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		query          string
		configMockFunc func(sqlmock.Sqlmock)
		wantErr        bool
	}{
		{
			name: "fail with ping error",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing().WillReturnError(errors.New("ping error"))
			},
			wantErr: true,
		},
		{
			name: "fail with exec error",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing()
				m.ExpectQuery("SELECT 1").WillReturnError(errors.New("exec error"))
			},
			wantErr: true,
		},
		{
			name: "succeed",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing()
				m.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
			},
			wantErr: false,
		},
		{
			name:  "succeed with custom validation query",
			query: "SELECT 1 FROM DUAL",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing()
				m.ExpectQuery("SELECT 1 FROM DUAL").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
			},
			wantErr: false,
		},
		{
			name:  "succeed with non-numeric validation query result",
			query: "SELECT 'ok'",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing()
				m.ExpectQuery("SELECT 'ok'").WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow("ok"))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			if tt.configMockFunc != nil {
				tt.configMockFunc(mock)
			}

			cfg := defaultConfig("testsql", "testdsn")
			if tt.query != "" {
				cfg.validationQuery = tt.query
			}

			err = cfg.checkConnection(t.Context(), db)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkConnection() error = %v, wantErr %v", err, tt.wantErr)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:gocognit
func Test_connectOnce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfgDriver      string
		cfgDSN         string
		setupConfig    func(*config, *sql.DB)
		configMockFunc func(sqlmock.Sqlmock)
		wantErrMsg     string
		want           bool
		wantErr        bool
	}{
		{
			name: "fail with sql error",
			setupConfig: func(c *config, _ *sql.DB) {
				c.sqlOpenFunc = func(_, _ string) (*sql.DB, error) {
					return nil, errors.New("open error")
				}
			},
			wantErr: true,
		},
		{
			name: "fail with nil DB from sql open",
			setupConfig: func(c *config, _ *sql.DB) {
				c.sqlOpenFunc = func(_, _ string) (*sql.DB, error) {
					return nil, nil //nolint:nilnil // exercise the nil-handle guard
				}
			},
			wantErrMsg: "nil database handle",
			wantErr:    true,
		},
		{
			name: "fail with connection check error and close the opened DB",
			setupConfig: func(c *config, db *sql.DB) {
				c.sqlOpenFunc = func(_, _ string) (*sql.DB, error) {
					return db, nil
				}
				c.checkConnectionFunc = func(_ context.Context, _ *sql.DB) error {
					return errors.New("check error")
				}
			},
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			wantErrMsg: "failed checking database connection",
			wantErr:    true,
		},
		{
			name: "fail with connection check error joined with close error",
			setupConfig: func(c *config, db *sql.DB) {
				c.sqlOpenFunc = func(_, _ string) (*sql.DB, error) {
					return db, nil
				}
				c.checkConnectionFunc = func(_ context.Context, _ *sql.DB) error {
					return errors.New("check error")
				}
			},
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose().WillReturnError(errors.New("close error"))
			},
			wantErrMsg: "close error",
			wantErr:    true,
		},
		{
			name: "succeed",
			setupConfig: func(c *config, db *sql.DB) {
				c.sqlOpenFunc = func(_, _ string) (*sql.DB, error) {
					return db, nil
				}
				c.checkConnectionFunc = func(_ context.Context, _ *sql.DB) error {
					return nil
				}
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			if tt.configMockFunc != nil {
				tt.configMockFunc(mock)
			}

			cfg := defaultConfig(tt.cfgDriver, tt.cfgDSN)
			if tt.setupConfig != nil {
				tt.setupConfig(cfg, db)
			}

			got, err := connectOnce(t.Context(), cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("connectOnce() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErrMsg != "" {
				require.ErrorContains(t, err, tt.wantErrMsg)
			}

			require.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

			if tt.want {
				require.Equal(t, db, got, "connectOnce() got = %v, want %v", got, db)
				return
			}

			require.Nil(t, got, "connectOnce() expected nil DB")
		})
	}
}

func Test_parseConnectionURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		url        string
		wantDriver string
		wantDSN    string
	}{
		{
			name:       "empty",
			url:        "",
			wantDriver: "",
			wantDSN:    "",
		},
		{
			name:       "mysql",
			url:        "mysql://user:pass@tcp(host:3306)/database",
			wantDriver: "mysql",
			wantDSN:    "user:pass@tcp(host:3306)/database",
		},
		{ //nolint:gosec
			name:       "postgres",
			url:        "pgx://postgres://user:pass@host:5432/database?sslmode=disable",
			wantDriver: "pgx",
			wantDSN:    "postgres://user:pass@host:5432/database?sslmode=disable",
		},
		{
			name:       "missing driver",
			url:        "user:pass@tcp(db1.host.invalid)/db1",
			wantDriver: "",
			wantDSN:    "user:pass@tcp(db1.host.invalid)/db1",
		},
		{
			name:       "full connection URL",
			url:        "testdriver://user:pass@tcp(db2.host.invalid)/db2",
			wantDriver: "testdriver",
			wantDSN:    "user:pass@tcp(db2.host.invalid)/db2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDriver, gotDSN := parseConnectionURL(tt.url)

			require.Equal(t, tt.wantDriver, gotDriver)
			require.Equal(t, tt.wantDSN, gotDSN)
		})
	}
}
