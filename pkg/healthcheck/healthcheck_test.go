// +build unit

package healthcheck_test

import (
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/healthcheck"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status healthcheck.Status
		want   string
	}{
		{
			name:   "Unavailable is N/A",
			status: healthcheck.Unavailable,
			want:   "N/A",
		},
		{
			name:   "OK is OK",
			status: healthcheck.OK,
			want:   "OK",
		},
		{
			name:   "Err is ERR",
			status: healthcheck.Err,
			want:   "ERR",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
