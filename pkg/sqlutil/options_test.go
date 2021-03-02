package sqlutil

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithQuoteIDFunc(t *testing.T) {
	t.Parallel()

	v := func(s string) string {
		return s
	}

	c := &SQLUtil{}
	WithQuoteIDFunc(v)(c)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(c.quoteIDFunc).Pointer())
}

func TestWithQuoteValueFunc(t *testing.T) {
	t.Parallel()

	v := func(s string) string {
		return s
	}

	c := &SQLUtil{}
	WithQuoteValueFunc(v)(c)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(c.quoteValueFunc).Pointer())
}
