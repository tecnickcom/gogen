// Package typeutil contains a collection of type-related utility functions.
package typeutil

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
)

// IsNil returns true if the input value is nil.
func IsNil[T any](v T) bool {
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

// Encode encodes and serialize the input data to a gob/base64 string.
func Encode[T any](data T) (string, error) {
	var buf bytes.Buffer

	if IsNil(data) {
		return "", nil
	}

	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return "", fmt.Errorf("failed to gob-encode: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decode decodes a message encoded with the Encode function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Decode[T any](msg string, data T) error {
	s, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return fmt.Errorf("failed to base64-decode: %w", err)
	}

	if err := gob.NewDecoder(bytes.NewBuffer(s)).Decode(data); err != nil {
		return fmt.Errorf("failed to gob-decode: %w", err)
	}

	return nil
}

// Serialize encodes the input data to a string that can be used for object comparison (json/base64).
func Serialize[T any](data T) (string, error) {
	var buf bytes.Buffer

	if IsNil(data) {
		return "", nil
	}

	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", fmt.Errorf("failed to json-encode: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Deserialize decodes a message encoded with the Serialize function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Deserialize[T any](msg string, data T) error {
	s, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return fmt.Errorf("failed to base64-decode: %w", err)
	}

	if err := json.NewDecoder(bytes.NewBuffer(s)).Decode(data); err != nil {
		return fmt.Errorf("failed to json-decode: %w", err)
	}

	return nil
}
