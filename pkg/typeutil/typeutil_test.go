// Package typeutil contains a collection of type-related utility functions.
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
		var nilInterface *interface{}

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
		var nilInterface *interface{}

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

func TestEncode(t *testing.T) {
	t.Parallel()

	var (
		nilPointer *int
		nilChan    chan int
	)

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name      string
		value     any
		wantEmpty bool
		wantErr   bool
	}{
		{
			name:      "unsupported type",
			value:     make(chan int),
			wantEmpty: true,
			wantErr:   true,
		},
		{
			name:      "nil pointer",
			value:     nilPointer,
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name:      "nil chan",
			value:     nilChan,
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name:      "success empty string",
			value:     "",
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name:      "success with string",
			value:     "test",
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name:      "success with int",
			value:     123,
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name:      "success with empty struct",
			value:     &TestData{},
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name:      "success with struct",
			value:     &TestData{Alpha: "abc123", Beta: -375},
			wantEmpty: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Encode(tt.value)

			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.wantEmpty, got == "")
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name    string
		msg     string
		want    TestData
		wantErr bool
	}{
		{
			name:    "success",
			msg:     "Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA==",
			want:    TestData{Alpha: "abc123", Beta: -375},
			wantErr: false,
		},
		{
			name:    "invalid base64",
			msg:     "你好世界",
			want:    TestData{},
			wantErr: true,
		},
		{
			name:    "empty",
			msg:     "",
			want:    TestData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data TestData

			err := Decode(tt.msg, &data)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.Alpha, data.Alpha)
			require.Equal(t, tt.want.Beta, data.Beta)
		})
	}
}
