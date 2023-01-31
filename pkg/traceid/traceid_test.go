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

	ctx := context.Background()

	// header not set
	r1, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	require.NoError(t, err)

	id1 := SetHTTPRequestHeaderFromContext(context.Background(), r1, DefaultHeader, DefaultValue)
	require.Equal(t, id1, DefaultValue)
	require.Equal(t, r1.Header.Get(DefaultHeader), DefaultValue)

	// header set
	r2, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	require.NoError(t, err)

	ctx = NewContext(ctx, "test-904117")
	r2 = r2.WithContext(ctx)

	id2 := SetHTTPRequestHeaderFromContext(ctx, r2, DefaultHeader, DefaultValue)
	require.NotEqual(t, id2, DefaultValue)
	require.Equal(t, "test-904117", r2.Header.Get(DefaultHeader))
}

func TestFromHTTPRequestHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		def  string
		want string
	}{
		{
			name: "set value",
			id:   "0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz",
			def:  "default-1-968041",
			want: "0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz",
		},
		{
			name: "default if empty",
			id:   "",
			def:  "default-2-103992",
			want: "default-2-103992",
		},
		{
			name: "default if invalid characters",
			id:   "0123#~'",
			def:  "default-3-103993",
			want: "default-3-103993",
		},
		{
			name: "default if too long",
			id:   "0123456789012345678901234567890123456789012345678901234567890123456789",
			def:  "default-4-103994",
			want: "default-4-103994",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			require.NoError(t, err)

			if tt.id != "" {
				r.Header.Add(DefaultHeader, tt.id)
			}

			v := FromHTTPRequestHeader(r, DefaultHeader, tt.def)
			require.Equal(t, tt.want, v)
		})
	}
}
