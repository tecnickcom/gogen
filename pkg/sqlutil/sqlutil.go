// Package sqlutil provides common SQL utilities.
package sqlutil

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	conditionIn    = "IN"
	conditionNotIn = "NOT IN"
)

// BuildInClauseString prepares a SQL IN clause with the given list of string values.
func BuildInClauseString(field string, values []string) string {
	return composeInClause(conditionIn, field, formatStrings(values))
}

// BuildNotInClauseString prepares a SQL NOT IN clause with the given list of string values.
func BuildNotInClauseString(field string, values []string) string {
	return composeInClause(conditionNotIn, field, formatStrings(values))
}

// BuildInClauseInt prepares a SQL IN clause with the given list of integer values.
func BuildInClauseInt(field string, values []int) string {
	return composeInClause(conditionIn, field, formatInts(values))
}

// BuildNotInClauseInt prepares a SQL NOT IN clause with the given list of integer values.
func BuildNotInClauseInt(field string, values []int) string {
	return composeInClause(conditionNotIn, field, formatInts(values))
}

// BuildInClauseUint prepares a SQL IN clause with the given list of integer values.
func BuildInClauseUint(field string, values []uint64) string {
	return composeInClause(conditionIn, field, formatUints(values))
}

// BuildNotInClauseUint prepares a SQL NOT IN clause with the given list of integer values.
func BuildNotInClauseUint(field string, values []uint64) string {
	return composeInClause(conditionNotIn, field, formatUints(values))
}

func formatStrings(values []string) []string {
	items := make([]string, len(values))

	for k, v := range values {
		items[k] = "'" + v + "'"
	}

	return items
}

func formatInts(values []int) []string {
	items := make([]string, len(values))

	for k, v := range values {
		items[k] = strconv.Itoa(v)
	}

	return items
}

func formatUints(values []uint64) []string {
	items := make([]string, len(values))

	for k, v := range values {
		items[k] = strconv.FormatUint(v, 10)
	}

	return items
}

func composeInClause(condition string, field string, values []string) string {
	if len(values) == 0 {
		return ""
	}

	return fmt.Sprintf("`%s` %s (%s)", field, condition, strings.Join(values, ","))
}
