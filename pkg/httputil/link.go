package httputil

import (
	"fmt"
	"strings"
)

// Link generates a public Link using the service url.
// It replaces all segments into the template.
// The template then gets joined at the end of the service url.
func Link(url, template string, segments ...any) string {
	template = strings.TrimLeft(template, "/")

	if len(segments) > 0 {
		template = fmt.Sprintf(template, segments...)
	}

	return url + "/" + template
}
