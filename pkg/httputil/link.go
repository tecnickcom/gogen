package httputil

import (
	"fmt"
	"strings"
)

// Link composes URL by substituting template segments using fmt.Sprintf and appending to service URL.
//
// When segments are provided, template is used as a fmt.Sprintf format string.
// For this reason template MUST be a trusted, compile-time constant format
// string controlled by the caller. Never pass an externally-derived or
// user-supplied value as template, as it would be interpreted as a format
// string (a format-string injection footgun).
func Link(url, template string, segments ...any) string {
	template = strings.TrimLeft(template, "/")

	if len(segments) > 0 {
		template = fmt.Sprintf(template, segments...)
	}

	return url + "/" + template
}
