package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzFilterApply feeds arbitrary filter JSON through ParseJSON and Apply to assert
// that untrusted rules can never panic the process (errors are acceptable). Run the
// generative fuzzer with "go test -fuzz FuzzFilterApply ./pkg/filter/".
func FuzzFilterApply(f *testing.F) {
	seeds := []string{
		`[[{"field":"StringField","type":"==","value":"x"}]]`,
		`[[{"field":"","type":"regexp","value":".*"}]]`,
		`[[{"field":"Nested.Inner","type":"<","value":5}]]`,
		`[[{"field":"Any","type":"=","value":"z"},{"field":"IntField","type":"!>=","value":3}]]`,
		`[[{"field":"Missing.Deep","type":"~=","value":"a"}]]`,
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
		Nested      inner
		Any         any
	}

	f.Fuzz(func(t *testing.T, s string) {
		rules, err := parseJSON(s)
		if err != nil {
			return
		}

		p, err := New()
		require.NoError(t, err)

		data := []row{
			{StringField: "x", IntField: 1, Nested: inner{Inner: 2, Name: "a"}, Any: "z"},
			{StringField: "y", IntField: 9, Any: nil},
		}

		// Must never panic on untrusted rules; a returned error is fine.
		_, _, _ = p.Apply(rules, &data)
	})
}
