// Package sqlutil provides SQL utilities.
package sqlutil

import (
	"fmt"
	"strings"
)

// SQLQuoteFunc is the type of function called to quote a string (ID or value).
type SQLQuoteFunc func(s string) string

// SQLUtil is the structure that helps to manage a SQL DB connection.
type SQLUtil struct {
	quoteIDFunc    SQLQuoteFunc
	quoteValueFunc SQLQuoteFunc
}

// New creates a new instance.
func New(opts ...Option) (*SQLUtil, error) {
	c := defaultSQLUtil()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return c, nil
}

func defaultSQLUtil() *SQLUtil {
	return &SQLUtil{
		quoteIDFunc:    defaultQuoteID,
		quoteValueFunc: defaultQuoteValue,
	}
}

func (c *SQLUtil) validate() error {
	if c.quoteIDFunc == nil {
		return fmt.Errorf("the QuoteID function must be set")
	}

	if c.quoteValueFunc == nil {
		return fmt.Errorf("the QuoteValue function must be set")
	}

	return nil
}

// QuoteID quotes identifiers such as schema, table, or column names.
func (c *SQLUtil) QuoteID(s string) string {
	return c.quoteIDFunc(s)
}

// QuoteValue quotes database string values.
// The returned value will include all surrounding quotes.
func (c *SQLUtil) QuoteValue(s string) string {
	return c.quoteValueFunc(s)
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

func escape(s string) string {
	dest := make([]byte, 0, 2*len(s))

	for i := 0; i < len(s); i++ {
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
