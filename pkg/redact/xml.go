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

// maxXMLContentScan bounds the window searched for an embedded comment/CDATA
// terminator. Without it, a line of sensitive tags each opening an unterminated
// `<!--`/`<![CDATA[` rescans the whole tail per tag (O(n^2) redaction DoS): the
// failed rule does not consume, so the outer scan re-enters the search at the
// next tag. Bounding the terminator search keeps every failed element's scan to
// a fixed window, so total cost is linear. The plain-text content scan is NOT
// bounded: it stops at the first '<' (the closing tag), so a large flat secret
// with no embedded comment/CDATA still redacts; only a comment/CDATA-wrapped
// secret longer than the window is left visible (an exotic shape, and the
// safe-vs-DoS tradeoff analogous to PEM's bounded body scan).
const maxXMLContentScan = 1 << 13

// xmlFlatContentEnd returns the end of the flat (element-free) text content
// starting at contentStart, requiring it to be non-empty and immediately
// followed by the matching closing tag; it returns 0 otherwise. Embedded
// comments and CDATA sections anywhere in the content are treated as opaque
// content, so `<password>a<![CDATA[x]]></password>` and
// `<password><!--c-->x</password>` redact like plain text. A nested element
// (any other `<...>`) still stops the scan, so prose placeholders such as
// "user <token> expired" stay untouched.
func xmlFlatContentEnd(src []byte, contentStart int, name []byte) int {
	contentEnd := contentStart

	for contentEnd < len(src) {
		if src[contentEnd] != '<' {
			contentEnd++

			continue
		}

		if skipTo, ok := skipXMLCommentOrCDATA(src, contentEnd); ok {
			contentEnd = skipTo

			continue
		}

		break // a real tag: the candidate closing tag
	}

	if contentEnd == contentStart || contentEnd >= len(src) || !isMatchingXMLClosingTag(src, contentEnd, name) {
		return 0
	}

	return contentEnd
}

// skipXMLCommentOrCDATA, when src[i] opens a comment (<!--) or CDATA section
// (<![CDATA[), returns the index just past its terminator and true. The
// terminator search is bounded by maxXMLContentScan; a section whose terminator
// is not found within that window is treated as unterminated (returns len(src),
// so the caller fails the element) rather than rescanned per tag, keeping the
// outer scan from going quadratic. It returns (i, false) for any other byte
// sequence.
func skipXMLCommentOrCDATA(src []byte, i int) (int, bool) {
	if hasPrefixAt(src, i, "<!--") {
		return skipToXMLTerminator(src, i+len("<!--"), "-->")
	}

	if hasPrefixAt(src, i, xmlCDATAPrefix) {
		return skipToXMLTerminator(src, i+len(xmlCDATAPrefix), "]]>")
	}

	return i, false
}

// skipToXMLTerminator searches for term within maxXMLContentScan bytes from
// start, returning the index just past it and true when found, or (len(src),
// true) when the section is unterminated within the window.
func skipToXMLTerminator(src []byte, start int, term string) (int, bool) {
	limit := min(len(src), start+maxXMLContentScan)

	if idx := bytes.Index(src[start:limit], []byte(term)); idx >= 0 {
		return start + idx + len(term), true
	}

	return len(src), true
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
// `</name>` for the given element name. The name is compared case-insensitively
// (HTML is case-insensitive and hand-written markup is inconsistent), and
// whitespace between the name and '>' is tolerated (`</token >`).
func isMatchingXMLClosingTag(src []byte, i int, name []byte) bool {
	if !hasPrefixAt(src, i, "</") {
		return false
	}

	nameStart := i + len("</")
	if nameStart+len(name) > len(src) {
		return false
	}

	for j := range name {
		if lowerASCIIByte(src[nameStart+j]) != lowerASCIIByte(name[j]) {
			return false
		}
	}

	gt := nameStart + len(name)
	for gt < len(src) && (src[gt] == ' ' || src[gt] == '\t') {
		gt++
	}

	return gt < len(src) && src[gt] == '>'
}

func isXMLNameByte(c byte) bool {
	return isHeaderNameByte(c) || c == ':' || c == '.'
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
