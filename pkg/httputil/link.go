package httputil

import (
	"fmt"
	"strings"
)

// Link composes URL by substituting template segments using fmt.Sprintf and appending to service URL.
//
// A trailing slash on url and a leading slash on template are trimmed so exactly
// one separator joins them.
//
// When segments are provided, template is used as a fmt.Sprintf format string.
// For this reason template MUST be a trusted, compile-time constant format
// string controlled by the caller. Never pass an externally-derived or
// user-supplied value as template, as it would be interpreted as a format
// string (a format-string injection footgun).
//
// Segments are not URL-escaped: callers passing untrusted string segments must
// escape them (e.g. with url.PathEscape) to avoid injecting path or query
// delimiters. Note also that a literal "%" in template is emitted verbatim when
// no segments are given but interpreted as a fmt verb once any segment is passed.
func Link(url, template string, segments ...any) string {
	url = strings.TrimRight(url, "/")
	template = strings.TrimLeft(template, "/")

	if len(segments) > 0 {
		template = fmt.Sprintf(template, segments...)
	}

	return url + "/" + template
}
