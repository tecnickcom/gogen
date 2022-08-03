// Package filter provides generic filtering capabilities for struct slices.
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

// New returns a new Processor with the rules and the given options.
//
// The first level of rules is matched with an AND operator and the second level with an OR.
//
// "[a,[b,c],d]" evaluates to "a AND (b OR c) AND d".
func New(opts ...Option) (*Processor, error) {
	p := &Processor{
		maxRules:          DefaultMaxRules,
		maxResults:        DefaultMaxResults,
		urlQueryFilterKey: DefaultURLQueryFilterKey,
	}

	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// ParseURLQuery parses and returns the defined query parameter from a *url.URL.
// Defaults to DefaultURLQueryFilterKey and can be customized with WithQueryFilterKey().
//
// If the query parameter is empty or missing, will return a nil slice.
// If there is a value which is invalid, will return an error.
func (p *Processor) ParseURLQuery(q url.Values) ([][]Rule, error) {
	value := q.Get(p.urlQueryFilterKey)
	if value == "" {
		return nil, nil
	}

	return ParseJSON(value)
}

// Apply filters the slice to remove elements not matching the defined rules.
// The slice parameter must be a pointer to a slice and is filtered *in place*.
//
// This is a shortcut to ApplySubset with 0 offset and maxResults length.
//
// Returns the length of the filtered slice, the total number of elements that matched the filter, and the eventual error.
func (p *Processor) Apply(rules [][]Rule, slicePtr interface{}) (sliceLen, totalMatches uint, err error) {
	return p.ApplySubset(rules, slicePtr, 0, p.maxResults)
}

// ApplySubset filters the slice to remove elements not matching the defined rules.
// The slice parameter must be a pointer to a slice and is filtered *in place*.
//
// Depending on offset, the first results are filtered even if they match
// Depending on length, the filtered slice will only contain a set number of elements.
//
// Returns the length of the filtered slice, the total number of elements that matched the filter, and the eventual error.
func (p *Processor) ApplySubset(rules [][]Rule, slicePtr interface{}, offset, length uint) (sliceLen, totalMatches uint, err error) {
	if length < 1 {
		return 0, 0, errors.New("length must be at least 1")
	}

	if length > p.maxResults {
		return 0, 0, errors.New("length must be less than maxResults")
	}

	err = p.checkRulesCount(rules)
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

	matcher := func(obj interface{}) (bool, error) {
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

// filterSliceValue filters a slice passed as a reflect.Value, in place.
// It calls the matcher function to evaluate whether to keep each item or not.
//
// n is number of matched elements in the slice.
// m is number of total matched elements.
func (p *Processor) filterSliceValue(slice reflect.Value, offset uint, length int, matcher func(interface{}) (bool, error)) (n int, m uint, err error) {
	skip := offset

	for i := 0; i < slice.Len(); i++ {
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

// nolint: gocognit
func (p *Processor) evaluateRules(rules [][]Rule, obj interface{}) (bool, error) {
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

// evaluateRule evaluates a specific rule over an object.
//
// It needs a pointer to let the Rule reuse its state (e.g. precompiled regexp).
func (p *Processor) evaluateRule(rule *Rule, obj interface{}) (bool, error) {
	value, err := p.fields.GetFieldValue(obj, rule.Field)
	if errors.Is(err, errFieldNotFound) {
		return false, nil // filter out missing field without error
	}

	if err != nil {
		return false, err
	}

	return rule.Evaluate(value)
}

// ParseJSON parses and returns a [][]Rule from its JSON representation.
func ParseJSON(s string) ([][]Rule, error) {
	var r [][]Rule
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, fmt.Errorf("failed unmarshaling rules: %w", err)
	}

	return r, nil
}
