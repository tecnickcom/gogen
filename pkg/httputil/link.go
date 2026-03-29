package httputil

import (
	"fmt"
	"strings"
)

// Link composes URL by substituting template segments using fmt.Sprintf and appending to service URL.
func Link(url, template string, segments ...any) string {
	template = strings.TrimLeft(template, "/")

	if len(segments) > 0 {
		template = fmt.Sprintf(template, segments...)
	}

	return url + "/" + template
}
