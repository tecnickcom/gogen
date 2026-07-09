package filter

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule_Evaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rule    Rule
		value   any
		want    bool
		wantErr bool
	}{
		{
			name:  "true - equal",
			rule:  Rule{Type: TypeEqual, Value: "hello"},
			value: "hello",
			want:  true,
		},
		{
			name:  "false - equal",
			rule:  Rule{Type: TypeEqual, Value: "hello"},
			value: "world",
			want:  false,
		},
		{
			name:  "true - negated equal",
			rule:  Rule{Type: TypePrefixNot + TypeEqual, Value: "hello"},
			value: "world",
			want:  true,
		},
		{
			name:    "error - invalid type",
			rule:    Rule{Type: "nope", Value: "hello"},
			value:   "hello",
			wantErr: true,
		},
		{
			name:    "error - invalid regexp",
			rule:    Rule{Type: TypeRegexp, Value: "("},
			value:   "hello",
			wantErr: true,
		},
		{
			name:    "error - invalid negated base type",
			rule:    Rule{Type: TypePrefixNot + TypeRegexp, Value: "("},
			value:   "hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.rule.evaluate(tt.value)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestRule_Evaluate_Concurrent confirms the exported Rule.Evaluate is safe to call on the
// same shared Rule from multiple goroutines (run under -race).
func TestRule_Evaluate_Concurrent(t *testing.T) {
	t.Parallel()

	rule := Rule{Field: "", Type: TypeRegexp, Value: "^hello"}

	const goroutines = 32

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			got, err := rule.evaluate("hello world")
			assert.NoError(t, err)
			assert.True(t, got)
		}()
	}

	wg.Wait()
}
