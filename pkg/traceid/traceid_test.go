// +build unit

package traceid

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewContext(t *testing.T) {
	t.Parallel()

	// store value in context
	ctx := NewContext(context.Background(), "test-1-218549")

	// load the value from context and ignore default
	el1 := FromContext(ctx, "default-104173")
	require.Equal(t, el1, "test-1-218549")

	// do not override the value in context
	ctx1 := NewContext(ctx, "test-2-563011")
	require.Equal(t, ctx, ctx1)
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	// context without set id, should return the default value
	id1 := FromContext(context.Background(), "default-1-206951")
	require.NotEmpty(t, id1)
	require.Equal(t, "default-1-206951", id1)

	// context with set id, should return the existing value
	ctx := NewContext(context.Background(), "default-2-616841")
	id2 := FromContext(ctx, "default-3-67890")
	require.NotEmpty(t, id2)
	require.Equal(t, "default-2-616841", id2)
}

func TestSetHTTPRequestHeaderFromContext(t *testing.T) {
	t.Parallel()

	// header not set
	r1, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	id1 := SetHTTPRequestHeaderFromContext(context.Background(), r1, DefaultHeader, DefaultValue)
	require.Equal(t, id1, DefaultValue)
	require.Equal(t, r1.Header.Get(DefaultHeader), DefaultValue)

	// header set
	r2, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	ctx := NewContext(context.Background(), "test-904117")
	r2 = r2.WithContext(ctx)

	id2 := SetHTTPRequestHeaderFromContext(ctx, r2, DefaultHeader, DefaultValue)
	require.NotEqual(t, id2, DefaultValue)
	require.Equal(t, "test-904117", r2.Header.Get(DefaultHeader))
}

func TestFromHTTPRequestHeader(t *testing.T) {
	t.Parallel()

	// header not set should return default
	r1, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	v1 := FromHTTPRequestHeader(r1, DefaultHeader, "default-1-103993")
	require.Equal(t, "default-1-103993", v1)

	// header set should return actual value
	r2, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	r2.Header.Add(DefaultHeader, "test-1-413579")

	v2 := FromHTTPRequestHeader(r2, DefaultHeader, "default-2-968041")
	require.Equal(t, "test-1-413579", v2)
}
