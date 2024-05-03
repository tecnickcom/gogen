/*
Package typeutil contains a collection of type-related utility functions.

This package provides a set of utility functions and definitions to work with generic types in Go.
*/
package typeutil

import (
	"reflect"
)

// IsNil returns true if the input value is nil.
func IsNil(v any) bool {
	if v == nil {
		return true
	}

	value := reflect.ValueOf(v)

	//nolint:exhaustive
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	}

	return false
}

// IsZero returns true if the input value is equal to the zero instance (e.g. empty string, 0 int, nil pointer).
func IsZero[T any](v T) bool {
	return reflect.ValueOf(&v).Elem().IsZero()
}

// Zero returns the zero instance (e.g. empty string, 0 int, nil pointer).
func Zero[T any](_ T) T {
	var zero T
	return zero
}

// Pointer returns the address of v.
func Pointer[T any](v T) *T {
	return &v
}

// Value returns the value of the provided pointer or the type default (zero value) if nil.
func Value[T any](p *T) T {
	if IsNil(p) {
		var zero T
		return zero
	}

	return *p
}
