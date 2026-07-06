package sqlutil

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	sqlConditionIn    = "IN"
	sqlConditionNotIn = "NOT IN"

	// sqlMatchNone is the predicate emitted for an empty IN clause: it never
	// matches any row, preserving the "value is in the empty set" semantics.
	sqlMatchNone = "1 = 0"

	// sqlMatchAll is the predicate emitted for an empty NOT IN clause: it matches
	// every row, preserving the "value is not in the empty set" semantics.
	sqlMatchAll = "1 = 1"
)

// BuildInClauseString prepares a SQL IN clause with the given list of string values.
// An empty list yields a never-matching predicate (1 = 0).
func (c *SQLUtil) BuildInClauseString(field string, values []string) string {
	return c.composeInClause(sqlConditionIn, field, c.formatStrings(values))
}

// BuildNotInClauseString prepares a SQL NOT IN clause with the given list of string values.
// An empty list yields an always-matching predicate (1 = 1).
func (c *SQLUtil) BuildNotInClauseString(field string, values []string) string {
	return c.composeInClause(sqlConditionNotIn, field, c.formatStrings(values))
}

// BuildInClauseInt prepares a SQL IN clause with the given list of signed integer values.
// An empty list yields a never-matching predicate (1 = 0).
func (c *SQLUtil) BuildInClauseInt(field string, values []int) string {
	return c.composeInClause(sqlConditionIn, field, formatInts(values))
}

// BuildNotInClauseInt prepares a SQL NOT IN clause with the given list of signed integer values.
// An empty list yields an always-matching predicate (1 = 1).
func (c *SQLUtil) BuildNotInClauseInt(field string, values []int) string {
	return c.composeInClause(sqlConditionNotIn, field, formatInts(values))
}

// BuildInClauseInt64 prepares a SQL IN clause with the given list of 64-bit signed integer values.
// An empty list yields a never-matching predicate (1 = 0).
func (c *SQLUtil) BuildInClauseInt64(field string, values []int64) string {
	return c.composeInClause(sqlConditionIn, field, formatInt64s(values))
}

// BuildNotInClauseInt64 prepares a SQL NOT IN clause with the given list of 64-bit signed integer values.
// An empty list yields an always-matching predicate (1 = 1).
func (c *SQLUtil) BuildNotInClauseInt64(field string, values []int64) string {
	return c.composeInClause(sqlConditionNotIn, field, formatInt64s(values))
}

// BuildInClauseUint prepares a SQL IN clause with the given list of unsigned integer values.
// An empty list yields a never-matching predicate (1 = 0).
func (c *SQLUtil) BuildInClauseUint(field string, values []uint64) string {
	return c.composeInClause(sqlConditionIn, field, formatUints(values))
}

// BuildNotInClauseUint prepares a SQL NOT IN clause with the given list of unsigned integer values.
// An empty list yields an always-matching predicate (1 = 1).
func (c *SQLUtil) BuildNotInClauseUint(field string, values []uint64) string {
	return c.composeInClause(sqlConditionNotIn, field, formatUints(values))
}

// composeInClause constructs the final IN or NOT IN clause string. An empty value
// list collapses to a constant predicate so that concatenating the result into a
// WHERE clause preserves set semantics instead of silently dropping the filter:
// an empty IN never matches, an empty NOT IN always matches.
func (c *SQLUtil) composeInClause(condition string, field string, values []string) string {
	if len(values) == 0 {
		if condition == sqlConditionNotIn {
			return sqlMatchAll
		}

		return sqlMatchNone
	}

	return fmt.Sprintf("%s %s (%s)", c.QuoteID(field), condition, strings.Join(values, ","))
}

// formatStrings quotes each string value for safe SQL inclusion.
func (c *SQLUtil) formatStrings(values []string) []string {
	items := make([]string, len(values))

	for k, v := range values {
		items[k] = c.QuoteValue(v)
	}

	return items
}

// formatInts converts each signed integer value to its string representation.
func formatInts(values []int) []string {
	items := make([]string, len(values))

	for k, v := range values {
		items[k] = strconv.Itoa(v)
	}

	return items
}

// formatInt64s converts each 64-bit signed integer value to its string representation.
func formatInt64s(values []int64) []string {
	items := make([]string, len(values))

	for k, v := range values {
		items[k] = strconv.FormatInt(v, 10)
	}

	return items
}

// formatUints converts each unsigned integer value to its string representation.
func formatUints(values []uint64) []string {
	items := make([]string, len(values))

	for k, v := range values {
		items[k] = strconv.FormatUint(v, 10)
	}

	return items
}
