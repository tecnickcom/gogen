package filter

import (
	"errors"
	"fmt"
)

// Option is the function that allows to set configuration options.
type Option func(p *Processor) error

// WithFieldNameTag allows to use the field names specified by the tag instead of the original struct names.
//
// Returns an error if the tag is empty.
func WithFieldNameTag(tag string) Option {
	return func(p *Processor) error {
		if tag == "" {
			return errors.New("tag cannot be empty")
		}

		p.fields.fieldTag = tag

		return nil
	}
}

// WithMaxRules sets the maximum number of rules to pass to the Processor.Apply() function without errors.
// If this option is not set, it defaults to 3.
//
// Return an error if max is less than 1.
func WithMaxRules(max int) Option {
	return func(p *Processor) error {
		if max < 1 {
			return fmt.Errorf("max rules must be stricly positive")
		}

		p.maxRules = max

		return nil
	}
}

// WithQueryFilterKey sets the query parameter key that Processor.ParseURLQuery() looks for.
func WithQueryFilterKey(key string) Option {
	return func(p *Processor) error {
		if key == "" {
			return errors.New("query filter key cannot be empty")
		}

		p.urlQueryFilterKey = key

		return nil
	}
}
