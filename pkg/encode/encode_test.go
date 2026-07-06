package encode

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockWriter struct{}

func (w *mockWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

func Test_Base64EncodeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "non-empty string",
			in:   "test string",
			want: "dGVzdCBzdHJpbmc=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc := Base64EncodeString(tt.in)

			require.Equal(t, tt.want, enc)
		})
	}
}

func Test_Base64DecodeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{
			name:    "empty string",
			in:      "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "non-empty string",
			in:      "dGVzdCBzdHJpbmc=",
			want:    "test string",
			wantErr: false,
		},
		{
			name:    "invalid base64",
			in:      "你好世界", //nolint:gosmopolitan
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec, err := Base64DecodeString(tt.in)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, dec)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, string(dec))
		})
	}
}

func Test_Base64EncodeDecodeString_RoundTrip(t *testing.T) {
	t.Parallel()

	const in = "arbitrary+bytes/with=padding and symbols"

	dec, err := Base64DecodeString(Base64EncodeString(in))

	require.NoError(t, err)
	require.Equal(t, in, string(dec))
}

func Test_Base64Encoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value io.Writer
	}{
		{
			name:  "bytes buffer",
			value: new(bytes.Buffer),
		},
		{
			name:  "mock writer",
			value: &mockWriter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc := Base64Encoder(tt.value)

			require.NotNil(t, enc)
		})
	}
}

func Test_Base64Decoder(t *testing.T) {
	t.Parallel()

	dec := Base64Decoder(strings.NewReader("dGVzdCBzdHJpbmc="))
	require.NotNil(t, dec)

	got, err := io.ReadAll(dec)

	require.NoError(t, err)
	require.Equal(t, "test string", string(got))
}

type mockWriteCloserCloseError struct{}

func (w *mockWriteCloserCloseError) Write(_ []byte) (int, error) {
	return 0, nil
}

func (w *mockWriteCloserCloseError) Close() error {
	return errors.New("close error")
}

type mockWriteCloserCloseTracker struct {
	closed bool
}

func (w *mockWriteCloserCloseTracker) Write(_ []byte) (int, error) {
	return 0, nil
}

func (w *mockWriteCloserCloseTracker) Close() error {
	w.closed = true
	return nil
}

func Test_GobEncoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    any
		enc     io.WriteCloser
		wantErr bool
	}{
		{
			name:    "close error",
			data:    2,
			enc:     &mockWriteCloserCloseError{},
			wantErr: true,
		},
		{
			name:    "writer error",
			data:    3,
			enc:     Base64Encoder(&mockWriter{}),
			wantErr: true,
		},
		{
			name:    "data error",
			data:    make(chan int),
			enc:     Base64Encoder(new(bytes.Buffer)),
			wantErr: true,
		},
		{
			name:    "success",
			data:    5,
			enc:     Base64Encoder(new(bytes.Buffer)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := GobEncoder(tt.enc, tt.data)

			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_JSONEncoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    any
		enc     io.WriteCloser
		wantErr bool
	}{
		{
			name:    "close error",
			data:    2,
			enc:     &mockWriteCloserCloseError{},
			wantErr: true,
		},
		{
			name:    "writer error",
			data:    3,
			enc:     Base64Encoder(&mockWriter{}),
			wantErr: true,
		},
		{
			name:    "data error",
			data:    make(chan int),
			enc:     Base64Encoder(new(bytes.Buffer)),
			wantErr: true,
		},
		{
			name:    "success",
			data:    5,
			enc:     Base64Encoder(new(bytes.Buffer)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := JSONEncoder(tt.enc, tt.data)

			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_JsonEncoder(t *testing.T) {
	t.Parallel()

	err := JsonEncoder(Base64Encoder(new(bytes.Buffer)), 5) // exercises the deprecated alias

	require.NoError(t, err)
}

func Test_GobEncoder_ClosesEncoderOnError(t *testing.T) {
	t.Parallel()

	enc := &mockWriteCloserCloseTracker{}

	err := GobEncoder(enc, make(chan int)) // unencodable value

	require.Error(t, err)
	require.True(t, enc.closed, "encoder must be closed on the encode error path")
}

func Test_JSONEncoder_ClosesEncoderOnError(t *testing.T) {
	t.Parallel()

	enc := &mockWriteCloserCloseTracker{}

	err := JSONEncoder(enc, make(chan int)) // unencodable value

	require.Error(t, err)
	require.True(t, enc.closed, "encoder must be closed on the encode error path")
}

func TestByteEncodeDecode(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name  string
		value TestData
	}{
		{
			name:  "empty",
			value: TestData{},
		},
		{
			name:  "full",
			value: TestData{Alpha: "abc1234", Beta: -3756, Gamma: 0.1234},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := ByteEncode(tt.value)

			require.NoError(t, err)

			var data TestData

			err = ByteDecode(enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.value.Alpha, data.Alpha)
			require.Equal(t, tt.value.Beta, data.Beta)
			require.InDelta(t, tt.value.Gamma, data.Gamma, 0.001)
		})
	}
}

func TestEncode(t *testing.T) {
	t.Parallel()

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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := Encode(tt.value)

			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.wantEmpty, enc == "")

			benc, berr := ByteEncode(tt.value)

			require.Equal(t, tt.wantErr, berr != nil)
			require.Equal(t, tt.wantEmpty, benc == nil)
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
			msg:     "你好世界", //nolint:gosmopolitan
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

			berr := ByteDecode([]byte(tt.msg), &data)

			if tt.wantErr {
				require.Error(t, berr)
				return
			}

			require.NoError(t, berr)
			require.Equal(t, tt.want.Alpha, data.Alpha)
			require.Equal(t, tt.want.Beta, data.Beta)
		})
	}
}

func TestEncodeDecode(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name  string
		value TestData
	}{
		{
			name:  "empty",
			value: TestData{},
		},
		{
			name:  "full",
			value: TestData{Alpha: "abc1234", Beta: -3756, Gamma: 0.1234},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := Encode(tt.value)

			require.NoError(t, err)

			var data TestData

			err = Decode(enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.value.Alpha, data.Alpha)
			require.Equal(t, tt.value.Beta, data.Beta)
			require.InDelta(t, tt.value.Gamma, data.Gamma, 0.001)
		})
	}
}

func TestSerialize(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name    string
		value   any
		want    string
		wantErr bool
	}{
		{
			name:    "unsupported type",
			value:   make(chan int),
			want:    "",
			wantErr: true,
		},
		{
			name:    "success empty string",
			value:   "",
			want:    "IiIK",
			wantErr: false,
		},
		{
			name:    "success with string",
			value:   "test",
			want:    "InRlc3QiCg==",
			wantErr: false,
		},
		{
			name:    "success with int",
			value:   123,
			want:    "MTIzCg==",
			wantErr: false,
		},
		{
			name:    "success with empty struct",
			value:   &TestData{},
			want:    "eyJBbHBoYSI6IiIsIkJldGEiOjB9Cg==",
			wantErr: false,
		},
		{
			name:    "success with struct",
			value:   &TestData{Alpha: "abc123", Beta: -375},
			want:    "eyJBbHBoYSI6ImFiYzEyMyIsIkJldGEiOi0zNzV9Cg==",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Serialize(tt.value)

			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.want, got)

			bgot, berr := ByteSerialize(tt.value)

			require.Equal(t, tt.wantErr, berr != nil)
			require.Equal(t, tt.want, string(bgot))
		})
	}
}

func TestDeserialize(t *testing.T) {
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
			msg:     "eyJBbHBoYSI6ImFiYzEyMyIsIkJldGEiOi0zNzV9Cg==",
			want:    TestData{Alpha: "abc123", Beta: -375},
			wantErr: false,
		},
		{
			name:    "invalid base64",
			msg:     "你好世界", //nolint:gosmopolitan
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data TestData

			err := Deserialize(tt.msg, &data)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.Alpha, data.Alpha)
			require.Equal(t, tt.want.Beta, data.Beta)

			berr := ByteDeserialize([]byte(tt.msg), &data)

			if tt.wantErr {
				require.Error(t, berr)
				return
			}

			require.NoError(t, berr)
			require.Equal(t, tt.want.Alpha, data.Alpha)
			require.Equal(t, tt.want.Beta, data.Beta)
		})
	}
}

func TestSerializeDeserialize(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name  string
		value TestData
	}{
		{
			name:  "empty",
			value: TestData{},
		},
		{
			name:  "full",
			value: TestData{Alpha: "abc1235", Beta: -3755, Gamma: 0.1235},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := Serialize(tt.value)

			require.NoError(t, err)

			var data TestData

			err = Deserialize(enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.value.Alpha, data.Alpha)
			require.Equal(t, tt.value.Beta, data.Beta)
			require.InDelta(t, tt.value.Gamma, data.Gamma, 0.001)
		})
	}
}

func TestByteSerializeDeserialize(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name  string
		value TestData
	}{
		{
			name:  "empty",
			value: TestData{},
		},
		{
			name:  "full",
			value: TestData{Alpha: "abc1235", Beta: -3755, Gamma: 0.1235},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := ByteSerialize(tt.value)

			require.NoError(t, err)

			var data TestData

			err = ByteDeserialize(enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.value.Alpha, data.Alpha)
			require.Equal(t, tt.value.Beta, data.Beta)
			require.InDelta(t, tt.value.Gamma, data.Gamma, 0.001)
		})
	}
}

func TestBufferEncodeDecode(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name  string
		value TestData
	}{
		{
			name:  "empty",
			value: TestData{},
		},
		{
			name:  "full",
			value: TestData{Alpha: "abc1234", Beta: -3756, Gamma: 0.1234},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := BufferEncode(tt.value)

			require.NoError(t, err)

			var data TestData

			err = BufferDecode(enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.value.Alpha, data.Alpha)
			require.Equal(t, tt.value.Beta, data.Beta)
			require.InDelta(t, tt.value.Gamma, data.Gamma, 0.001)
		})
	}
}

func TestBufferSerializeDeserialize(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name  string
		value TestData
	}{
		{
			name:  "empty",
			value: TestData{},
		},
		{
			name:  "full",
			value: TestData{Alpha: "abc1235", Beta: -3755, Gamma: 0.1235},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := BufferSerialize(tt.value)

			require.NoError(t, err)

			var data TestData

			err = BufferDeserialize(enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.value.Alpha, data.Alpha)
			require.Equal(t, tt.value.Beta, data.Beta)
			require.InDelta(t, tt.value.Gamma, data.Gamma, 0.001)
		})
	}
}
