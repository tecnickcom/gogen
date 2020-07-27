// +build unit

package httputil_test

import (
	"net/http"
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/stretchr/testify/require"
)

func TestHeaderOrDefault(t *testing.T) {
	t.Parallel()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	r.Header.Add("set-header", "test")

	v1 := httputil.HeaderOrDefault(r, "unset-header", "default")
	require.Equal(t, "default", v1)

	v2 := httputil.HeaderOrDefault(r, "set-header", "default")
	require.Equal(t, "test", v2)
}
