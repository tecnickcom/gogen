package sqlutil

// Option is a type alias for a function that configures the DB connector.
type Option func(*SQLUtil)

// WithQuoteIDFunc replaces the default QuoteID function.
func WithQuoteIDFunc(fn SQLQuoteFunc) Option {
	return func(c *SQLUtil) {
		c.quoteIDFunc = fn
	}
}

// WithQuoteValueFunc replaces the default QuoteValue function.
func WithQuoteValueFunc(fn SQLQuoteFunc) Option {
	return func(c *SQLUtil) {
		c.quoteValueFunc = fn
	}
}
