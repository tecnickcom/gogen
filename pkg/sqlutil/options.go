package sqlutil

// Option configures a [SQLUtil] instance.
type Option func(*SQLUtil)

// WithQuoteIDFunc customizes identifier quoting function (e.g., for Postgres backticks vs PostgreSQL quotes).
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
