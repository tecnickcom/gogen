package enumdb

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	table := "test_table"
	query := "SELECT `id`, `name` FROM `" + table + "`"
	queries := EnumTableQuery{
		table: query,
	}

	tests := []struct {
		name      string
		setupMock func(m sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "fails prepare statement",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectPrepare(query).
					WillReturnError(fmt.Errorf("load error"))
			},
			wantErr: true,
		},
		{
			name: "fails query",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectPrepare(query).
					ExpectQuery().
					WillReturnError(fmt.Errorf("query error"))
			},
			wantErr: true,
		},
		{
			name: "fails scan",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.
					NewRows([]string{"id", "name"}).
					AddRow("wrong_type", "test_value")

				m.ExpectPrepare(query).
					ExpectQuery().
					WillReturnRows(rows).
					RowsWillBeClosed()
			},
			wantErr: true,
		},
		{
			name: "fails with rows error",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.
					NewRows([]string{"id", "name"}).
					AddRow(7, "test_value").
					RowError(0, fmt.Errorf("row error"))

				m.ExpectPrepare(query).
					ExpectQuery().
					WillReturnRows(rows).
					RowsWillBeClosed()
			},
			wantErr: true,
		},
		{
			name: "succeed loading data",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.
					NewRows([]string{"id", "name"}).
					AddRow(1, "alpha").
					AddRow(2, "bravo").
					AddRow(3, "charlie")

				m.ExpectPrepare(query).
					ExpectQuery().
					WillReturnRows(rows).
					RowsWillBeClosed()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err, "Unexpected error while creating sqlmock", err)
			defer func() { _ = mockDB.Close() }()

			mock.MatchExpectationsInOrder(false)
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			cache, err := New(testutil.Context(), mockDB, queries)

			if tt.wantErr {
				require.Error(t, err, "an error was expected")
			} else {
				require.NotNil(t, cache, "the cache should not be nil")
				id, err := cache[table].ID("bravo")
				require.NoError(t, err)
				require.Equal(t, 2, id)
			}

			require.NoError(t, mock.ExpectationsWereMet(), "DB expectations not met")
		})
	}
}
