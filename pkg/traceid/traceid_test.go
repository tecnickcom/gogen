package traceid

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToContext(t *testing.T) {
	t.Parallel()

	// store value in context
	ctx := ToContext(context.Background(), "test-1-218549")

	// load the value from context and ignore default
	el1 := FromContext(ctx, "default-104173")
	require.Equal(t, el1, "test-1-218549")

	// do not override the value in context
	ctx1 := ToContext(ctx, "test-2-563011")
	require.Equal(t, ctx, ctx1)
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	// context without set id, should return the default value
	id1 := FromContext(context.Background(), "default-1-206951")
	require.NotEmpty(t, id1)
	require.Equal(t, "default-1-206951", id1)

	// context with set id, should return the existing value
	ctx := ToContext(context.Background(), "default-2-616841")
	id2 := FromContext(ctx, "default-3-67890")
	require.NotEmpty(t, id2)
	require.Equal(t, "default-2-616841", id2)
}

func TestToHTTPRequest(t *testing.T) {
	t.Parallel()

	// header not set
	r1, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	id1 := ToHTTPRequest(context.Background(), r1, DefaultKey, DefaultValue)
	require.Equal(t, id1, DefaultValue)
	require.Equal(t, r1.Header.Get(DefaultKey), DefaultValue)

	// header set
	r2, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	ctx := ToContext(context.Background(), "test-904117")
	r2 = r2.WithContext(ctx)

	id2 := ToHTTPRequest(ctx, r2, DefaultKey, DefaultValue)
	require.NotEqual(t, id2, DefaultValue)
	require.Equal(t, "test-904117", r2.Header.Get(DefaultKey))
}

func TestFromHTTPRequest(t *testing.T) {
	t.Parallel()

	// header not set should return default
	r1, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	v1 := FromHTTPRequest(r1, DefaultKey, "default-1-103993")
	require.Equal(t, "default-1-103993", v1)

	// header set should return actual value
	r2, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	r2.Header.Add(DefaultKey, "test-1-413579")

	v2 := FromHTTPRequest(r2, DefaultKey, "default-2-968041")
	require.Equal(t, "test-1-413579", v2)
}
