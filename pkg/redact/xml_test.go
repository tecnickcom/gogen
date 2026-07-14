package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactXMLElementValues(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

func TestIsMatchingXMLClosingTagNameMismatch(t *testing.T) {
	t.Parallel()

	// Same-length closing tag with a different name must not match.
	input := "<password>x</passw0rd>"
	require.Equal(t, input, Default().String(input))
}

// TestRedactXMLCDATA verifies CDATA sections under sensitive elements are
// redacted as content.
func TestRedactXMLCDATA(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

// TestXMLNonFlatContent covers content shapes that previously failed open
// (leaving the whole element visible): embedded comments and mid-content CDATA,
// whitespace inside the closing tag, and a case-mismatched closing tag. Prose
// placeholders and genuinely nested elements must still stay untouched.
func TestXMLNonFlatContent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`<password>ab<![CDATA[hunter2]]></password>`, `<password>***</password>`},
		{`<password><!--comment-->hunter2</password>`, `<password>***</password>`},
		{`<token>abc123</token >`, `<token>***</token >`},
		{`<password>hunter2</Password>`, `<password>***</Password>`}, // HTML-style case mismatch
		// Guarantees that must not regress:
		{`user <token> expired`, `user <token> expired`},                       // prose placeholder
		{`<password>a<b>c</b>d</password>`, `<password>a<b>c</b>d</password>`}, // nested element (non-goal)
		{`<note>ab<![CDATA[ok]]></note>`, `<note>ab<![CDATA[ok]]></note>`},     // non-sensitive element
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)

		once := Default().String(tc.input)
		require.Equal(t, once, Default().String(once), "not idempotent: %s", tc.input)
	}
}

// TestXMLUnterminatedMarkup verifies an unterminated comment or CDATA in the
// content is treated as running to end of input, so the element is left
// unredacted (safe) rather than mis-parsed.
func TestXMLUnterminatedMarkup(t *testing.T) {
	t.Parallel()

	unchanged := []string{
		"<password>x<!--unterminated comment",
		"<password>x<![CDATA[unterminated cdata",
	}
	for _, in := range unchanged {
		require.Equal(t, in, Default().String(in), "should be unchanged: %q", in)
	}
}
