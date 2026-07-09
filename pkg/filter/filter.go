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

Rules can be supplied directly or parsed from JSON via [Processor.ParseJSON], and
query parameter payloads can be loaded with [Processor.ParseURLQuery].

# Key Features

  - JSON-friendly filtering grammar for client-provided filters.
  - Rich comparison operators: regexp, equality/equal-fold, prefix/suffix,
    contains, and numeric ordering (<, <=, >, >=) with a collection/string
    length fallback.
  - Optional negation prefix (`!`) for every operator.
  - Dot-path field selection for nested struct fields
    (for example `address.country`).
  - Optional struct-tag based field lookup via [WithFieldNameTag].
  - In-place filtering with pagination-style controls via [Processor.ApplySubset]
    (offset + length), plus total-match count.
  - Limits to constrain runtime and untrusted-input cost: rule/result counts
    ([WithMaxRules], [WithMaxResults]), value length ([WithMaxValueLength]), payload
    size ([WithMaxFilterBytes]), and field-path depth ([WithMaxFieldDepth]).
  - Reflection-path caching for repeated evaluations on the same types.

# Important Behavior

  - The slice argument for [Processor.Apply] / [Processor.ApplySubset] must be a
    pointer to a slice and is modified in place: matching elements are compacted to
    the front and the slice is shortened (its backing array is not zeroed beyond the
    new length).
  - When a rule selects a field that cannot be resolved on an element — the field is
    missing, or the element (or a pointer along the path) is nil — the element is a
    non-match (filtered out) rather than an error.
  - [Processor.ParseURLQuery] returns nil rules when the configured query key is
    missing or empty.
  - A [Processor] and a compiled [][]Rule are safe for concurrent use across
    goroutines. That safety does not extend to a target slice shared between
    concurrent [Processor.Apply] calls, since Apply mutates it in place.

# Security

This package performs filtering only; it does NOT perform authorization. Filter
expressions are untrusted client input, and the grammar lets a client select and
compare any exported field of the elements (including nested fields). Callers MUST
therefore apply it only to data the requesting user is already authorized to see:
pre-filter the slice to that user's permitted records (and, where relevant, project
away fields they may not read) before calling [Processor.Apply]. Passing records that
contain fields the user is not entitled to would let a crafted filter probe those
values through the match result and total-match count.

Regular-expression rules use Go's RE2 engine, which matches in linear time with no
catastrophic backtracking, so a malicious pattern cannot cause exponential-time
matching. Cost is instead driven by input size, which the Processor bounds by
default: rule string values (including regexp patterns) are limited to
[DefaultMaxValueLength] bytes ([WithMaxValueLength]), the raw payload accepted by
[Processor.ParseURLQuery] is limited to [DefaultMaxFilterBytes] bytes
([WithMaxFilterBytes]), [WithMaxRules] caps how many rules may be applied, and
[WithMaxFieldDepth] bounds field-selector nesting (which also limits reflection-path
cache growth for recursive element types). Note that [Processor.Apply] still evaluates
every rule against every element regardless of the requested result window, so callers
should also bound the size of the slice being filtered. Decode untrusted filter JSON
with [Processor.ParseJSON] (or [Processor.ParseURLQuery]): these apply the
payload-size and rule-count limits.

Errors caused by a malformed or disallowed client filter are wrapped with
[ErrInvalidFilter]. A handler processing untrusted input can test
errors.Is(err, [ErrInvalidFilter]) to return a generic rejection (for example an HTTP
400) and log the detail server-side, rather than returning the underlying message,
which may echo client input or internal type names.

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

The ordering operators (<, <=, >, >=) require a numeric reference value and
compare numeric values directly; for strings, arrays, slices, and maps they
compare the length of the value against the reference (not lexicographic
order). Anything else evaluates to false.

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

	// DefaultMaxValueLength is the default maximum byte length of a rule's string
	// value (e.g. a regexp pattern). It bounds regexp compilation and matching cost
	// for untrusted filters. Can be overridden with WithMaxValueLength().
	DefaultMaxValueLength = 4096

	// DefaultMaxFilterBytes is the default maximum byte length of the raw filter
	// payload accepted by Processor.ParseURLQuery() before it is JSON-decoded. It
	// bounds parse-time cost for untrusted input. Can be overridden with WithMaxFilterBytes().
	DefaultMaxFilterBytes = 1 << 16 // 64 KiB

	// DefaultMaxFieldDepth is the default maximum number of dot-separated segments in a
	// rule field selector (e.g. "a.b.c" has depth 3). It bounds field-resolution cost
	// and, for recursive element types, the growth of the reflection-path cache. Can be
	// overridden with WithMaxFieldDepth().
	DefaultMaxFieldDepth = 32
)

// ErrInvalidFilter wraps every error attributable to a malformed or disallowed
// client filter: invalid JSON, an oversized payload or value, too many rules, an
// unsupported or mistyped rule, an uncompilable regexp, or a field selector that
// cannot be resolved. Callers handling untrusted input can test
// errors.Is(err, ErrInvalidFilter) to reject the request generically (for example
// an HTTP 400) and log the detail server-side, instead of returning the underlying
// message, which may echo client input or internal type names.
//
// Errors caused by misuse of the Apply/ApplySubset arguments by the calling code —
// a non-slice-pointer target, or an out-of-range length — are NOT wrapped with it.
var ErrInvalidFilter = errors.New("invalid filter")

// Processor provides the filtering logic and methods.
type Processor struct {
	fields            fieldGetter
	maxRules          uint
	maxResults        uint
	maxValueLen       uint
	maxFilterBytes    uint
	urlQueryFilterKey string
}

// New constructs a Processor for declarative rule-based filtering with custom options.
// The first rule level uses AND semantics; the second uses OR ("[a,[b,c],d]" → "a AND (b OR c) AND d").
// Customizable via field-tag lookup, query-parameter keys, count limits, and the
// untrusted-input safety limits (value length, payload size, field-path depth).
func New(opts ...Option) (*Processor, error) {
	p := &Processor{
		fields:            fieldGetter{maxDepth: DefaultMaxFieldDepth},
		maxRules:          DefaultMaxRules,
		maxResults:        DefaultMaxResults,
		maxValueLen:       DefaultMaxValueLength,
		maxFilterBytes:    DefaultMaxFilterBytes,
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
// Defaults to the "filter" key; override with [WithQueryFilterKey].
// Returns nil rules when the key is missing or empty; otherwise it delegates to
// [Processor.ParseJSON] and applies the same limits ([WithMaxFilterBytes], [WithMaxRules]).
// Filter-attributable errors are wrapped with [ErrInvalidFilter].
func (p *Processor) ParseURLQuery(q url.Values) ([][]Rule, error) {
	value := q.Get(p.urlQueryFilterKey)
	if value == "" {
		return nil, nil
	}

	return p.ParseJSON(value)
}

// ParseJSON decodes a JSON filter rule set with the Processor's safety limits applied:
// it rejects payloads larger than the configured maximum ([WithMaxFilterBytes]) before
// decoding, and rejects rule sets exceeding [WithMaxRules]. It is the only supported way
// to decode untrusted filter JSON; the unbounded decoder is internal. Filter-attributable
// errors are wrapped with [ErrInvalidFilter].
func (p *Processor) ParseJSON(s string) ([][]Rule, error) {
	// Reject oversized payloads before decoding, so a huge value cannot force a large
	// JSON allocation just to be rejected afterwards by the rule-count limit.
	if uint(len(s)) > p.maxFilterBytes {
		return nil, fmt.Errorf("%w: filter payload too large: got %d bytes max is %d", ErrInvalidFilter, len(s), p.maxFilterBytes)
	}

	rules, err := parseJSON(s)
	if err != nil {
		return nil, err
	}

	// Defense-in-depth: reject oversized client-provided rule sets at parse time.
	err = p.checkRulesCount(rules)
	if err != nil {
		return nil, err
	}

	return rules, nil
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
	if vSlicePtr.Kind() != reflect.Pointer {
		return 0, 0, fmt.Errorf("slicePtr should be a slice pointer but is %s", vSlicePtr.Type())
	}

	vSlice := vSlicePtr.Elem()
	if vSlice.Kind() != reflect.Slice {
		return 0, 0, fmt.Errorf("slicePtr should be a slice pointer but is %s", vSlicePtr.Type())
	}

	// Compile all evaluators and resolve field paths up front, before touching the
	// input slice. Paths for concrete element types are resolved once here rather than
	// per element; interface element types (e.g. []any) are resolved per element since
	// their concrete type varies. Compiling first also makes concurrent Apply on a
	// shared [][]Rule race-free (no per-rule lazy mutation) and ensures a misconfigured
	// rule fails before any in-place mutation truncates the slice.
	compiled, err := p.compileRules(rules, vSlice.Type().Elem())
	if err != nil {
		return 0, 0, err
	}

	matcher := func(elem reflect.Value) (bool, error) {
		return p.evaluateRules(compiled, elem)
	}

	n, m, err := p.filterSliceValue(vSlice, offset, int(length), matcher)

	return uint(n), m, err
}

func (p *Processor) checkRulesCount(rules [][]Rule) error {
	// Bound the number of OR groups too: empty groups contribute no rules to the count
	// below, so without this an arbitrarily large set of empty groups would pass while
	// still being allocated and compiled.
	if uint(len(rules)) > p.maxRules {
		return fmt.Errorf("%w: too many rule groups: got %d max is %d", ErrInvalidFilter, len(rules), p.maxRules)
	}

	var count int

	for i := range rules {
		count += len(rules[i])
	}

	if uint(count) > p.maxRules {
		return fmt.Errorf("%w: too many rules: got %d max is %d", ErrInvalidFilter, count, p.maxRules)
	}

	return nil
}

// filterSliceValue filters a reflect.Value slice in place using a matcher function.
// Returns matched-element count within pagination window and total-match count overall.
//
// The matching pass is completed before any mutation: matched indices within the
// pagination window are collected first, and the in-place compaction is committed only
// once the full pass succeeds. A matcher error therefore leaves the input slice untouched.
func (p *Processor) filterSliceValue(slice reflect.Value, offset uint, length int, matcher func(reflect.Value) (bool, error)) (int, uint, error) {
	selected, total, err := selectMatches(slice, offset, length, matcher)
	if err != nil {
		return 0, 0, err
	}

	// Commit phase: only reached after a fully successful pass, so a mid-iteration
	// error above can never truncate or clobber the caller's slice.
	for n, i := range selected {
		if n != i {
			slice.Index(n).Set(slice.Index(i))
		}
	}

	n := len(selected)

	// shorten the slice to the actual number of elements
	slice.SetLen(n)

	return n, total, nil
}

// selectMatches runs the matcher over every element and returns the original indices that
// match within the pagination window, plus the overall match count. It mutates nothing.
func selectMatches(slice reflect.Value, offset uint, length int, matcher func(reflect.Value) (bool, error)) ([]int, uint, error) {
	skip := offset

	var (
		selected []int
		total    uint
	)

	for i := range slice.Len() {
		// The element is passed as a reflect.Value rather than boxed via Interface():
		// only the specific field an evaluator reads is ever materialized, avoiding a
		// per-element allocation and copy of the whole element.
		match, err := matcher(slice.Index(i))
		if err != nil {
			return nil, 0, err
		}

		if !match {
			continue
		}

		total++

		if skip > 0 {
			skip--
			continue
		}

		if len(selected) < length {
			selected = append(selected, i)
		}
	}

	return selected, total, nil
}

// compiledRule pairs a resolved field selector with its pre-built evaluator.
//
// For concrete element types the field path is resolved once, at compile time,
// into path (or into the missing/resolveErr sentinels). For interface element
// types the concrete type is only known per element, so dynamic is set and the
// path is resolved during evaluation using field.
type compiledRule struct {
	eval       evaluator   // pre-built type-specific evaluator
	field      string      // original dot-path selector (for dynamic resolution and errors)
	path       reflectPath // field-index path for concrete element types
	wholeElem  bool        // field == "" → evaluate the element itself
	dynamic    bool        // interface element type → resolve path per element
	missing    bool        // field is statically absent → always a non-match
	resolveErr error       // static resolution failed (e.g. descent into a non-struct)
}

// compileRules pre-builds the evaluator and resolves the field path for every rule,
// surfacing configuration errors (invalid type, bad regexp, non-numeric ordering
// reference, ...) before any filtering. The result is independent of the input
// [][]Rule, so a shared rule set can be filtered concurrently without mutating
// per-rule state.
func (p *Processor) compileRules(rules [][]Rule, elemType reflect.Type) ([][]compiledRule, error) {
	compiled := make([][]compiledRule, len(rules))

	for i := range rules {
		compiled[i] = make([]compiledRule, len(rules[i]))

		for j := range rules[i] {
			cr, err := p.compileRule(rules[i][j], elemType)
			if err != nil {
				return nil, err
			}

			compiled[i][j] = cr
		}
	}

	return compiled, nil
}

// compileRule builds the evaluator and resolves the field path for a single rule
// against the slice element type. Missing fields are recorded as a non-match
// sentinel rather than an error, preserving the "missing field filters out" behavior.
func (p *Processor) compileRule(rule Rule, elemType reflect.Type) (compiledRule, error) {
	// Reject oversized string values (e.g. a huge regexp pattern) before building the
	// evaluator, so a pathological pattern is never handed to regexp.Compile.
	if s, ok := rule.Value.(string); ok && uint(len(s)) > p.maxValueLen {
		return compiledRule{}, fmt.Errorf("%w: rule value too large: got %d bytes max is %d", ErrInvalidFilter, len(s), p.maxValueLen)
	}

	eval, err := rule.getEvaluator()
	if err != nil {
		return compiledRule{}, err
	}

	cr := compiledRule{eval: eval, field: rule.Field}

	switch {
	case rule.Field == "":
		cr.wholeElem = true
	case elemType.Kind() == reflect.Interface:
		cr.dynamic = true
	default:
		path, rerr := p.fields.resolvePath(elemType, rule.Field)

		switch {
		case errors.Is(rerr, errFieldNotFound):
			cr.missing = true
		case rerr != nil:
			// Deferred so it only surfaces while evaluating an element, matching the
			// previous per-element behavior (an empty slice stays error-free).
			cr.resolveErr = rerr
		default:
			cr.path = path
		}
	}

	return cr, nil
}

// evaluateRules applies AND-OR rule composition to determine element match: outer-AND, inner-OR.
//
//nolint:gocognit
func (p *Processor) evaluateRules(rules [][]compiledRule, elem reflect.Value) (bool, error) {
	for i := range rules {
		orResult := false

		for j := range rules[i] {
			match, err := p.evaluateRule(rules[i][j], elem)
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

// evaluateRule resolves a rule's target value within elem and applies its evaluator.
// Missing fields and nil pointers along the path are treated as non-matches without error.
func (p *Processor) evaluateRule(rule compiledRule, elem reflect.Value) (bool, error) {
	value, ok, err := p.ruleValue(rule, elem)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, nil // missing field or nil pointer in path: non-match
	}

	return rule.eval.Evaluate(value), nil
}

// ruleValue resolves the reflect.Value a rule should evaluate for the given element.
// ok is false (with a nil error) when the field is absent or made unreachable by a
// nil pointer along the path: such elements are a non-match, not an error.
func (p *Processor) ruleValue(rule compiledRule, elem reflect.Value) (reflect.Value, bool, error) {
	if rule.missing {
		return reflect.Value{}, false, nil
	}

	if rule.resolveErr != nil {
		return reflect.Value{}, false, rule.resolveErr
	}

	// Unwrap an interface-typed element (e.g. []any) to its concrete dynamic value.
	value := elem
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	if rule.wholeElem {
		return value, true, nil
	}

	// Interface element types resolve their concrete field path per element; concrete
	// element types use the path resolved once at compile time (the common hot path).
	if rule.dynamic {
		return p.dynamicFieldValue(rule, value)
	}

	return walkFieldPath(value, rule.path)
}

// dynamicFieldValue resolves and reads a rule's field for an element of an
// interface-typed slice, whose concrete type is only known per element. ok is
// false without error when the field is absent (a non-match).
func (p *Processor) dynamicFieldValue(rule compiledRule, value reflect.Value) (reflect.Value, bool, error) {
	if !value.IsValid() {
		// A nil interface element has no fields: treat the selector as absent (a
		// non-match), consistently with a nil pointer along the path. This keeps a
		// single nil datum from failing the whole filter and avoids misattributing a
		// data condition to the client filter.
		return reflect.Value{}, false, nil
	}

	path, err := p.fields.resolvePath(value.Type(), rule.field)
	if errors.Is(err, errFieldNotFound) {
		return reflect.Value{}, false, nil
	}

	if err != nil {
		return reflect.Value{}, false, err
	}

	return walkFieldPath(value, path)
}

// walkFieldPath descends the field-index path from value, dereferencing pointers
// between segments. It returns ok=false (non-match) when a nil pointer makes the
// field unreachable, and an error only when the target field cannot be read
// (an unexported field). The returned Value has any interface leaf unwrapped to
// its concrete dynamic value.
func walkFieldPath(value reflect.Value, path reflectPath) (reflect.Value, bool, error) {
	for _, fieldIndex := range path {
		value = reflect.Indirect(value)
		if !value.IsValid() {
			// nil pointer along the path: treat the field as missing (non-match).
			return reflect.Value{}, false, nil
		}

		value = value.Field(fieldIndex)
	}

	if !value.CanInterface() {
		return reflect.Value{}, false, fmt.Errorf("%w: %s cannot be interfaced", ErrInvalidFilter, value.Type())
	}

	// Unwrap an interface-typed leaf field to the concrete value it holds, matching
	// how a boxed value would have been evaluated.
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	return value, true, nil
}

// parseJSON unmarshals rule-set JSON into its composite AND-OR structure. It applies
// no size limits; use [Processor.ParseJSON] for untrusted input.
func parseJSON(s string) ([][]Rule, error) {
	var r [][]Rule

	err := json.Unmarshal([]byte(s), &r)
	if err != nil {
		return nil, fmt.Errorf("%w: failed unmarshaling rules: %w", ErrInvalidFilter, err)
	}

	return r, nil
}
