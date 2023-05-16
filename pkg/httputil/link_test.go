package httputil

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		domain   string
		basePath string
		template string
		segments []any
		want     string
	}{
		{
			name:     "double slashes (domain and basePath)",
			domain:   "http://host.invalid/",
			basePath: "/path",
			template: "path_segment",
			want:     "http://host.invalid/path/path_segment",
		},
		{
			name:     "double slashes (basePath and template)",
			domain:   "http://host.invalid/",
			basePath: "path/",
			template: "/path_segment",
			want:     "http://host.invalid/path/path_segment",
		},
		{
			name:     "no slashes",
			domain:   "http://host.invalid",
			basePath: "path",
			template: "path_segment",
			want:     "http://host.invalid/path/path_segment",
		},
		{
			name:     "single segment replacing in the template",
			domain:   "http://host.invalid",
			basePath: "path",
			template: "brands/%s",
			segments: []any{"123"},
			want:     "http://host.invalid/path/brands/123",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			domain := strings.TrimRight(tt.domain, "/")
			basePath := strings.TrimLeft(tt.basePath, "/")
			u, err := url.Parse(domain + "/" + basePath)
			require.NoError(t, err)

			urlString := strings.TrimRight(u.String(), "/")
			got := Link(urlString, tt.template, tt.segments...)
			require.Equal(t, tt.want, got, "Link() = %v, want %v", got, tt.want)
		})
	}
}
