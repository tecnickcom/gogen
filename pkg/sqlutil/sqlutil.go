// Package sqlutil provides common SQL utilities.
package sqlutil

import (
	"strconv"
	"strings"
)

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

	values := make([]string, len(in))

	for k, v := range in {
		values[k] = "'" + v + "'"
	}

	return "`" + field + "` IN (" + strings.Join(values, ",") + ")"
}
