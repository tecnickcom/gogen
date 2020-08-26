// +build unit

package sqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func newMockConnectFunc(db *sql.DB, err error) ConnectFunc {
	return func(ctx context.Context, cfg *config) (*sql.DB, error) {
		return db, err
	}
}

func TestConnect(t *testing.T) {
	tests := []struct {
		name           string
		connectDSN     string
		connectErr     error
		configMockFunc func(sqlmock.Sqlmock)
		wantConn       bool
		wantErr        bool
	}{
		{
			name:       "fail with dsn error",
			connectDSN: "driver1://driver2://connection_string",
			wantErr:    true,
		},
		{
			name:       "fail with config validation error",
			connectDSN: "",
			wantErr:    true,
		},
		{
			name:       "fail to open DB connection",
			connectDSN: "testsql://user:pass@tcp(db.host.invalid:1234)/testdb",
			connectErr: fmt.Errorf("db open error"),
			wantErr:    true,
		},
		{
			name:       "success with close error",
			connectDSN: "testsql://user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose().WillReturnError(fmt.Errorf("close error"))
			},
			wantConn: true,
		},
		{
			name:       "success",
			connectDSN: "testsql://user:pass@tcp(db.host.invalid:1234)/testdb",
			configMockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			wantConn: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			if tt.configMockFunc != nil {
				tt.configMockFunc(mock)
			}

			ctx, cancel := context.WithCancel(testutil.Context())
			defer func() {
				cancel()

				// wait to allow the disconnect goroutine to execute
				time.Sleep(100 * time.Millisecond)

				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unfulfilled expectations: %s", err)
				}
			}()

			mockConnectFunc := newMockConnectFunc(db, nil)
			if tt.connectErr != nil {
				mockConnectFunc = newMockConnectFunc(nil, tt.connectErr)
			}

			conn, err := Connect(ctx, tt.connectDSN, WithConnectFunc(mockConnectFunc))
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

	ctx, cancel := context.WithCancel(testutil.Context())
	defer cancel()

	mockConnectFunc := newMockConnectFunc(db, nil)
	conn, err := Connect(ctx, "testsql://user:pass@tcp(db.host.invalid:1234)/testdb", WithConnectFunc(mockConnectFunc))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, db, conn.DB())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSQLConn_HealthCheck(t *testing.T) {
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
				WithCheckConnectionFunc(func(ctx context.Context, db *sql.DB) error {
					return fmt.Errorf("check error")
				}),
			},
			wantErr: true,
		},
		{
			name: "success",
			configOpts: []Option{
				WithCheckConnectionFunc(func(ctx context.Context, db *sql.DB) error {
					return nil
				}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(testutil.Context())
			defer cancel()

			mockConnectFunc := newMockConnectFunc(db, nil)

			opts := []Option{
				WithConnectFunc(mockConnectFunc),
			}
			opts = append(opts, tt.configOpts...)

			conn, err := Connect(ctx, "testsql://user:pass@tcp(db.host.invalid:1234)/testdb", opts...)
			require.NoError(t, err)
			require.NotNil(t, conn)
			require.Equal(t, db, conn.DB())

			if tt.disconnectBeforeCheck {
				cancel()

				// wait to allow the disconnect goroutine to execute
				time.Sleep(100 * time.Millisecond)
			}

			if err := conn.HealthCheck(testutil.Context()); (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_checkConnection(t *testing.T) {
	tests := []struct {
		name           string
		configMockFunc func(sqlmock.Sqlmock)
		wantErr        bool
	}{
		{
			name: "fail with ping error",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing().WillReturnError(fmt.Errorf("ping error"))
			},
			wantErr: true,
		},
		{
			name: "fail with exec error",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing()
				m.ExpectQuery("SELECT 1").WillReturnError(fmt.Errorf("exec error"))
			},
			wantErr: true,
		},
		{
			name: "succeed",
			configMockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectPing()
				m.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}))
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			if tt.configMockFunc != nil {
				tt.configMockFunc(mock)
			}

			if err := checkConnection(testutil.Context(), db); (err != nil) != tt.wantErr {
				t.Errorf("checkConnection() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func Test_connectWithBackoff(t *testing.T) {
	tests := []struct {
		name        string
		cfgDriver   string
		cfgDSN      string
		setupConfig func(*config, *sql.DB)
		want        bool
		wantErr     bool
	}{
		{
			name: "fail with sql error",
			setupConfig: func(c *config, db *sql.DB) {
				c.sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
					return nil, fmt.Errorf("open error")
				}
			},
			wantErr: true,
		},
		{
			name: "fail with connection check error",
			setupConfig: func(c *config, db *sql.DB) {
				c.sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
					return db, nil
				}
				c.checkConnectionFunc = func(ctx context.Context, db *sql.DB) error {
					return fmt.Errorf("check error")
				}
			},
			wantErr: true,
		},
		{
			name: "succeed",
			setupConfig: func(c *config, db *sql.DB) {
				c.sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
					return db, nil
				}
				c.checkConnectionFunc = func(ctx context.Context, db *sql.DB) error {
					return nil
				}
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			cfg := defaultConfig(tt.cfgDriver, tt.cfgDSN)
			if tt.setupConfig != nil {
				tt.setupConfig(cfg, db)
			}

			got, err := connectWithBackoff(testutil.Context(), cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("connectWithBackoff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want {
				require.Equal(t, db, got, "connectWithBackoff() got = %v, want %v", got, db)
				return
			}
			require.Nil(t, got, "connectWithBackoff() expected nil DB")
		})
	}
}

func Test_parseConnectionURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantDriver string
		wantDSN    string
		wantErr    bool
	}{
		{
			name:       "empty",
			url:        "",
			wantDriver: "",
			wantDSN:    "",
		},
		{
			name:       "empty",
			url:        "driver1://driver2://user:pass@tcp(db0.host.invalid)/db0",
			wantDriver: "",
			wantDSN:    "",
			wantErr:    true,
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
			gotDriver, gotDSN, err := parseConnectionURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConnectionURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotDriver != tt.wantDriver {
				t.Errorf("parseConnectionURL() gotDriver = %v, want %v", gotDriver, tt.wantDriver)
			}
			if gotDSN != tt.wantDSN {
				t.Errorf("parseConnectionURL() gotDSN = %v, want %v", gotDSN, tt.wantDSN)
			}
		})
	}
}
