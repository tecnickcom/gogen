package logutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value   string
		want    LogFormat
		wantErr bool
	}{
		{
			value:   "",
			want:    FormatNone,
			wantErr: true,
		},
		{
			value:   "xml",
			want:    FormatNone,
			wantErr: true,
		},
		{
			value: "none",
			want:  FormatNone,
		},
		{
			value: "discard",
			want:  FormatNone,
		},
		{
			value: "noop",
			want:  FormatNone,
		},
		{
			value: "console",
			want:  FormatConsole,
		},
		{
			value: "json",
			want:  FormatJSON,
		},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			got, err := ParseFormat(tt.value)
			if tt.wantErr {
				require.Error(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value LogFormat
		want  bool
	}{
		{
			name:  "invalid",
			value: -16,
			want:  false,
		},
		{
			name:  "none",
			value: FormatNone,
			want:  true,
		},
		{
			name:  "json",
			value: FormatJSON,
			want:  true,
		},
		{
			name:  "console",
			value: FormatConsole,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ValidFormat(tt.value)

			require.Equal(t, tt.want, got)
		})
	}
}
