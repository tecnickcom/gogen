package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPDataXMLElementValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// Sensitive element content is replaced; tags stay visible.
		{"<password>SECRET</password>", "<password>***</password>"},
		{"<Token>SECRET</Token>", "<Token>***</Token>"},
		{"<ns:Token>SECRET</ns:Token>", "<ns:Token>***</ns:Token>"},
		{"<api.key>SECRET</api.key>", "<api.key>***</api.key>"},
		// Open tags with attributes still redact their content; sensitive
		// attributes themselves are covered by the URL-encoded rule.
		{`<password attr="x">SECRET</password>`, `<password attr="x">***</password>`},
		{`<user password="SECRET" name="bob"/>`, `<user password=*** name="bob"/>`},
		// Multi-line content is redacted whole.
		{"<password>\nline1\nline2\n</password>", "<password>***</password>"},
		// Non-sensitive elements are untouched.
		{"<note>ok</note>", "<note>ok</note>"},
		// Prose placeholders without a matching closing tag stay visible.
		{"user <token> expired", "user <token> expired"},
		// Mismatched closing tag: not a flat element, left alone.
		{"<password>x</secret>", "<password>x</secret>"},
		// Nested elements are handled per-element (flat-content rule only).
		{"<password><v>x</v></password>", "<password><v>x</v></password>"},
		// Self-closing, closing, declaration, and comment tags never match.
		{"<password/>", "<password/>"},
		{"</password>", "</password>"},
		{"<?xml version=\"1.0\"?>", "<?xml version=\"1.0\"?>"},
		{"<!-- password hint -->", "<!-- password hint -->"},
		// Empty content is left alone (nothing to hide).
		{"<password></password>", "<password></password>"},
		// Tag broken by a newline or another '<' is not an element.
		{"<password\n>SECRET</password>", "<password\n>SECRET</password>"},
		{"<password <x>y</x>", "<password <x>y</x>"},
		// '<' at end of input.
		{"a <", "a <"},
		// Already-redacted output is stable (idempotency).
		{"<password>***</password>", "<password>***</password>"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestIsMatchingXMLClosingTagNameMismatch(t *testing.T) {
	t.Parallel()

	// Same-length closing tag with a different name must not match.
	input := "<password>x</passw0rd>"
	require.Equal(t, input, HTTPData(input))
}

// TestHTTPDataXMLCDATA verifies CDATA sections under sensitive elements are
// redacted as content.
func TestHTTPDataXMLCDATA(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"<password><![CDATA[SECRET]]></password>", "<password>***</password>"},
		{"<password><![CDATA[SECRET]]>tail</password>", "<password>***</password>"},
		// Unterminated CDATA is not flat content: left to the normal scan.
		{"<password><![CDATA[SECRET</password>", "<password><![CDATA[SECRET</password>"},
		// CDATA under a non-sensitive element stays visible.
		{"<note><![CDATA[ok]]></note>", "<note><![CDATA[ok]]></note>"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}
