/*
Package strsplit contains utility functions to split Unicode strings.
*/
package strsplit

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Chunk slices the input text block string s into substrings of maximum byte size at the closest separator.
// Account for Unicode strings to avoid splitting multi-byte characters.
// Newlines characters are applied first.
//
//nolint:gocognit
func Chunk(s string, size, n int) []string {
	if (size < 1) || (n == 0) {
		return nil
	}

	ret := make([]string, 0)

	// look for newlines first
	for line := range strings.SplitSeq(s, "\n") {
		if (n > 0) && (len(ret) >= n) {
			break
		}

		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		if len(line) <= size {
			ret = append(ret, line)
			continue
		}

		ns := n
		if ns > 0 {
			ns = n - len(ret)
		}

		ret = append(ret, ChunkLine(line, size, ns)...)
	}

	return ret
}

// ChunkLine slices a string line into substrings of maximum byte size at the closest unicode space separator.
// Account for Unicode strings to avoid splitting multy-byte characters.
//
//nolint:gocognit,gocyclo,cyclop
func ChunkLine(s string, size, n int) []string {
	if (size < 1) || (n == 0) {
		return nil
	}

	ret := make([]string, 0)

	for len(s) > size {
		if (n > 0) && (len(ret) >= n) {
			break
		}

		end := size

		// ensure we aren't in the middle of a multi-byte Unicode character
		for (end > 0) && !utf8.RuneStart(s[end]) {
			end--
		}

		// try to split by unicode spaces
		sepIdx := strings.LastIndexFunc(s[:end], unicode.IsSpace)
		if sepIdx == -1 {
			// try to split by punctuaction characters
			sepIdx = strings.LastIndexFunc(s[:end], unicode.IsPunct)
			if sepIdx >= 0 {
				// keep the puctuation character with the line
				_, offset := utf8.DecodeRuneInString(s[sepIdx:])
				sepIdx += offset
			}
		}

		if sepIdx >= 0 {
			end = sepIdx
		}

		chunk := strings.TrimSpace(s[:end])
		if len(chunk) > 0 {
			ret = append(ret, chunk)
		}

		s = strings.TrimSpace(s[end:])
	}

	if (len(s) > 0) && ((n < 0) || (len(ret) < n)) {
		ret = append(ret, strings.TrimSpace(s))
	}

	return ret
}
