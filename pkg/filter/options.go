package filter

import (
	"errors"
)

// Option configures a [Processor] instance.
type Option func(p *Processor) error

// WithFieldNameTag configures field lookup to use the given struct tag instead of field names.
// Evaluates the tag value before the first comma (e.g., "json:my_field,omitempty" → "my_field").
// Returns error if tag is empty.
func WithFieldNameTag(tag string) Option {
	return func(p *Processor) error {
		if tag == "" {
			return errors.New("tag cannot be empty")
		}

		p.fields.fieldTag = tag

		return nil
	}
}

// WithQueryFilterKey customizes the URL query parameter key that ParseURLQuery() searches for.
// Returns error if key is empty.
func WithQueryFilterKey(key string) Option {
	return func(p *Processor) error {
		if key == "" {
			return errors.New("query filter key cannot be empty")
		}

		p.urlQueryFilterKey = key

		return nil
	}
}

// WithMaxRules sets the maximum permitted rule count to limit evaluation runtime cost.
// Defaults to 3 if not set. Returns error if rulemax < 1.
func WithMaxRules(rulemax uint) Option {
	return func(p *Processor) error {
		if rulemax < 1 {
			return errors.New("max Rules must be at least 1")
		}

		p.maxRules = rulemax

		return nil
	}
}

// WithMaxResults sets the maximum returned-element count for Apply() and ApplySubset().
// Returns error if resmax < 1 or resmax > MaxResults.
func WithMaxResults(resmax uint) Option {
	return func(p *Processor) error {
		if resmax < 1 {
			return errors.New("maxResults must be at least 1")
		}

		if resmax > MaxResults {
			return errors.New("maxResults must be less than MaxResults")
		}

		p.maxResults = resmax

		return nil
	}
}
