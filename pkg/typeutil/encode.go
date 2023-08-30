package typeutil

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

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
