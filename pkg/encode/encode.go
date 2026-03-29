/*
Package encode provides a collection of helper functions for safe serialization
and deserialization across system boundaries such as databases, queues, caches,
and RPC payloads.

It solves the common problem of reliably encoding Go values into transport-safe
formats and decoding them back without losing structure or introducing unsafe
byte streams.

The package supports two main modes:

- Gob + Base64 encoding for arbitrary Go values
- JSON + Base64 encoding for interoperable text-based payloads

Top features:

- encode/decode helpers for strings, byte slices, buffers, and io.Reader/io.Writer flows
- consistent Base64 wrapping to keep binary payloads safe for text-only channels
- explicit error wrapping at encode/decode/serialize/deserialize boundaries
- parallel API families for Gob (Encode/Decode) and JSON (Serialize/Deserialize)
- support for arbitrary serializable Go values, with clear failures for unsupported types

Benefits:

  - reduce boilerplate for common serialization flows
  - avoid accidental misuse of raw binary in text-only transports
  - simplify data exchange between services and storage layers
*/
package encode

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Base64EncodeString returns the Base64 representation of s.
//
// Use it when text-safe transport is required for arbitrary bytes.
func Base64EncodeString(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Encoder wraps w with a streaming Base64 encoder.
//
// The caller should close the returned writer to flush buffered output.
func Base64Encoder(w io.Writer) io.WriteCloser {
	return base64.NewEncoder(base64.StdEncoding, w)
}

// GobEncoder gob-encodes data into enc and closes enc.
//
// It centralizes gob encoding and close handling for stream-based pipelines.
func GobEncoder(enc io.WriteCloser, data any) error {
	err := gob.NewEncoder(enc).Encode(data)
	if err != nil {
		return fmt.Errorf("gob: %w", err)
	}

	return enc.Close() //nolint:wrapcheck
}

// JsonEncoder JSON-encodes data into enc and closes enc.
//
// It is useful for stream-friendly JSON pipelines with explicit close semantics.
func JsonEncoder(enc io.WriteCloser, data any) error {
	err := json.NewEncoder(enc).Encode(data)
	if err != nil {
		return fmt.Errorf("JSON: %w", err)
	}

	return enc.Close() //nolint:wrapcheck
}

// ByteEncode encodes data as gob+Base64 bytes.
//
// This format is convenient for binary channels while still staying text-safe.
func ByteEncode(data any) ([]byte, error) {
	buf, err := BufferEncode(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ByteDecode decodes gob+Base64 bytes into data.
//
// data must be a pointer to the destination type.
func ByteDecode(msg []byte, data any) error {
	return BufferDecode(bytes.NewReader(msg), data)
}

// Encode encodes data as gob+Base64 string.
//
// It is useful for storing typed payloads in text-only fields.
func Encode(data any) (string, error) {
	buf, err := BufferEncode(data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Decode decodes a gob+Base64 string into data.
//
// data must be a pointer to the destination type.
func Decode(msg string, data any) error {
	return BufferDecode(strings.NewReader(msg), data)
}

// Serialize encodes data as JSON+Base64 string.
//
// Choose this over Encode when interoperability with non-Go systems matters.
func Serialize(data any) (string, error) {
	buf, err := BufferSerialize(data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Deserialize decodes a JSON+Base64 string into data.
//
// data must be a pointer to the destination type.
func Deserialize(msg string, data any) error {
	return BufferDeserialize(strings.NewReader(msg), data)
}

// ByteSerialize encodes data as JSON+Base64 bytes.
func ByteSerialize(data any) ([]byte, error) {
	buf, err := BufferSerialize(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ByteDeserialize decodes JSON+Base64 bytes into data.
//
// data must be a pointer to the destination type.
func ByteDeserialize(msg []byte, data any) error {
	return BufferDeserialize(bytes.NewReader(msg), data)
}

// BufferEncode encodes data as gob+Base64 and returns an in-memory buffer.
//
// It is a low-allocation building block reused by Encode and ByteEncode.
func BufferEncode(data any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}

	err := GobEncoder(Base64Encoder(buf), data)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return buf, nil
}

// BufferDecode reads gob+Base64 content from reader into data.
//
// data must be a pointer to the destination type.
func BufferDecode(reader io.Reader, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, reader)

	err := gob.NewDecoder(decoder).Decode(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

// BufferSerialize encodes data as JSON+Base64 and returns an in-memory buffer.
//
// It is a low-allocation building block reused by Serialize and ByteSerialize.
func BufferSerialize(data any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}

	err := JsonEncoder(Base64Encoder(buf), data)
	if err != nil {
		return nil, fmt.Errorf("serialize: %w", err)
	}

	return buf, nil
}

// BufferDeserialize reads JSON+Base64 content from reader into data.
//
// data must be a pointer to the destination type.
func BufferDeserialize(reader io.Reader, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, reader)

	err := json.NewDecoder(decoder).Decode(data)
	if err != nil {
		return fmt.Errorf("deserialize: %w", err)
	}

	return nil
}
