package typeutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      []byte
		key       []byte
		wantEmpty bool
		wantErr   bool
	}{
		{
			name:    "empty key",
			data:    []byte(""),
			key:     []byte(""),
			wantErr: true,
		},
		{
			name:    "wrong key length",
			data:    []byte(""),
			key:     []byte("0123"),
			wantErr: true,
		},
		{
			name:    "ok nil data",
			data:    nil,
			key:     []byte("01234567890123456789012345678901"),
			wantErr: false,
		},
		{
			name:    "ok key 16 bytes",
			data:    []byte("text to encrypt 16"),
			key:     []byte("abcdefghijklmnop"),
			wantErr: false,
		},
		{
			name:    "ok key 24 bytes",
			data:    []byte("text to encrypt 24"),
			key:     []byte("abcdefghijklmnopqrstuvwx"),
			wantErr: false,
		},
		{
			name:    "ok key 32 bytes",
			data:    []byte("text to encrypt 32"),
			key:     []byte("abcdefghijklmnopqrstuvwxyz012345"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := Encrypt(tt.key, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, enc)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, enc)

			dec, err := Decrypt(tt.key, enc)

			require.NoError(t, err)
			require.Equal(t, tt.data, dec)
		})
	}
}

func TestDecryptErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		key  []byte
	}{
		{
			name: "empty key",
			data: []byte(""),
			key:  []byte(""),
		},
		{
			name: "wrong key length",
			data: []byte(""),
			key:  []byte("0123"),
		},
		{
			name: "nil data",
			data: nil,
			key:  []byte("01234567890123456789012345678901"),
		},
		{
			name: "invalid data",
			data: []byte("123"),
			key:  []byte("abcdefghijklmnop"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec, err := Decrypt(tt.key, tt.data)

			require.Error(t, err)
			require.Nil(t, dec)
		})
	}
}

func Test_ByteEncryptAny_ByteDecryptAny(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name    string
		data    TestData
		key     []byte
		wantErr bool
	}{
		{
			name:    "empty key",
			data:    TestData{},
			key:     []byte(""),
			wantErr: true,
		},
		{
			name:    "wrong key length",
			data:    TestData{},
			key:     []byte("0123"),
			wantErr: true,
		},
		{
			name:    "ok key 16 bytes",
			data:    TestData{Alpha: "text to encrypt 16", Beta: -6, Gamma: 0.1234},
			key:     []byte("abcdefghijklmnop"),
			wantErr: false,
		},
		{
			name:    "ok key 24 bytes",
			data:    TestData{Alpha: "text to encrypt 24", Beta: 24, Gamma: -0.1234},
			key:     []byte("abcdefghijklmnopqrstuvwx"),
			wantErr: false,
		},
		{
			name:    "ok key 32 bytes",
			data:    TestData{Alpha: "text to encrypt 32", Beta: 32, Gamma: 0.1234},
			key:     []byte("abcdefghijklmnopqrstuvwxyz012345"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := ByteEncryptAny(tt.key, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, enc)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, enc)

			var data TestData

			err = ByteDecryptAny(tt.key, enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.data, data)
		})
	}
}

func TestByteEncryptAny_Error(t *testing.T) {
	t.Parallel()

	enc, err := ByteEncryptAny([]byte(""), make(chan int))

	require.Error(t, err)
	require.Nil(t, enc)
}

func TestByteDecryptAny_Errors(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name string
		enc  []byte
		key  []byte
	}{
		{
			name: "empty data",
			enc:  []byte(""),
			key:  []byte("abcdefghijklmnop"),
		},
		{
			name: "bad base64 data",
			enc:  []byte("~~~"),
			key:  []byte("abcdefghijklmnopqrstuvwxyz012345"),
		},
		{
			name: "empty key",
			enc:  []byte("dGVzdA=="),
			key:  []byte(""),
		},
		{
			name: "bad encryption data",
			enc:  []byte("dGVzdA=="),
			key:  []byte("abcdefghijklmnopqrstuvwxyz012345"),
		},
		{
			name: "bad gob data",
			enc:  []byte{47, 47, 51, 82, 121, 89, 90, 68, 54, 86, 65, 66, 83, 51, 72, 82, 81, 112, 51, 120, 111, 57, 67, 97, 56, 120, 49, 78, 105, 102, 52, 116, 114, 113, 90, 68, 109, 53, 99, 98, 90, 120, 103, 80, 79, 78, 80, 65, 56, 75, 72, 84, 49, 53, 76, 69, 54, 120, 119, 99, 51, 103, 61, 61},
			key:  []byte("abcdefghijklmnopqrstuvwxyz012345"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data TestData

			err := ByteDecryptAny(tt.key, tt.enc, &data)
			require.Error(t, err)
		})
	}
}

func Test_EncryptAny_DecryptAny(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
		Gamma float32
	}

	tests := []struct {
		name    string
		data    TestData
		key     []byte
		wantErr bool
	}{
		{
			name:    "empty key",
			data:    TestData{},
			key:     []byte(""),
			wantErr: true,
		},
		{
			name:    "wrong key length",
			data:    TestData{},
			key:     []byte("0123"),
			wantErr: true,
		},
		{
			name:    "ok key 16 bytes",
			data:    TestData{Alpha: "text to encrypt 16", Beta: -6, Gamma: 0.1234},
			key:     []byte("abcdefghijklmnop"),
			wantErr: false,
		},
		{
			name:    "ok key 24 bytes",
			data:    TestData{Alpha: "text to encrypt 24", Beta: 24, Gamma: -0.1234},
			key:     []byte("abcdefghijklmnopqrstuvwx"),
			wantErr: false,
		},
		{
			name:    "ok key 32 bytes",
			data:    TestData{Alpha: "text to encrypt 32", Beta: 32, Gamma: 0.1234},
			key:     []byte("abcdefghijklmnopqrstuvwxyz012345"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc, err := EncryptAny(tt.key, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, enc)

				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, enc)

			var data TestData

			err = DecryptAny(tt.key, enc, &data)

			require.NoError(t, err)
			require.Equal(t, tt.data, data)
		})
	}
}
