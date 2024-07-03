package passwordpwned

import (
	"testing"

	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "fails with invalid character in URL",
			opts:    []Option{WithURL("http://invalid-url.domain.invalid\u007F")},
			wantErr: true,
		},
		{
			name:    "succeeds with defaults",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.opts = append(tt.opts, WithRetryAttempts(1))
			c, err := New(tt.opts...)

			if tt.wantErr {
				require.Nil(t, c, "New() returned client should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, c, "New() returned client should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func TestClient_HealthCheck(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err)

	err = c.HealthCheck(testutil.Context())
	require.NoError(t, err)
}
