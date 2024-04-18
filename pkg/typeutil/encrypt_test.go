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
