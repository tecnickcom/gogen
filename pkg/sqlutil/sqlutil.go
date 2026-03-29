/*
Package sqlutil solves a common SQL string-construction problem: safely quoting
identifiers and string literals when generating query fragments dynamically.

# Problem

Applications that build SQL fragments at runtime often need to quote table,
schema, and column identifiers, as well as string values. Doing this ad hoc
across a codebase is error-prone and can lead to malformed SQL or inconsistent
escaping behavior.

sqlutil centralizes quoting behavior behind a small configurable API.

# What It Provides

[New] returns a configurable [SQLUtil] instance exposing:

  - [SQLUtil.QuoteID] for quoting identifiers (schema/table/column names).
  - [SQLUtil.QuoteValue] for quoting string literal values.

The default implementation is mysql-like:

  - identifiers are split by "." and each segment is wrapped in backticks,
    with embedded backticks escaped as doubled backticks.
  - values are wrapped in single quotes, with embedded single quotes doubled.
  - control characters are escaped (`\0`, `\n`, `\r`, `\\`, `\Z`).

# Customization

Use options to adapt quoting rules for different SQL dialects:

  - [WithQuoteIDFunc] replaces identifier quoting behavior.
  - [WithQuoteValueFunc] replaces value quoting behavior.

This allows Postgres/SQLite/other dialect-specific quoting while preserving the
same calling pattern throughout the codebase.

# Important Boundary

This package is intended for quoting identifiers and string literals in dynamic
query generation. It is not a replacement for prepared statements and query
parameterization. Continue using placeholders and bound parameters for runtime
data whenever possible.

# Usage

	u, err := sqlutil.New()
	if err != nil {
	    return err
	}

	col := u.QuoteID("users.email")      // `users`.`email`
	val := u.QuoteValue("o'reilly")      // 'o''reilly'
	query := "SELECT " + col + " FROM " + u.QuoteID("users") + " WHERE " + col + " = " + val
	_ = query
*/
package sqlutil

import (
	"errors"
	"strings"
)

// SQLQuoteFunc is the type of function called to quote a string (ID or value).
type SQLQuoteFunc func(s string) string

// SQLUtil is the structure that helps to manage a SQL DB connection.
type SQLUtil struct {
	quoteIDFunc    SQLQuoteFunc
	quoteValueFunc SQLQuoteFunc
}

// New constructs SQL utility with configurable identifier and value quoting functions (default: MySQL-style).
func New(opts ...Option) (*SQLUtil, error) {
	c := defaultSQLUtil()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	err := c.validate()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// defaultSQLUtil creates a SQLUtil instance with default settings.
func defaultSQLUtil() *SQLUtil {
	return &SQLUtil{
		quoteIDFunc:    defaultQuoteID,
		quoteValueFunc: defaultQuoteValue,
	}
}

// QuoteID quotes identifiers (schema/table/column names) with configurable SQL dialect rules.
func (c *SQLUtil) QuoteID(s string) string {
	return c.quoteIDFunc(s)
}

// QuoteValue quotes string literal values with configurable SQL dialect escape rules; includes surrounding quotes.
func (c *SQLUtil) QuoteValue(s string) string {
	return c.quoteValueFunc(s)
}

// validate checks if the SQLUtil instance is properly configured.
func (c *SQLUtil) validate() error {
	if c.quoteIDFunc == nil {
		return errors.New("the QuoteID function must be set")
	}

	if c.quoteValueFunc == nil {
		return errors.New("the QuoteValue function must be set")
	}

	return nil
}

// defaultQuoteID is the QuoteID default function for mysql-like databases.
func defaultQuoteID(s string) string {
	if s == "" {
		return s
	}

	parts := strings.Split(s, ".")

	for k, v := range parts {
		parts[k] = "`" + strings.ReplaceAll(escape(v), "`", "``") + "`"
	}

	return strings.Join(parts, ".")
}

// defaultQuoteValue is the QuoteValue default function for mysql-like databases.
func defaultQuoteValue(s string) string {
	if s == "" {
		return s
	}

	return "'" + strings.ReplaceAll(escape(s), "'", "''") + "'"
}

// escape escapes special characters in a string for safe SQL inclusion.
func escape(s string) string {
	dest := make([]byte, 0, 2*len(s))

	for i := range len(s) {
		c := s[i]

		switch c {
		case 0:
			dest = append(dest, '\\', '0')
		case '\n':
			dest = append(dest, '\\', 'n')
		case '\r':
			dest = append(dest, '\\', 'r')
		case '\\':
			dest = append(dest, '\\', '\\')
		case '\032':
			dest = append(dest, '\\', 'Z')
		default:
			dest = append(dest, c)
		}
	}

	return string(dest)
}
