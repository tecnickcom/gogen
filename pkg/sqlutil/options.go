package sqlutil

// Option configures a [SQLUtil] instance.
type Option func(*SQLUtil)

// WithQuoteIDFunc customizes the identifier quoting function, e.g. double-quoted identifiers for Postgres/SQLite instead of the default MySQL-style backticks.
func WithQuoteIDFunc(fn SQLQuoteFunc) Option {
	return func(c *SQLUtil) {
		c.quoteIDFunc = fn
	}
}

// WithQuoteValueFunc customizes value quoting function for different SQL dialects.
func WithQuoteValueFunc(fn SQLQuoteFunc) Option {
	return func(c *SQLUtil) {
		c.quoteValueFunc = fn
	}
}
