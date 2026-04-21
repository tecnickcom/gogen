/*
Package typeutil solves a handful of small but recurring type-handling problems
in Go generics code: reliably detecting nil through interfaces, obtaining zero
values generically, dereferencing pointers safely, and converting booleans to
integers without a branch in the generated assembly.

# Problem

Go's type system has several well-known rough edges. The `v == nil` check
silently misses non-nil interfaces wrapping a nil concrete pointer. Generic
code frequently needs the zero value of an unknown type T without an instance
to copy. Dereferencing a pointer that may be nil requires a nil guard every
time. And converting a bool to 0/1 via an if/else produces less optimal code
than what the compiler can emit with the right pattern. This package centralizes
correct, idiomatic solutions to all four in one small dependency.

# Key Features

  - [IsNil]: a reflection-based nil check that correctly handles all nilable
    kinds — chan, func, interface, map, pointer, slice, unsafe pointer — as well
    as the untyped nil case. Use this wherever you receive an `any` and need a
    reliable nil test.
  - [IsZero]: a generic function that returns true when the value equals the
    zero value for its type (empty string, 0, nil pointer, false, …), without
    requiring a comparable constraint.
  - [Zero]: returns the zero value for any type T. Useful as a readable
    sentinel return in generic functions: `return typeutil.Zero(v), err`.
  - [Value]: safely dereferences a pointer, returning the zero value of T when
    the pointer is nil. Eliminates repetitive nil-guard boilerplate at every
    pointer dereference.
  - [BoolToInt]: converts a bool to 0 or 1 using the pattern that the Go
    compiler optimizes to a single MOVBLZX instruction — no branch, no
    conditional in the hot path. Preferable to an inline if/else when the
    result feeds into arithmetic or array indexing.

# Usage

	// Reliable nil detection through an interface:
	var p *MyStruct
	var i any = p
	typeutil.IsNil(i) // true, whereas i == nil is false

	// Zero value of an arbitrary generic type:
	func empty[T any]() T { return typeutil.Zero(*new(T)) }

	// Safe pointer dereference:
	var timeout *time.Duration
	d := typeutil.Value(timeout) // 0, no panic

	// Branch-free bool-to-int:
	score += typeutil.BoolToInt(isBonus) * bonusPoints

This package is ideal for any Go codebase that uses generics, works with
`any`-typed values from external sources, or needs micro-optimized arithmetic
on boolean conditions.
*/
package typeutil

import (
	"reflect"
)

// IsNil reliably detects nil including nil pointers wrapped in non-nil interfaces; works on any nilable type.
func IsNil(v any) bool {
	if v == nil {
		return true
	}

	value := reflect.ValueOf(v)

	switch value.Kind() { //nolint:exhaustive
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	}

	return false
}

// IsZero returns true if value equals type T's zero value without requiring comparable constraint.
func IsZero[T any](v T) bool {
	return reflect.ValueOf(&v).Elem().IsZero()
}

// Zero returns zero value for type T as a generic sentinel, useful for readable error returns.
func Zero[T any](_ T) T {
	var zero T
	return zero
}

// Pointer returns the address of v.
//
// Deprecated: use new() instead.
func Pointer[T any](v T) *T {
	return new(v)
}

// Value safely dereferences pointer p, returning zero value of T if nil.
func Value[T any](p *T) T {
	if IsNil(p) {
		var zero T
		return zero
	}

	return *p
}

// BoolToInt converts bool to 0 or 1 with compiler optimization to MOVBLZX; avoids branch in hot paths.
func BoolToInt(b bool) int {
	var i int

	if b {
		i = 1
	} else {
		i = 0
	}

	return i
}
