// Package typeutil contains a collection of type-related utility functions.
package typeutil

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
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

func base64Encoder(w io.Writer) io.WriteCloser {
	return base64.NewEncoder(base64.StdEncoding, w)
}

func gobEncode(enc io.WriteCloser, data any) error {
	if err := gob.NewEncoder(enc).Encode(data); err != nil {
		return fmt.Errorf("gob: %w", err)
	}

	return enc.Close() //nolint:wrapcheck
}

func jsonEncode(enc io.WriteCloser, data any) error {
	if err := json.NewEncoder(enc).Encode(data); err != nil {
		return fmt.Errorf("JSON: %w", err)
	}

	return enc.Close() //nolint:wrapcheck
}

// Encode encodes the input data to gob/base64 format into a string.
func Encode(data any) (string, error) {
	var buf bytes.Buffer
	if err := gobEncode(base64Encoder(&buf), data); err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}

	return buf.String(), nil
}

// Decode decodes a message encoded with the Encode function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Decode(msg string, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(msg))
	if err := gob.NewDecoder(decoder).Decode(data); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

// Serialize encodes the input data to JSON/base64 format into a string.
func Serialize(data any) (string, error) {
	var buf bytes.Buffer
	if err := jsonEncode(base64Encoder(&buf), data); err != nil {
		return "", fmt.Errorf("serialize: %w", err)
	}

	return buf.String(), nil
}

// Deserialize decodes a message encoded with the Serialize function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Deserialize(msg string, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(msg))
	if err := json.NewDecoder(decoder).Decode(data); err != nil {
		return fmt.Errorf("deserialize: %w", err)
	}

	return nil
}
