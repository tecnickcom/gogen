package redact

import (
	"bytes"
)

// appendRedactedXMLValueAt handles a '<' at src[i]: when it opens an XML/HTML
// element whose name is sensitive (via the same tokenized keyword check as
// keys) and the element has flat text content followed by its matching
// closing tag, the content is replaced with the marker:
//
//	<password>SECRET</password>  ->  <password>***</password>
//
// The matching-closing-tag requirement keeps prose placeholders like
// "user <token> expired" untouched. Only flat content is redacted; nested
// elements are handled by their own names when scanned. Sensitive XML
// *attributes* (password="...") are already covered by the URL-encoded rule.
func (re *Redactor) appendRedactedXMLValueAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	nameStart := i + 1
	if nameStart >= len(src) || !isASCIIAlpha(src[nameStart]) {
		return 0, dst, false
	}

	nameEnd := nameStart
	for nameEnd < len(src) && isXMLNameByte(src[nameEnd]) {
		nameEnd++
	}

	gt := xmlOpenTagClose(src, nameEnd)
	if gt == 0 {
		return 0, dst, false
	}

	name := src[nameStart:nameEnd]
	if !re.isSensitiveKey(name) {
		return 0, dst, false
	}

	contentStart := gt + 1

	contentEnd := xmlFlatContentEnd(src, contentStart, name)
	if contentEnd == 0 {
		return 0, dst, false
	}

	dst = append(dst, src[i:contentStart]...)
	dst = append(dst, re.marker...)

	return contentEnd, dst, true
}

// xmlCDATAPrefix opens a CDATA section, the idiomatic way to carry opaque
// (often secret) payloads in SOAP/XML documents.
const xmlCDATAPrefix = "<![CDATA["

// xmlFlatContentEnd returns the end of the flat (element-free) text content
// starting at contentStart, requiring it to be non-empty and immediately
// followed by the matching closing tag; it returns 0 otherwise. A leading
// CDATA section is treated as content, so `<password><![CDATA[x]]></password>`
// redacts like plain text.
func xmlFlatContentEnd(src []byte, contentStart int, name []byte) int {
	contentEnd := contentStart

	if hasPrefixAt(src, contentEnd, xmlCDATAPrefix) {
		idx := bytes.Index(src[contentEnd+len(xmlCDATAPrefix):], []byte("]]>"))
		if idx < 0 {
			return 0
		}

		contentEnd += len(xmlCDATAPrefix) + idx + len("]]>")
	}

	for contentEnd < len(src) && src[contentEnd] != '<' {
		contentEnd++
	}

	if contentEnd == contentStart || !isMatchingXMLClosingTag(src, contentEnd, name) {
		return 0
	}

	return contentEnd
}

// xmlOpenTagClose locates the '>' closing an open tag whose name ends at
// nameEnd (attributes allowed), returning 0 for self-closing tags and tags
// broken across lines or by another '<'.
func xmlOpenTagClose(src []byte, nameEnd int) int {
	gt := nameEnd
	for gt < len(src) && src[gt] != '>' && src[gt] != '\n' && src[gt] != '<' {
		gt++
	}

	if gt >= len(src) || src[gt] != '>' || src[gt-1] == '/' {
		return 0
	}

	return gt
}

// isMatchingXMLClosingTag reports whether src[i:] begins with the closing tag
// `</name>` for the given element name.
func isMatchingXMLClosingTag(src []byte, i int, name []byte) bool {
	if !hasPrefixAt(src, i, "</") {
		return false
	}

	nameStart := i + len("</")
	gt := nameStart + len(name)

	if gt >= len(src) || src[gt] != '>' {
		return false
	}

	for j, c := range name {
		if src[nameStart+j] != c {
			return false
		}
	}

	return true
}

func isXMLNameByte(c byte) bool {
	return isHeaderNameByte(c) || c == ':' || c == '.'
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
