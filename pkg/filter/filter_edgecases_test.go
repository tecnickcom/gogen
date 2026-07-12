package filter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestFilter_Apply_PointerLeafField covers comparison against a pointer-typed leaf field: the
// leaf must be dereferenced so its pointed-to value is compared, and a nil leaf compares as a
// nil operand. A between-segments reflect.Indirect never reaches the leaf, so this needs its
// own handling.
func TestFilter_Apply_PointerLeafField(t *testing.T) {
	t.Parallel()

	name := "alice"
	age := 30
	pname := &name

	type row struct {
		Name    *string
		Age     *int
		PPName  **string
		Missing *string
	}

	p, err := New()
	require.NoError(t, err)

	bob := "bob"
	age99 := 99

	t.Run("string pointer leaf equals pointed-to value", func(t *testing.T) {
		t.Parallel()

		data := []row{{Name: &name}, {Name: &bob}}
		n, total, err := p.Apply([][]Rule{{{Field: "Name", Type: TypeEqual, Value: "alice"}}}, &data)
		require.NoError(t, err)
		require.Equal(t, uint(1), n)
		require.Equal(t, uint(1), total)
		require.Equal(t, []row{{Name: &name}}, data)
	})

	t.Run("int pointer leaf orders by pointed-to value", func(t *testing.T) {
		t.Parallel()

		data := []row{{Age: &age}, {Age: &age99}}
		n, _, err := p.Apply([][]Rule{{{Field: "Age", Type: TypeLTE, Value: 42}}}, &data)
		require.NoError(t, err)
		require.Equal(t, uint(1), n)
		require.Equal(t, []row{{Age: &age}}, data)
	})

	t.Run("double pointer leaf is fully dereferenced", func(t *testing.T) {
		t.Parallel()

		data := []row{{PPName: &pname}}
		n, _, err := p.Apply([][]Rule{{{Field: "PPName", Type: TypeEqual, Value: "alice"}}}, &data)
		require.NoError(t, err)
		require.Equal(t, uint(1), n)
	})

	t.Run("deep finite pointer chain resolves", func(t *testing.T) {
		t.Parallel()

		// A five-pointer leaf must still dereference to its value. This pins the unwrapLeaf
		// indirection bound to comfortably exceed any realistic pointer depth: resolving it
		// takes six loop iterations, so a bound below that would drop the leaf to a non-match.
		s := "deep"
		p1 := &s
		p2 := &p1
		p3 := &p2
		p4 := &p3
		p5 := &p4

		type deep struct{ V *****string }

		data := []deep{{V: p5}}
		n, _, err := p.Apply([][]Rule{{{Field: "V", Type: TypeEqual, Value: "deep"}}}, &data)
		require.NoError(t, err)
		require.Equal(t, uint(1), n)
	})

	t.Run("nil pointer leaf matches a null reference", func(t *testing.T) {
		t.Parallel()

		data := []row{{Name: &name}, {Name: nil}}
		n, _, err := p.Apply([][]Rule{{{Field: "Missing", Type: TypeEqual, Value: nil}}}, &data)
		require.NoError(t, err)
		require.Equal(t, uint(2), n) // Missing is nil on both
	})

	t.Run("negation of an equality on a non-nil pointer leaf excludes the match", func(t *testing.T) {
		t.Parallel()

		data := []row{{Name: &name}, {Name: &bob}}
		n, _, err := p.Apply([][]Rule{{{Field: "Name", Type: "!==", Value: "alice"}}}, &data)
		require.NoError(t, err)
		require.Equal(t, uint(1), n) // only "bob" survives
		require.Equal(t, "bob", *data[0].Name)
	})
}

// TestFilter_Apply_NilTarget verifies that a nil target is reported as an error rather than
// panicking, for both an untyped nil and a nil interface value.
func TestFilter_Apply_NilTarget(t *testing.T) {
	t.Parallel()

	p, err := New()
	require.NoError(t, err)

	rules := [][]Rule{{{Field: "", Type: TypeEqual, Value: 1}}}

	require.NotPanics(t, func() {
		_, _, aerr := p.Apply(rules, nil)
		require.Error(t, aerr)
	})

	require.NotPanics(t, func() {
		var data any

		_, _, aerr := p.Apply(rules, data)
		require.Error(t, aerr)
	})
}

// TestFilter_Apply_UnresolvableFieldIsDeterministic verifies that every static resolution
// failure on a concrete element type is reported as ErrInvalidFilter regardless of whether the
// slice is empty — the empty slice no longer silently accepts a malformed selector.
func TestFilter_Apply_UnresolvableFieldIsDeterministic(t *testing.T) {
	t.Parallel()

	type row struct {
		Name   string
		secret string //nolint:unused // selected by name in the "unexported field" case
	}

	cases := []struct {
		name  string
		opts  []Option
		field string
	}{
		{name: "unknown field", field: "Nope"},
		{name: "descent into non-struct", field: "Name.Sub"},
		{name: "unexported field", field: "secret"},
		{name: "too deep", opts: []Option{WithMaxFieldDepth(1)}, field: "Name.Sub"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New(tc.opts...)
			require.NoError(t, err)

			rule := [][]Rule{{{Field: tc.field, Type: TypeEqual, Value: "x"}}}

			nonEmpty := []row{{Name: "x"}}
			_, _, errNonEmpty := p.Apply(rule, &nonEmpty)
			require.ErrorIs(t, errNonEmpty, ErrInvalidFilter, "non-empty slice")

			empty := []row{}
			_, _, errEmpty := p.Apply(rule, &empty)
			require.ErrorIs(t, errEmpty, ErrInvalidFilter, "empty slice")
		})
	}
}

// TestFilter_Apply_NarrowNumericFieldKinds exercises the narrow numeric kinds end-to-end
// through Apply's reflect.Value path (toNumericValue), which the any-boxing numeric_test only
// covers indirectly.
func TestFilter_Apply_NarrowNumericFieldKinds(t *testing.T) {
	t.Parallel()

	type row struct {
		I8  int8
		I16 int16
		I32 int32
		U8  uint8
		U16 uint16
		U32 uint32
		F32 float32
	}

	p, err := New()
	require.NoError(t, err)

	for _, field := range []string{"I8", "I16", "I32", "U8", "U16", "U32", "F32"} {
		data := []row{
			{I8: 10, I16: 10, I32: 10, U8: 10, U16: 10, U32: 10, F32: 10},
			{I8: 20, I16: 20, I32: 20, U8: 20, U16: 20, U32: 20, F32: 20},
		}

		nEq, _, err := p.Apply([][]Rule{{{Field: field, Type: TypeEqual, Value: 10}}}, &data)
		require.NoError(t, err)
		require.Equal(t, uint(1), nEq, "== on %s", field)

		data2 := []row{
			{I8: 10, I16: 10, I32: 10, U8: 10, U16: 10, U32: 10, F32: 10},
			{I8: 20, I16: 20, I32: 20, U8: 20, U16: 20, U32: 20, F32: 20},
		}

		nGt, _, err := p.Apply([][]Rule{{{Field: field, Type: TypeGT, Value: 15}}}, &data2)
		require.NoError(t, err)
		require.Equal(t, uint(1), nGt, "> on %s", field)
	}

	// A float32 field does not equal the float64 a client sends for a value it cannot hold.
	data := []row{{F32: 0.1}}
	nEq, _, err := p.Apply([][]Rule{{{Field: "F32", Type: TypeEqual, Value: 0.1}}}, &data)
	require.NoError(t, err)
	require.Equal(t, uint(0), nEq)
}

// TestFilter_Apply_OrderLengthFallback_ArrayAndMap covers the array and map cases of the
// ordering operators' collection-length fallback (only slice and string were tested).
func TestFilter_Apply_OrderLengthFallback_ArrayAndMap(t *testing.T) {
	t.Parallel()

	type row struct {
		Arr [3]int
		M   map[string]int
	}

	p, err := New()
	require.NoError(t, err)

	data := []row{
		{Arr: [3]int{1, 2, 3}, M: map[string]int{"a": 1, "b": 2}},
		{Arr: [3]int{1, 2, 3}, M: map[string]int{"a": 1}},
	}

	// Array length is always 3: > 2 matches both.
	nArr, _, err := p.Apply([][]Rule{{{Field: "Arr", Type: TypeGT, Value: 2}}}, &data)
	require.NoError(t, err)
	require.Equal(t, uint(2), nArr)

	data2 := []row{
		{M: map[string]int{"a": 1, "b": 2}},
		{M: map[string]int{"a": 1}},
	}

	// Map length >= 2 matches only the first.
	nMap, _, err := p.Apply([][]Rule{{{Field: "M", Type: TypeGTE, Value: 2}}}, &data2)
	require.NoError(t, err)
	require.Equal(t, uint(1), nMap)
}

// TestFilter_Apply_UnreachableFieldNegationAndNull verifies that a field made unreachable by a
// nil pointer is a non-match even under a negated operator or a null reference: the evaluator
// is never called for such an element, so negation cannot flip it to a match.
func TestFilter_Apply_UnreachableFieldNegationAndNull(t *testing.T) {
	t.Parallel()

	type inner struct{ Country string }

	type outer struct{ Addr *inner }

	p, err := New()
	require.NoError(t, err)

	newData := func() []outer {
		return []outer{{Addr: nil}, {Addr: &inner{Country: "EN"}}}
	}

	// !== over a nil-pointer path: the unreachable element stays a non-match, only the
	// reachable non-"EN"... here the reachable one IS "EN", so !== EN keeps neither.
	data := newData()
	nNeg, _, err := p.Apply([][]Rule{{{Field: "Addr.Country", Type: "!==", Value: "EN"}}}, &data)
	require.NoError(t, err)
	require.Equal(t, uint(0), nNeg)

	// == null over the same path: the unreachable element is a non-match, not a null match.
	data2 := newData()
	nNull, _, err := p.Apply([][]Rule{{{Field: "Addr.Country", Type: TypeEqual, Value: nil}}}, &data2)
	require.NoError(t, err)
	require.Equal(t, uint(0), nNull)
}

// TestFilter_Apply_NilVsEmptyCollectionOrdering distinguishes a nil collection (no ordering)
// from an empty one (length 0) under the ordering operators.
func TestFilter_Apply_NilVsEmptyCollectionOrdering(t *testing.T) {
	t.Parallel()

	type row struct{ Tags []string }

	p, err := New()
	require.NoError(t, err)

	// Empty (non-nil) slice has length 0, so < 1 matches it.
	empty := []row{{Tags: []string{}}}
	nEmpty, _, err := p.Apply([][]Rule{{{Field: "Tags", Type: TypeLT, Value: 1}}}, &empty)
	require.NoError(t, err)
	require.Equal(t, uint(1), nEmpty)

	// Nil slice has no ordering, so < 1 does not match, and !< 1 does.
	nilData := []row{{Tags: nil}}
	nNil, _, err := p.Apply([][]Rule{{{Field: "Tags", Type: TypeLT, Value: 1}}}, &nilData)
	require.NoError(t, err)
	require.Equal(t, uint(0), nNil)

	nilData2 := []row{{Tags: nil}}
	nNegNil, _, err := p.Apply([][]Rule{{{Field: "Tags", Type: "!<", Value: 1}}}, &nilData2)
	require.NoError(t, err)
	require.Equal(t, uint(1), nNegNil)
}

// TestLimitBoundariesAccepted verifies that a value exactly at a limit is accepted (only
// over-limit values are rejected).
func TestLimitBoundariesAccepted(t *testing.T) {
	t.Parallel()

	t.Run("field depth at the limit", func(t *testing.T) {
		t.Parallel()

		type inner struct{ C string }

		type mid struct{ B inner }

		type top struct{ A mid }

		p, err := New(WithMaxFieldDepth(3))
		require.NoError(t, err)

		data := []top{{A: mid{B: inner{C: "x"}}}}
		_, _, err = p.Apply([][]Rule{{{Field: "A.B.C", Type: TypeEqual, Value: "x"}}}, &data)
		require.NoError(t, err)
	})

	t.Run("value length at the limit", func(t *testing.T) {
		t.Parallel()

		p, err := New(WithMaxValueLength(8))
		require.NoError(t, err)

		type row struct{ Name string }

		data := []row{{Name: "12345678"}}
		_, _, err = p.Apply([][]Rule{{{Field: "Name", Type: TypeEqual, Value: "12345678"}}}, &data)
		require.NoError(t, err)
	})

	t.Run("filter payload at the limit", func(t *testing.T) {
		t.Parallel()

		payload := `[[{"field":"","type":"==","value":1}]]`

		p, err := New(WithMaxFilterBytes(uint(len(payload))))
		require.NoError(t, err)

		_, err = p.ParseJSON(payload)
		require.NoError(t, err)
	})
}

// TestRuleTypeCaseInsensitive pins that rule types are matched case-insensitively.
func TestRuleTypeCaseInsensitive(t *testing.T) {
	t.Parallel()

	p, err := New()
	require.NoError(t, err)

	type row struct{ Name string }

	data := []row{{Name: "hello"}, {Name: "world"}}
	n, _, err := p.Apply([][]Rule{{{Field: "Name", Type: "REGEXP", Value: "^hel"}}}, &data)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)
}

// TestApplySubset_SelectivePagination exercises offset/length against a rule that matches only
// some elements, so "skip N matches" is distinguishable from "skip N elements" and total
// counts all matches beyond the window.
func TestApplySubset_SelectivePagination(t *testing.T) {
	t.Parallel()

	p, err := New()
	require.NoError(t, err)

	// Matches are at indices 0,2,4,6 (the "m*" values); "x" never matches.
	data := []string{"m0", "x", "m1", "x", "m2", "x", "m3"}

	// Skip the first 2 matches, take 1: expect the third match "m2".
	n, total, err := p.ApplySubset([][]Rule{{{Field: "", Type: TypeRegexp, Value: "^m"}}}, &data, 2, 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)
	require.Equal(t, uint(4), total)
	require.Equal(t, []string{"m2"}, data)
}

// TestFieldCache_BoundedGrowth verifies that the per-Processor path cache stops growing at its
// ceiling while resolution keeps working past it.
func TestFieldCache_BoundedGrowth(t *testing.T) {
	t.Parallel()

	type node struct {
		Left  *node
		Right *node
		Name  string
	}

	p, err := New()
	require.NoError(t, err)

	p.fields.cache.maxEntries = 2

	for _, path := range []string{"Name", "Left.Name", "Right.Name", "Left.Left.Name", "Right.Right.Name"} {
		data := []node{{Name: "x"}}

		_, _, err := p.Apply([][]Rule{{{Field: path, Type: TypeEqual, Value: "x"}}}, &data)
		require.NoError(t, err, "path %s", path)
	}

	// The count must land exactly on the ceiling: it grew (so the counter increments and the
	// cache is actually populated) but did not exceed the cap (so growth is bounded).
	require.Equal(t, int64(2), p.fields.cache.count.Load())

	// Resolution still works for a path that was never cached.
	data := []node{{Name: "x"}}
	n, _, err := p.Apply([][]Rule{{{Field: "Left.Right.Name", Type: TypeEqual, Value: "x"}}}, &data)
	require.NoError(t, err)
	require.Equal(t, uint(0), n) // Left is nil, so the path is unreachable: non-match
}

// TestFieldCache_DefaultProcessorCaches verifies that a default Processor (maxEntries == 0)
// actually caches resolved paths, exercising the ceiling == 0 default branch.
func TestFieldCache_DefaultProcessorCaches(t *testing.T) {
	t.Parallel()

	type row struct {
		A string
		B string
	}

	p, err := New()
	require.NoError(t, err)

	for _, field := range []string{"A", "B"} {
		data := []row{{A: "x", B: "y"}}

		_, _, err := p.Apply([][]Rule{{{Field: field, Type: TypeEqual, Value: "x"}}}, &data)
		require.NoError(t, err)
	}

	require.Equal(t, int64(2), p.fields.cache.count.Load())
}

// TestFilter_Apply_CyclicLeafDoesNotHang verifies that a pointer/interface cycle at a resolved
// leaf — a pathological data shape, not a filter shape — resolves to a non-match instead of
// spinning forever. The cycle is in the (server-side) data; the correct response is to exclude
// the element, never to hang the request.
func TestFilter_Apply_CyclicLeafDoesNotHang(t *testing.T) {
	t.Parallel()

	t.Run("self-referential interface", func(t *testing.T) {
		t.Parallel()

		type SelfRef struct{ Any any }

		var s SelfRef

		s.Any = &s.Any // an *any pointing at itself

		requireApplyReturns(t, []SelfRef{s}, Rule{Field: "Any", Type: TypeEqual, Value: "x"})
	})

	t.Run("self-pointer named type", func(t *testing.T) {
		t.Parallel()

		type Cycle *Cycle

		var c Cycle

		c = &c // a pointer that points at itself

		requireApplyReturns(t, []struct{ C Cycle }{{C: c}}, Rule{Field: "C", Type: TypeEqual, Value: "x"})
	})
}

// requireApplyReturns applies one rule to data under a watchdog and fails if Apply hangs.
func requireApplyReturns[T any](t *testing.T, data []T, rule Rule) {
	t.Helper()

	p, err := New()
	require.NoError(t, err)

	done := make(chan struct{})

	go func() {
		defer close(done)

		_, _, _ = p.Apply([][]Rule{{rule}}, &data)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Apply did not return: unwrapLeaf looped on a cyclic reference")
	}
}

// TestFilter_Apply_ComparableEqualIdentity pins that a Go-constructed equality rule whose
// reference is a comparable composite uses == (identity), not reflect.DeepEqual. Only reachable
// from a Go caller; JSON rule values are scalars.
func TestFilter_Apply_ComparableEqualIdentity(t *testing.T) {
	t.Parallel()

	type box struct{ P *int }

	a, b := 7, 7 // equal values, distinct addresses

	data := []box{{P: &b}, {P: &a}}

	p, err := New()
	require.NoError(t, err)

	// == box{P:&a} matches only the element holding &a; reflect.DeepEqual would match both.
	n, total, err := p.Apply([][]Rule{{{Field: "", Type: TypeEqual, Value: box{P: &a}}}}, &data)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)
	require.Equal(t, uint(1), total)
	require.Equal(t, &a, data[0].P)
}
