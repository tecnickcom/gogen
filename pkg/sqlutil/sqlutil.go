// Package sqlutil provides common SQL utilities.
package sqlutil

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

// CloseRows closes a rows instance or log an error.
func CloseRows(ctx context.Context, rows *sql.Rows) {
	if rows == nil {
		return
	}
	if err := rows.Close(); err != nil {
		logging.FromContext(ctx).Error("failed closing SQL rows", zap.Error(err))
	}
}

// CloseStatement closes a prepared statement or log an error.
func CloseStatement(ctx context.Context, stmt *sql.Stmt) {
	if stmt == nil {
		return
	}
	if err := stmt.Close(); err != nil {
		logging.FromContext(ctx).Error("failed closing SQL statement", zap.Error(err))
	}
}

// BuildInClauseInt prepares a SQL IN clause with the given list of integer values.
func BuildInClauseInt(field string, in []int) string {
	if len(in) == 0 {
		return ""
	}

	values := make([]string, len(in))
	for i, v := range in {
		values[i] = strconv.Itoa(v)
	}
	return "`" + field + "` IN (" + strings.Join(values, ",") + ")"
}

// BuildInClauseString prepares a SQL IN clause with the given list of string values.
func BuildInClauseString(field string, in []string) string {
	if len(in) == 0 {
		return ""
	}

	var values []string
	for _, v := range in {
		values = append(values, "'"+v+"'")
	}
	return "`" + field + "` IN (" + strings.Join(values, ",") + ")"
}
