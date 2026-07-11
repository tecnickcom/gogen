package encode_test

import (
	"testing"

	"github.com/tecnickcom/nurago/pkg/encode"
)

type benchData struct {
	Alpha string
	Beta  int
	Gamma float32
	Delta []string
}

// benchValue is a representative payload exercising strings, ints, floats, and slices.
func benchValue() benchData {
	return benchData{
		Alpha: "the quick brown fox jumps over the lazy dog",
		Beta:  -123456,
		Gamma: 3.14159,
		Delta: []string{"one", "two", "three", "four", "five"},
	}
}

const benchString = "the quick brown fox jumps over the lazy dog"

func BenchmarkBase64EncodeString(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = encode.Base64EncodeString(benchString)
	}
}

func BenchmarkBase64DecodeString(b *testing.B) {
	enc := encode.Base64EncodeString(benchString)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = encode.Base64DecodeString(enc)
	}
}

func BenchmarkEncode(b *testing.B) {
	val := benchValue()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = encode.Encode(val)
	}
}

func BenchmarkDecode(b *testing.B) {
	enc, err := encode.Encode(benchValue())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		var out benchData

		_ = encode.Decode(enc, &out)
	}
}

func BenchmarkSerialize(b *testing.B) {
	val := benchValue()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = encode.Serialize(val)
	}
}

func BenchmarkDeserialize(b *testing.B) {
	enc, err := encode.Serialize(benchValue())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		var out benchData

		_ = encode.Deserialize(enc, &out)
	}
}
