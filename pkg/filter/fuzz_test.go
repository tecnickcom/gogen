package filter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzFilterApply feeds arbitrary filter JSON through the bounded Processor.ParseJSON and then
// Apply, over both a concrete struct slice and an interface ([]any) slice, asserting the
// package's untrusted-input contract: parsing and applying can never panic; every error a
// client filter causes is wrapped with ErrInvalidFilter (so a handler can always answer 400);
// and no json.Number ever survives parsing into a rule value (it is a string kind and would be
// mistaken for a quoted string by an evaluator). Cross-checking JSON-vs-Go value exactness is
// covered deterministically by TestFilter_Apply_JSONGoRuleParity. Run the generative fuzzer
// with "go test -fuzz FuzzFilterApply ./pkg/filter/".
func FuzzFilterApply(f *testing.F) {
	seeds := []string{
		`[[{"field":"StringField","type":"==","value":"x"}]]`,
		`[[{"field":"","type":"regexp","value":".*"}]]`,
		`[[{"field":"Nested.Inner","type":"<","value":5}]]`,
		`[[{"field":"Any","type":"=","value":"z"},{"field":"IntField","type":"!>=","value":3}]]`,
		`[[{"field":"Missing.Deep","type":"~=","value":"a"}]]`,
		`[[{"field":"BigField","type":"==","value":9007199254740993}]]`,
		`[[{"field":"BigField","type":">","value":18446744073709551615}]]`,
		`[[{"field":"Any","type":"==","value":{"a":[1,2]}}]]`,
		`[[{"field":"Nested.Inner","type":">=","value":1e400}]]`,
		`[[{"field":"StringField","type":"=="}]]`,
		`[[{"type":"==","value":"x"}]]`,
		`[[{"field":"Any","type":"==","value":null}]]`,
		`[[]]`,
		`[]`,
		`not json`,
		``,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	type inner struct {
		Inner int
		Name  string
	}

	type row struct {
		StringField string
		IntField    int
		BigField    int64
		Nested      inner
		Any         any
	}

	structData := func() []row {
		return []row{
			{StringField: "x", IntField: 1, BigField: 1 << 53, Nested: inner{Inner: 2, Name: "a"}, Any: "z"},
			{StringField: "y", IntField: 9, BigField: 1<<53 + 1, Any: nil},
		}
	}

	anyData := func() []any {
		return []any{
			row{StringField: "x", IntField: 1, BigField: 1 << 53, Any: "z"},
			nil,
			42,
		}
	}

	p, err := New()
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, s string) {
		rules, err := p.ParseJSON(s)
		if err != nil {
			require.ErrorIs(t, err, ErrInvalidFilter)

			return
		}

		requireNoJSONNumberInRules(t, rules)

		// Must never panic on untrusted rules over either element-slice shape, and every error
		// must be attributable to the filter so a handler can answer it with a 400.
		structSlice := structData()
		requireApplyErrorIsInvalid(t, p, rules, &structSlice)

		anySlice := anyData()
		requireApplyErrorIsInvalid(t, p, rules, &anySlice)
	})
}

// requireApplyErrorIsInvalid applies rules to slicePtr and asserts that any error is an
// ErrInvalidFilter (Apply must never panic on untrusted rules).
func requireApplyErrorIsInvalid(t *testing.T, p *Processor, rules [][]Rule, slicePtr any) {
	t.Helper()

	_, _, err := p.Apply(rules, slicePtr)
	if err != nil {
		require.ErrorIs(t, err, ErrInvalidFilter)
	}
}

// requireNoJSONNumberInRules asserts that no parsed rule value is a json.Number. The parser
// rejects array and object values outright, so a surviving value is always a scalar — a
// non-recursive check suffices, and its purpose is to catch a regression that re-admits a
// composite (or a raw json.Number) without conversion.
func requireNoJSONNumberInRules(t *testing.T, rules [][]Rule) {
	t.Helper()

	for i := range rules {
		for j := range rules[i] {
			_, isNumber := rules[i][j].Value.(json.Number)
			require.False(t, isNumber, "rule value is still a json.Number")

			switch rules[i][j].Value.(type) {
			case []any, map[string]any:
				t.Fatalf("composite rule value survived parsing: %T", rules[i][j].Value)
			}
		}
	}
}
