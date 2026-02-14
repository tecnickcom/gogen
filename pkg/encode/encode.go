/*
Package encode provides a collection of functions for safe serialization and
deserialization of data between different systems, such as databases, queues,
and caches.
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

// Base64EncodeString encodes a string in Base64.
func Base64EncodeString(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Encoder wraps an io.Writer with a base64 encoder.
func Base64Encoder(w io.Writer) io.WriteCloser {
	return base64.NewEncoder(base64.StdEncoding, w)
}

// GobEncoder encodes data using gob encoding and writes it to the provided io.WriteCloser.
func GobEncoder(enc io.WriteCloser, data any) error {
	err := gob.NewEncoder(enc).Encode(data)
	if err != nil {
		return fmt.Errorf("gob: %w", err)
	}

	return enc.Close() //nolint:wrapcheck
}

// JsonEncoder encodes data using JSON encoding and writes it to the provided io.WriteCloser.
func JsonEncoder(enc io.WriteCloser, data any) error {
	err := json.NewEncoder(enc).Encode(data)
	if err != nil {
		return fmt.Errorf("JSON: %w", err)
	}

	return enc.Close() //nolint:wrapcheck
}

// ByteEncode encodes the input data to gob+base64 byte slice.
func ByteEncode(data any) ([]byte, error) {
	buf, err := BufferEncode(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ByteDecode decodes a byte slice message encoded with the ByteEncode function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func ByteDecode(msg []byte, data any) error {
	return BufferDecode(bytes.NewReader(msg), data)
}

// Encode encodes the input data to gob+base64 string.
func Encode(data any) (string, error) {
	buf, err := BufferEncode(data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Decode decodes a string message encoded with the Encode function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Decode(msg string, data any) error {
	return BufferDecode(strings.NewReader(msg), data)
}

// Serialize encodes the input data to JSON+base64 string.
func Serialize(data any) (string, error) {
	buf, err := BufferSerialize(data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Deserialize decodes a byte slice message encoded with the Serialize function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Deserialize(msg string, data any) error {
	return BufferDeserialize(strings.NewReader(msg), data)
}

// ByteSerialize encodes the input data to JSON+base64 byte slice.
func ByteSerialize(data any) ([]byte, error) {
	buf, err := BufferSerialize(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ByteDeserialize decodes a string message encoded with the Serialize function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func ByteDeserialize(msg []byte, data any) error {
	return BufferDeserialize(bytes.NewReader(msg), data)
}

// BufferEncode encodes the input data to gob+base64 and returns it as a bytes.Buffer.
func BufferEncode(data any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}

	err := GobEncoder(Base64Encoder(buf), data)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return buf, nil
}

// BufferDecode decodes gob+base64 data from the provided io.Reader into the provided data object.
func BufferDecode(reader io.Reader, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, reader)

	err := gob.NewDecoder(decoder).Decode(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

// BufferSerialize encodes the input data to JSON+base64 and returns it as a bytes.Buffer.
func BufferSerialize(data any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}

	err := JsonEncoder(Base64Encoder(buf), data)
	if err != nil {
		return nil, fmt.Errorf("serialize: %w", err)
	}

	return buf, nil
}

// BufferDeserialize decodes JSON+base64 data from the provided io.Reader into the provided data object.
func BufferDeserialize(reader io.Reader, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, reader)

	err := json.NewDecoder(decoder).Decode(data)
	if err != nil {
		return fmt.Errorf("deserialize: %w", err)
	}

	return nil
}
