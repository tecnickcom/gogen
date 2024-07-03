package typeutil

import (
	"testing"

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

	t.Run("nil interface", func(t *testing.T) {
		t.Parallel()

		var nilInterface *any

		got := IsNil(nilInterface)
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

	t.Run("nil interface", func(t *testing.T) {
		t.Parallel()

		var nilInterface *any

		got := IsZero(nilInterface)
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
}

func TestZero(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		v := "test"

		got := Zero(v)
		require.Equal(t, "", got)
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
