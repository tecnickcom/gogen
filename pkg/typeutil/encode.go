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

func bufferEncode(data any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	if err := gobEncode(base64Encoder(buf), data); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return buf, nil
}

// ByteEncode encodes the input data to gob/base64 format into a byte slice.
func ByteEncode(data any) ([]byte, error) {
	buf, err := bufferEncode(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Encode encodes the input data to gob/base64 format into a string.
func Encode(data any) (string, error) {
	buf, err := bufferEncode(data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func bufferDecode(reader io.Reader, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, reader)
	if err := gob.NewDecoder(decoder).Decode(data); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

// ByteDecode decodes a byte slice message encoded with the ByteEncode function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func ByteDecode(msg []byte, data any) error {
	return bufferDecode(bytes.NewReader(msg), data)
}

// Decode decodes a string message encoded with the Encode function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Decode(msg string, data any) error {
	return bufferDecode(strings.NewReader(msg), data)
}

func bufferSerialize(data any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	if err := jsonEncode(base64Encoder(buf), data); err != nil {
		return nil, fmt.Errorf("serialize: %w", err)
	}

	return buf, nil
}

// ByteSerialize encodes the input data to JSON/base64 format into a byte slice.
func ByteSerialize(data any) ([]byte, error) {
	buf, err := bufferSerialize(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Serialize encodes the input data to JSON/base64 format into a string.
func Serialize(data any) (string, error) {
	buf, err := bufferSerialize(data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func bufferDeserialize(reader io.Reader, data any) error {
	decoder := base64.NewDecoder(base64.StdEncoding, reader)
	if err := json.NewDecoder(decoder).Decode(data); err != nil {
		return fmt.Errorf("deserialize: %w", err)
	}

	return nil
}

// ByteDeserialize decodes a string message encoded with the Serialize function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func ByteDeserialize(msg []byte, data any) error {
	return bufferDeserialize(bytes.NewReader(msg), data)
}

// Deserialize decodes a byte slice message encoded with the Serialize function to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func Deserialize(msg string, data any) error {
	return bufferDeserialize(strings.NewReader(msg), data)
}
