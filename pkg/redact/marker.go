package redact

// Redaction patterns and replacements.
const (
	// RedactionMarker is the placeholder used to replace sensitive values.
	RedactionMarker = `***`
)

// redactedBytes is the marker of the default configuration, reused by every
// instance that does not override it with [WithMarker].
var redactedBytes = []byte(RedactionMarker) //nolint:gochecknoglobals

// markerEndsAt reports whether this instance's marker occupies the bytes
// ending at src[j] (inclusive).
func (re *Redactor) markerEndsAt(src []byte, j int) bool {
	start := j + 1 - len(re.marker)
	if start < 0 || src[j] != re.marker[len(re.marker)-1] {
		return false
	}

	for k, c := range re.marker {
		if src[start+k] != c {
			return false
		}
	}

	return true
}

// markerAt reports whether this instance's marker starts at src[i].
func (re *Redactor) markerAt(src []byte, i int) bool {
	if i+len(re.marker) > len(src) {
		return false
	}

	for k, c := range re.marker {
		if src[i+k] != c {
			return false
		}
	}

	return true
}
