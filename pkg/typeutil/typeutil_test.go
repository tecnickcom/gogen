package typeutil

import (
	"math"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestIsNil(t *testing.T) {
	t.Parallel()

	t.Run("not nil", func(t *testing.T) {
		t.Parallel()

		got := IsNil("string")
		require.False(t, got)
	})

	t.Run("nil value", func(t *testing.T) {
		t.Parallel()

		got := IsNil(nil)
		require.True(t, got)
	})

	t.Run("nil chan", func(t *testing.T) {
		t.Parallel()

		var nilChan chan int

		got := IsNil(nilChan)
		require.True(t, got)
	})

	t.Run("nil func", func(t *testing.T) {
		t.Parallel()

		var nilFunc func()

		got := IsNil(nilFunc)
		require.True(t, got)
	})

	t.Run("nil pointer to any", func(t *testing.T) {
		t.Parallel()

		var nilPtrToAny *any

		got := IsNil(nilPtrToAny)
		require.True(t, got)
	})

	t.Run("nil map", func(t *testing.T) {
		t.Parallel()

		var nilMap map[int]int

		got := IsNil(nilMap)
		require.True(t, got)
	})

	t.Run("nil slice", func(t *testing.T) {
		t.Parallel()

		var nilSlice []int

		got := IsNil(nilSlice)
		require.True(t, got)
	})

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()

		var nilPointer *int

		got := IsNil(nilPointer)
		require.True(t, got)
	})

	t.Run("nil unsafe pointer", func(t *testing.T) {
		t.Parallel()

		var nilUnsafe unsafe.Pointer

		got := IsNil(nilUnsafe)
		require.True(t, got)
	})

	// Headline case from the package doc: a nil concrete pointer stored in a
	// non-nil interface. IsNil sees through it, whereas i == nil would be false.
	t.Run("nil struct pointer wrapped in interface", func(t *testing.T) {
		t.Parallel()

		type myStruct struct{}

		var p *myStruct

		got := IsNil(any(p))
		require.True(t, got)
	})

	t.Run("non-nil nilable kinds return false", func(t *testing.T) {
		t.Parallel()

		n := 1
		ch := make(chan int)

		tests := []struct {
			name  string
			value any
		}{
			{name: "non-nil pointer", value: &n},
			{name: "non-nil map", value: map[int]int{}},
			{name: "non-nil slice", value: []int{}},
			{name: "non-nil func", value: func() {}},
			{name: "non-nil chan", value: ch},
			{name: "non-nil unsafe pointer", value: unsafe.Pointer(&n)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				require.False(t, IsNil(tt.value))
			})
		}
	})
}

func TestIsZero(t *testing.T) {
	t.Parallel()

	t.Run("not empty string", func(t *testing.T) {
		t.Parallel()

		got := IsZero("string")
		require.False(t, got)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		var emptyString string

		got := IsZero(emptyString)
		require.True(t, got)
	})

	t.Run("nil chan", func(t *testing.T) {
		t.Parallel()

		var nilChan chan int

		got := IsZero(nilChan)
		require.True(t, got)
	})

	t.Run("nil func", func(t *testing.T) {
		t.Parallel()

		var nilFunc func()

		got := IsZero(nilFunc)
		require.True(t, got)
	})

	t.Run("nil pointer to any", func(t *testing.T) {
		t.Parallel()

		var nilPtrToAny *any

		got := IsZero(nilPtrToAny)
		require.True(t, got)
	})

	t.Run("nil map", func(t *testing.T) {
		t.Parallel()

		var nilMap map[int]int

		got := IsZero(nilMap)
		require.True(t, got)
	})

	t.Run("nil slice", func(t *testing.T) {
		t.Parallel()

		var nilSlice []int

		got := IsZero(nilSlice)
		require.True(t, got)
	})

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()

		var nilPointer *int

		got := IsZero(nilPointer)
		require.True(t, got)
	})

	t.Run("non-zero int", func(t *testing.T) {
		t.Parallel()

		require.False(t, IsZero(1))
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		t.Parallel()

		n := 0

		require.False(t, IsZero(&n))
	})

	t.Run("non-nil slice", func(t *testing.T) {
		t.Parallel()

		require.False(t, IsZero([]int{}))
	})

	t.Run("non-zero struct", func(t *testing.T) {
		t.Parallel()

		require.False(t, IsZero(struct{ A int }{A: 1}))
	})

	t.Run("true bool", func(t *testing.T) {
		t.Parallel()

		require.False(t, IsZero(true))
	})

	t.Run("negative zero float is zero", func(t *testing.T) {
		t.Parallel()

		require.True(t, IsZero(math.Copysign(0, -1)))
	})

	t.Run("NaN is not zero", func(t *testing.T) {
		t.Parallel()

		require.False(t, IsZero(math.NaN()))
	})
}

func TestZero(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		v := "test"

		got := Zero(v)
		require.Empty(t, got)
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()

		var nilSlice []int

		v := []int{1, 2, 3}

		got := Zero(v)
		require.Equal(t, nilSlice, got)
	})
}

func TestPointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "int",
			value: 1,
		},
		{
			name:  "string",
			value: "test",
		},
		{
			name:  "slice",
			value: []int{1, 2},
		},
		{
			name:  "map",
			value: map[string]string{"one": "alpha", "two": "beta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Pointer(tt.value)
			require.NotNil(t, got)
			require.Equal(t, tt.value, *got)
		})
	}
}

func TestValue(t *testing.T) {
	t.Parallel()

	var nilPtr *int

	got := Value(nilPtr)
	require.Equal(t, 0, got)

	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "int",
			value: 1,
		},
		{
			name:  "string",
			value: "test",
		},
		{
			name:  "slice",
			value: []int{1, 2},
		},
		{
			name:  "map",
			value: map[string]string{"one": "alpha", "two": "beta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Value(&tt.value)
			require.Equal(t, tt.value, got)
		})
	}
}

func TestBoolToInt(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		got := BoolToInt(true)
		require.Equal(t, 1, got)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		got := BoolToInt(false)
		require.Equal(t, 0, got)
	})
}

func TestBoolToNum(t *testing.T) {
	t.Parallel()

	t.Run("int true", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, 1, BoolToNum[int](true))
	})

	t.Run("int false", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, 0, BoolToNum[int](false))
	})

	t.Run("int8 true", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, int8(1), BoolToNum[int8](true))
	})

	t.Run("uint64 false", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, uint64(0), BoolToNum[uint64](false))
	})

	t.Run("float64 true", func(t *testing.T) {
		t.Parallel()

		require.InDelta(t, 1.0, BoolToNum[float64](true), 1e-9)
	})

	t.Run("float64 false", func(t *testing.T) {
		t.Parallel()

		require.InDelta(t, 0.0, BoolToNum[float64](false), 1e-9)
	})

	t.Run("custom numeric type", func(t *testing.T) {
		t.Parallel()

		type myInt int

		require.Equal(t, myInt(1), BoolToNum[myInt](true))
	})
}
