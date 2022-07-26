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
	defaultMaxRules = 3

	// DefaultURLQueryFilterKey is the default URL query key used by Processor.ParseURLQuery().
	// Can be customized with WithQueryFilterKey().
	DefaultURLQueryFilterKey = "filter"
)

// ParseJSON parses and returns a [][]Rule from its JSON representation.
func ParseJSON(s string) ([][]Rule, error) {
	var r [][]Rule
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, fmt.Errorf("failed unmarshaling rules: %w", err)
	}

	return r, nil
}

// New returns a new Processor with the rules and the given options.
//
// The first level of rules is matched with an AND operator and the second level with an OR.
//
// "[a,[b,c],d]" evaluates to "a AND (b OR c) AND d".
func New(opts ...Option) (*Processor, error) {
	p := &Processor{
		maxRules:          defaultMaxRules,
		urlQueryFilterKey: DefaultURLQueryFilterKey,
	}

	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// Processor provides the filtering logic and methods.
type Processor struct {
	fields            fieldGetter
	maxRules          int
	urlQueryFilterKey string
}

// ParseURLQuery parses and returns the defined query parameter from a *url.URL.
// Defaults to DefaultURLQueryFilterKey and can be customized with WithQueryFilterKey().
func (p *Processor) ParseURLQuery(u *url.URL) ([][]Rule, error) {
	return ParseJSON(
		u.Query().Get(p.urlQueryFilterKey),
	)
}

// Apply filters the slice to remove elements not matching the defined rules.
// The slice parameter must be a pointer to a slice and is filtered *in place*.
func (p *Processor) Apply(rules [][]Rule, slicePtr interface{}) error {
	if len(rules) == 0 {
		return nil
	}

	err := p.checkRulesCount(rules)
	if err != nil {
		return err
	}

	vSlicePtr := reflect.ValueOf(slicePtr)
	if vSlicePtr.Kind() != reflect.Ptr {
		return fmt.Errorf("slicePtr should be a slice pointer but is %s", vSlicePtr.Type())
	}

	vSlice := vSlicePtr.Elem()
	if vSlice.Kind() != reflect.Slice {
		return fmt.Errorf("slicePtr should be a slice pointer but is %s", vSlicePtr.Type())
	}

	matcher := func(obj interface{}) (bool, error) {
		return p.evaluateRules(rules, obj)
	}

	return p.filterSliceValue(vSlice, matcher)
}

func (p *Processor) checkRulesCount(rules [][]Rule) error {
	count := 0
	for i := range rules {
		count += len(rules[i])
	}

	if count > p.maxRules {
		return fmt.Errorf("too many rules: got %d max is %d", count, p.maxRules)
	}

	return nil
}

// filterSliceValue filters a slice passed as a reflect.Value, in place.
// It calls the matcher function to evaluate whether to keep each item or not.
func (p *Processor) filterSliceValue(slice reflect.Value, matcher func(interface{}) (bool, error)) error {
	n := 0

	for i := 0; i < slice.Len(); i++ {
		value := slice.Index(i)

		// value can always be Interface() because it's in a slice and cannot point to an unexported field
		match, err := matcher(value.Interface())
		if err != nil {
			return err
		}

		if match {
			// replace unselected elements by the ones that match
			slice.Index(n).Set(value)
			n++
		}
	}

	// shorten the slice to the actual number of elements
	slice.SetLen(n)

	return nil
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
