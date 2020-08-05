// +build unit

package requestid

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromContext(t *testing.T) {
	t.Parallel()

	// Context without request id, should return given default
	id1 := FromContext(context.Background(), "default-12345")
	require.NotEmpty(t, id1)
	require.Equal(t, "default-12345", id1)

	// Context with request id, should return the existing value
	ctx := WithRequestID(context.Background(), "context-12345")
	id2 := FromContext(ctx, "default-67890")
	require.NotEmpty(t, id2)
	require.Equal(t, "context-12345", id2)
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()

	ctx := WithRequestID(context.Background(), "test-123456")

	el1 := FromContext(ctx, "default-123456")
	require.Equal(t, el1, "test-123456")

	// do not override with the same id
	ctx1 := WithRequestID(ctx, "test-123456")
	require.Equal(t, ctx, ctx1)

	// do not override with other id
	ctx2 := WithRequestID(ctx, "other-123456")
	require.Equal(t, ctx, ctx2)
}

func TestFromHTTPRequest(t *testing.T) {
	t.Parallel()

	// header not set should return default
	r1, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	v1 := FromHTTPRequest(r1, "default-123456")
	require.Equal(t, "default-123456", v1)

	// header set should return actual value
	r2, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	r2.Header.Add(headerRequestID, "reqid-1234565789")

	v2 := FromHTTPRequest(r2, "default-123456")
	require.Equal(t, "reqid-1234565789", v2)
}
