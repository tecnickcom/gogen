/*
Package filter provides declarative, rule-based filtering for in-memory slices.

# Problem

API endpoints frequently need to apply client-driven filters to server-side
collections (for example, after parsing a `filter=` query parameter). Handwritten
loop logic is repetitive, hard to validate, and difficult to keep consistent
across data models.

# Solution

This package evaluates structured [Rule] expressions against slice elements and
filters the slice in place.

Rules are grouped as `[][]Rule` with boolean semantics:
  - outer slice: AND
  - inner slice: OR

So `[A, [B, C], D]` evaluates as `A AND (B OR C) AND D`.

Rules can be supplied directly or parsed from JSON via [ParseJSON], and query
parameter payloads can be loaded with [Processor.ParseURLQuery].

# Key Features

  - JSON-friendly filtering grammar for client-provided filters.
  - Rich comparison operators: regexp, equality/equal-fold, prefix/suffix,
    contains, and numeric/string ordering (<, <=, >, >=).
  - Optional negation prefix (`!`) for every operator.
  - Dot-path field selection for nested struct fields
    (for example `address.country`).
  - Optional struct-tag based field lookup via [WithFieldNameTag].
  - In-place filtering with pagination-style controls via [Processor.ApplySubset]
    (offset + length), plus total-match count.
  - Rule and result limits ([WithMaxRules], [WithMaxResults]) to constrain
    runtime cost.
  - Reflection-path caching for repeated evaluations on the same types.

# Important Behavior

  - The slice argument for [Processor.Apply] / [Processor.ApplySubset] must be a
    pointer to a slice and is modified in place.
  - Missing fields are treated as a non-match (filtered out) rather than an
    error.
  - ParseURLQuery returns nil rules when the configured query key is missing or
    empty.

# Example

The following pretty-printed JSON:

	[
	  [
	    {
	      "field": "name",
	      "type": "==",
	      "value": "doe"
	    },
	    {
	      "field": "age",
	      "type": "<=",
	      "value": 42
	    }
	  ],
	  [
	    {
	      "field": "address.country",
	      "type": "regexp",
	      "value": "^EN$|^FR$"
	    }
	  ]
	]

can be represented in one line as:

	[[{"field":"name","type":"==","value":"doe"},{"field":"age","type":"<=","value":42}],[{"field":"address.country","type":"regexp","value":"^EN$|^FR$"}]]

and URL-encoded as a query parameter:

	filter=%5B%5B%7B%22field%22%3A%22name%22%2C%22type%22%3A%22%3D%3D%22%2C%22value%22%3A%22doe%22%7D%2C%7B%22field%22%3A%22age%22%2C%22type%22%3A%22%3C%3D%22%2C%22value%22%3A42%7D%5D%2C%5B%7B%22field%22%3A%22address.country%22%2C%22type%22%3A%22regexp%22%2C%22value%22%3A%22%5EEN%24%7C%5EFR%24%22%7D%5D%5D

The equivalent logic is:

	((name==doe OR age<=42) AND (address.country match "EN" or "FR"))

# Available Rule Types

Supported rule types are:

  - `regexp` : matches the value against a reference regular expression.
  - `==`     : Equal to - matches exactly the reference value.
  - `=`      : Equal fold - matches when strings, interpreted as UTF-8, are equal under simple Unicode case-folding, which is a more general form of case-insensitivity. For example `AB` will match `ab`.
  - `^=`     : Starts with - (strings only) matches when the value begins with the reference string.
  - `=$`     : Ends with - (strings only) matches when the value ends with the reference string.
  - `~=`     : Contains - (strings only) matches when the reference string is a sub-string of the value.
  - `<`      : Less than - matches when the value is less than the reference.
  - `<=`     : Less than or equal to - matches when the value is less than or equal the reference.
  - `>`      : Greater than - matches when the value is greater than reference.
  - `>=`     : Greater than or equal to - matches when the value is greater than or equal the reference.

Every rule type can be prefixed with the symbol `!` to get the negated value.
For example `!==` is equivalent to "Not Equal", matching values that are
different.

# Benefits

filter enables expressive, validated, and reusable filtering logic for API and
application layers while minimizing repetitive per-type query code.
*/
package filter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
)

const (
	// MaxResults is the maximum number of results that can be returned.
	MaxResults = 1<<31 - 1 // math.MaxInt32

	// DefaultMaxResults is the default number of results for Apply.
	// Can be overridden with WithMaxResults().
	DefaultMaxResults = MaxResults

	// DefaultMaxRules is the default maximum number of rules.
	// Can be overridden with WithMaxRules().
	DefaultMaxRules = 3

	// DefaultURLQueryFilterKey is the default URL query key used by Processor.ParseURLQuery().
	// Can be customized with WithQueryFilterKey().
	DefaultURLQueryFilterKey = "filter"
)

// Processor provides the filtering logic and methods.
type Processor struct {
	fields            fieldGetter
	maxRules          uint
	maxResults        uint
	urlQueryFilterKey string
}

// New constructs a Processor for declarative rule-based filtering with custom options.
// The first rule level uses AND semantics; the second uses OR ("[a,[b,c],d]" → "a AND (b OR c) AND d").
// Customizable via field-tag lookup, query-parameter keys, and result/rule count limits.
func New(opts ...Option) (*Processor, error) {
	p := &Processor{
		maxRules:          DefaultMaxRules,
		maxResults:        DefaultMaxResults,
		urlQueryFilterKey: DefaultURLQueryFilterKey,
	}

	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

// ParseURLQuery unmarshals the JSON filter rule set from a URL query parameter.
// Defaults to the "filter" key; override with WithQueryFilterKey().
// Returns nil rules when the key is missing or empty; returns error on invalid JSON.
func (p *Processor) ParseURLQuery(q url.Values) ([][]Rule, error) {
	value := q.Get(p.urlQueryFilterKey)
	if value == "" {
		return nil, nil
	}

	return ParseJSON(value)
}

// Apply filters a slice by removing non-matching elements in place.
// Convenience method equivalent to ApplySubset(rules, slicePtr, 0, p.maxResults).
// Returns filtered-slice length, total-match count, and any error.
func (p *Processor) Apply(rules [][]Rule, slicePtr any) (uint, uint, error) {
	return p.ApplySubset(rules, slicePtr, 0, p.maxResults)
}

// ApplySubset filters a slice by removing non-matching elements in place, with pagination support.
// The slicePtr must be a pointer to a slice; offset and length control pagination within matches.
// Returns filtered-slice length within window, total-match count overall, and any error.
func (p *Processor) ApplySubset(rules [][]Rule, slicePtr any, offset, length uint) (uint, uint, error) {
	if length < 1 {
		return 0, 0, errors.New("length must be at least 1")
	}

	if length > p.maxResults {
		return 0, 0, errors.New("length must be less than maxResults")
	}

	err := p.checkRulesCount(rules)
	if err != nil {
		return 0, 0, err
	}

	vSlicePtr := reflect.ValueOf(slicePtr)
	if vSlicePtr.Kind() != reflect.Ptr {
		return 0, 0, fmt.Errorf("slicePtr should be a slice pointer but is %s", vSlicePtr.Type())
	}

	vSlice := vSlicePtr.Elem()
	if vSlice.Kind() != reflect.Slice {
		return 0, 0, fmt.Errorf("slicePtr should be a slice pointer but is %s", vSlicePtr.Type())
	}

	matcher := func(obj any) (bool, error) {
		return p.evaluateRules(rules, obj)
	}

	n, m, err := p.filterSliceValue(vSlice, offset, int(length), matcher)

	return uint(n), m, err
}

func (p *Processor) checkRulesCount(rules [][]Rule) error {
	var count int

	for i := range rules {
		count += len(rules[i])
	}

	if uint(count) > p.maxRules {
		return fmt.Errorf("too many rules: got %d max is %d", count, p.maxRules)
	}

	return nil
}

// filterSliceValue filters a reflect.Value slice in place using a matcher function.
// Returns matched-element count within pagination window and total-match count overall.
func (p *Processor) filterSliceValue(slice reflect.Value, offset uint, length int, matcher func(any) (bool, error)) (int, uint, error) {
	skip := offset

	var (
		n int
		m uint
	)

	for i := range slice.Len() {
		value := slice.Index(i)

		// value can always be Interface() because it's in a slice and cannot point to an unexported field
		match, err := matcher(value.Interface())
		if err != nil {
			return 0, 0, err
		}

		if !match {
			continue
		}

		m++

		if skip > 0 {
			skip--
			continue
		}

		if n < length {
			// replace unselected elements by the ones that match
			slice.Index(n).Set(value)

			n++
		}
	}

	// shorten the slice to the actual number of elements
	slice.SetLen(n)

	return n, m, nil
}

// evaluateRules applies AND-OR rule composition to determine object match: outer-AND, inner-OR.
//
//nolint:gocognit
func (p *Processor) evaluateRules(rules [][]Rule, obj any) (bool, error) {
	for i := range rules {
		orResult := false

		for j := range rules[i] {
			match, err := p.evaluateRule(&rules[i][j], obj)
			if err != nil {
				return false, err
			}

			if match {
				orResult = true
				break
			}
		}

		if !orResult {
			return false, nil
		}
	}

	return true, nil
}

// evaluateRule resolves a rule field, applies the evaluator, and treats missing fields as non-matches.
// Returns false without error for missing fields; pointer to obj allows Rule state reuse (e.g. compiled regexp).
func (p *Processor) evaluateRule(rule *Rule, obj any) (bool, error) {
	value, err := p.fields.GetFieldValue(obj, rule.Field)
	if errors.Is(err, errFieldNotFound) {
		return false, nil // filter out missing field without error
	}

	if err != nil {
		return false, err
	}

	return rule.Evaluate(value)
}

// ParseJSON unmarshals rule-set JSON into its composite AND-OR structure.
func ParseJSON(s string) ([][]Rule, error) {
	var r [][]Rule

	err := json.Unmarshal([]byte(s), &r)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling rules: %w", err)
	}

	return r, nil
}
