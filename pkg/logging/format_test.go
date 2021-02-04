package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    Format
		wantErr bool
	}{
		{
			name:    "fails with invalid string",
			value:   "xml",
			want:    noFormat,
			wantErr: true,
		},
		{
			name:    "succeed with 'console'",
			value:   "console",
			want:    ConsoleFormat,
			wantErr: false,
		},
		{
			name:    "succeed with 'json'",
			value:   "json",
			want:    JSONFormat,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseFormat(tt.value)
			if tt.wantErr {
				require.NotNil(t, err, "ParseFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
			require.Equal(t, tt.want, got, "ParseFormat() got = %v, want %v", got, tt.want)
		})
	}
}
