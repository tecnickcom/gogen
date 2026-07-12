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
//
// The limit is applied to two counts: the number of AND groups (the outer slice), and the
// total number of rules summed across all the OR groups (the inner slices). So
// "name AND age AND (country==EN OR country==FR)" counts as 3 groups and 4 rules, and
// needs a limit of at least 4.
//
// Defaults to [DefaultMaxRules] if not set. Returns error if rulemax < 1.
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
			return errors.New("maxResults must not exceed MaxResults")
		}

		p.maxResults = resmax

		return nil
	}
}

// WithMaxValueLength sets the maximum byte length allowed for a rule's string value,
// such as a regexp pattern or a string comparison operand. Rules whose string value
// exceeds the limit are rejected at compile time (by [Processor.Apply] / [Processor.ApplySubset]),
// bounding regexp compilation and matching cost for untrusted filters. The check applies to any
// string-kinded value, including a named string type. Defaults to DefaultMaxValueLength.
// Returns error if maxlen < 1.
func WithMaxValueLength(maxlen uint) Option {
	return func(p *Processor) error {
		if maxlen < 1 {
			return errors.New("max value length must be at least 1")
		}

		p.maxValueLen = maxlen

		return nil
	}
}

// WithMaxFilterBytes sets the maximum byte length of the raw filter payload accepted
// by Processor.ParseJSON() (and hence Processor.ParseURLQuery()) before it is JSON-decoded,
// bounding parse-time cost for untrusted input. Defaults to DefaultMaxFilterBytes.
// Returns error if maxbytes < 1.
func WithMaxFilterBytes(maxbytes uint) Option {
	return func(p *Processor) error {
		if maxbytes < 1 {
			return errors.New("max filter bytes must be at least 1")
		}

		p.maxFilterBytes = maxbytes

		return nil
	}
}

// WithMaxFieldDepth sets the maximum number of dot-separated segments allowed in a
// rule's field selector (for example "a.b.c" has depth 3). Deeper selectors are rejected,
// bounding per-selector field-resolution cost. It bounds each selector's length, not the
// number of distinct selectors cached; that is capped separately and internally.
// Defaults to DefaultMaxFieldDepth. Returns error if maxdepth < 1.
func WithMaxFieldDepth(maxdepth uint) Option {
	return func(p *Processor) error {
		if maxdepth < 1 {
			return errors.New("max field depth must be at least 1")
		}

		p.fields.maxDepth = maxdepth

		return nil
	}
}
