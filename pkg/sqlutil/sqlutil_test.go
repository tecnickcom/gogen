package sqlutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// nolint:gocognit
func TestCloseRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMock   func(m sqlmock.Sqlmock)
		wantNilTest bool
		wantLog     bool
	}{
		{
			name: "fails with close error",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message"}).AddRow("hello")
				rows.CloseError(fmt.Errorf("close error"))

				m.ExpectPrepare("SELECT").
					ExpectQuery().
					WillReturnRows(rows).
					RowsWillBeClosed()
			},
			wantLog: true,
		},
		{
			name:        "nop with nil statement",
			wantNilTest: true,
			wantLog:     false,
		},
		{
			name: "succeed",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message"}).AddRow("hello")
				m.ExpectPrepare("SELECT").
					ExpectQuery().
					WillReturnRows(rows)
			},
			wantLog: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			require.NoError(t, err, "Unexpected error while creating sqlmock", err)
			defer func() { _ = db.Close() }()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			var rows *sql.Rows
			if !tt.wantNilTest {
				// nolint:sqlclosecheck
				stmt, err := db.Prepare("SELECT")
				defer CloseStatement(context.Background(), stmt)
				require.NoError(t, err)

				rows, err = stmt.Query()
				require.NoError(t, err)
				require.NoError(t, rows.Err())
			}

			ctx, logs := testutil.ContextWithLogObserver(zap.ErrorLevel)
			CloseRows(ctx, rows)

			if tt.wantLog {
				require.Equal(t, 1, logs.Len(), "missing expected logs")
			} else {
				require.Equal(t, 0, logs.Len(), "unexpected logs")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

// nolint:gocognit
func TestCloseStatement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(m sqlmock.Sqlmock)
		setupStmt bool
		wantLog   bool
	}{
		{
			name:      "fails with close error",
			setupStmt: true,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectPrepare("SELECT").
					WillBeClosed().
					WillReturnCloseError(fmt.Errorf("close error"))
			},
			wantLog: true,
		},
		{
			name: "nop with nil statement",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
			},
			wantLog: false,
		},
		{
			name: "succeed",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
			},
			wantLog: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			require.NoError(t, err, "Unexpected error while creating sqlmock", err)
			defer func() { _ = db.Close() }()
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}
			tx, err := db.Begin()
			require.NoError(t, err)
			var stmt *sql.Stmt
			if tt.setupStmt {
				stmt, err = tx.Prepare("SELECT")
				require.NoError(t, err)
			}
			ctx, logs := testutil.ContextWithLogObserver(zap.ErrorLevel)
			CloseStatement(ctx, stmt)
			if tt.wantLog {
				require.Equal(t, 1, logs.Len(), "missing expected logs")
			} else {
				require.Equal(t, 0, logs.Len(), "unexpected logs")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestBuildInClauseInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []int
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_1",
			values: []int{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_2",
			values: []int{99},
			want:   "`test_2` IN (99)",
		},
		{
			name:   "expect multiple values",
			field:  "test_3",
			values: []int{11, 13, 17},
			want:   "`test_3` IN (11,13,17)",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BuildInClauseInt(tt.field, tt.values); got != tt.want {
				t.Errorf("BuildInClauseInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildInClauseString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		values []string
		want   string
	}{
		{
			name:   "expect empty",
			field:  "test_1",
			values: []string{},
			want:   "",
		},
		{
			name:   "expect single value",
			field:  "test_2",
			values: []string{"A"},
			want:   "`test_2` IN ('A')",
		},
		{
			name:   "expect multiple values",
			field:  "test_3",
			values: []string{"B", "C"},
			want:   "`test_3` IN ('B','C')",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BuildInClauseString(tt.field, tt.values); got != tt.want {
				t.Errorf("BuildInClauseString() = %v, want %v", got, tt.want)
			}
		})
	}
}
