package traceid

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "empty", id: "", want: false},
		{name: "single char", id: "a", want: true},
		{name: "all allowed classes at max length", id: "0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz", want: true},
		{name: "dot allowed", id: "a.b.c", want: true},
		{name: "max length", id: strings.Repeat("a", MaxIDLen), want: true},
		{name: "one over max length", id: strings.Repeat("a", MaxIDLen+1), want: false},
		{name: "space", id: "ab cd", want: false},
		{name: "tab", id: "ab\tcd", want: false},
		{name: "trailing newline", id: "abc\n", want: false},
		{name: "trailing carriage return newline", id: "abc\r\n", want: false},
		{name: "colon", id: "a:b", want: false},
		{name: "invalid punctuation", id: "0123#~'", want: false},
		{name: "non-ascii", id: "abcé", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, Valid(tt.id))
		})
	}
}

func TestNewContext(t *testing.T) {
	t.Parallel()

	// store value in context
	ctx := NewContext(t.Context(), "test-1-218549")

	// load the value from context and ignore default
	el1 := FromContext(ctx, "default-104173")
	require.Equal(t, "test-1-218549", el1)

	// do not override the value in context
	ctx1 := NewContext(ctx, "test-2-563011")
	require.Equal(t, ctx, ctx1)
}

func TestNewContextEmptyIDSkipped(t *testing.T) {
	t.Parallel()

	// an empty id must not be stored, so it does not shadow a later real id
	base := t.Context()

	ctxEmpty := NewContext(base, "")
	require.Equal(t, base, ctxEmpty)

	// the default value is still returned because nothing was stored
	require.Equal(t, "default-720931", FromContext(ctxEmpty, "default-720931"))

	// a subsequent real id is stored as expected
	ctxReal := NewContext(ctxEmpty, "real-720932")
	require.Equal(t, "real-720932", FromContext(ctxReal, "default-720933"))
}

func TestForceContext(t *testing.T) {
	t.Parallel()

	// an empty id must not be stored and the context is returned unchanged
	base := t.Context()
	require.Equal(t, base, ForceContext(base, ""))

	// overwrites an already-stored id (unlike NewContext, which preserves it)
	ctxExisting := NewContext(base, "old-330011")
	ctxForced := ForceContext(ctxExisting, "new-330012")
	require.Equal(t, "new-330012", FromContext(ctxForced, "default-330013"))

	// storing the same id again returns the same context (no re-wrap)
	ctxSame := ForceContext(ctxForced, "new-330012")
	require.Equal(t, ctxForced, ctxSame)

	// stores the id when the context has none yet
	ctxFresh := ForceContext(base, "fresh-330014")
	require.Equal(t, "fresh-330014", FromContext(ctxFresh, "default-330015"))
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	// context without set id, should return the default value
	id1 := FromContext(t.Context(), "default-1-206951")
	require.NotEmpty(t, id1)
	require.Equal(t, "default-1-206951", id1)

	// context with set id, should return the existing value
	ctx := NewContext(t.Context(), "default-2-616841")
	id2 := FromContext(ctx, "default-3-67890")
	require.NotEmpty(t, id2)
	require.Equal(t, "default-2-616841", id2)
}

func TestSetHTTPRequestHeaderFromContext(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// header not set: with the empty DefaultValue no header must be transmitted
	r1, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	require.NoError(t, err)

	id1 := SetHTTPRequestHeaderFromContext(t.Context(), r1, DefaultHeader, DefaultValue)
	require.Equal(t, DefaultValue, id1)
	require.Empty(t, r1.Header.Values(DefaultHeader), "an empty header must not be set")

	// header set
	r2, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	require.NoError(t, err)

	ctx = NewContext(ctx, "test-904117")
	r2 = r2.WithContext(ctx)

	id2 := SetHTTPRequestHeaderFromContext(ctx, r2, DefaultHeader, DefaultValue)
	require.NotEqual(t, DefaultValue, id2)
	require.Equal(t, "test-904117", r2.Header.Get(DefaultHeader))
}

func TestSetHTTPRequestHeaderFromContextInvalidIDFallback(t *testing.T) {
	t.Parallel()

	// an invalid id stored in the context must not be written to the header;
	// it must fall back to the (valid) default value to prevent header injection.
	const injection = "valid\r\nX-Injected: evil"

	ctx := NewContext(t.Context(), injection)

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	require.NoError(t, err)

	got := SetHTTPRequestHeaderFromContext(ctx, r, DefaultHeader, "fallback-771201")
	require.Equal(t, "fallback-771201", got)
	require.Equal(t, "fallback-771201", r.Header.Get(DefaultHeader))

	// when both the context id and the default are invalid (empty), nothing is
	// written: no empty header is transmitted and an empty id is returned.
	ctx2 := NewContext(t.Context(), injection)

	r2, err := http.NewRequestWithContext(ctx2, http.MethodGet, "/", nil)
	require.NoError(t, err)

	got2 := SetHTTPRequestHeaderFromContext(ctx2, r2, DefaultHeader, DefaultValue)
	require.Equal(t, DefaultValue, got2)
	require.Empty(t, r2.Header.Values(DefaultHeader), "an empty header must not be set")

	// an invalid non-empty default is validated with the same regex: it must not
	// be written to the header either.
	ctx3 := NewContext(t.Context(), injection)

	r3, err := http.NewRequestWithContext(ctx3, http.MethodGet, "/", nil)
	require.NoError(t, err)

	got3 := SetHTTPRequestHeaderFromContext(ctx3, r3, DefaultHeader, "invalid default\r\n")
	require.Empty(t, got3)
	require.Empty(t, r3.Header.Values(DefaultHeader), "an invalid default must not be set")
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

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

func TestSetHTTPRequestHeaderFromContextNilRequest(t *testing.T) {
	t.Parallel()

	// a nil request must not panic; nothing is set and an empty id is returned.
	ctx := NewContext(t.Context(), "id-550101")
	require.Empty(t, SetHTTPRequestHeaderFromContext(ctx, nil, DefaultHeader, DefaultValue))
}

func TestSetHTTPRequestHeaderFromContextNilHeaderMap(t *testing.T) {
	t.Parallel()

	// a request whose Header map is nil must not panic; the map is created and the id is set.
	ctx := NewContext(t.Context(), "id-550102")

	r := &http.Request{}

	got := SetHTTPRequestHeaderFromContext(ctx, r, DefaultHeader, DefaultValue)
	require.Equal(t, "id-550102", got)
	require.Equal(t, "id-550102", r.Header.Get(DefaultHeader))
}

func TestFromHTTPRequestHeaderNilRequest(t *testing.T) {
	t.Parallel()

	// a nil request must not panic; the default value is returned.
	require.Equal(t, "default-550103", FromHTTPRequestHeader(nil, DefaultHeader, "default-550103"))
}

func BenchmarkValid(b *testing.B) {
	const id = "0191b2f1-8f3a-7c2d-9e4b-1a2b3c4d5e6f"

	for b.Loop() {
		_ = Valid(id)
	}
}

func BenchmarkFromHTTPRequestHeader(b *testing.B) {
	r, err := http.NewRequestWithContext(b.Context(), http.MethodGet, "/", nil)
	require.NoError(b, err)

	r.Header.Set(DefaultHeader, "0191b2f1-8f3a-7c2d-9e4b-1a2b3c4d5e6f")

	for b.Loop() {
		_ = FromHTTPRequestHeader(r, DefaultHeader, "")
	}
}
