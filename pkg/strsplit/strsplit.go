/*
Package strsplit solves the common text-wrapping problem of splitting strings
into bounded-size chunks without breaking Unicode characters and while trying
to keep human-readable boundaries (spaces, punctuation, and line breaks).

# Problem

Naive string slicing by byte index is unsafe for UTF-8 text and can split a
multi-byte rune in the middle, producing invalid output. Even when output stays
valid, hard cuts in the middle of words make messages difficult to read (for
example in chat payload limits, SMS segmentation, logs, or fixed-size transport
frames).

strsplit provides chunking helpers that are UTF-8 aware and separator-aware,
so chunks stay valid and readable.

# How It Works

The package exposes two functions:

  - [Chunk]: splits a full text block, prioritizing newline boundaries first,
    trimming whitespace per line, then delegating long lines to [ChunkLine].
  - [ChunkLine]: splits a single line by maximum byte size, ensuring the split
    point is at a rune boundary and preferring the closest separator before the
    limit.

Separator preference order in [ChunkLine]:

 1. Unicode whitespace.
 2. Unicode punctuation (kept with the preceding chunk).
 3. Hard UTF-8-safe cut when no separator exists.

Both functions support an optional chunk limit `n`:

  - `n > 0`: return at most `n` chunks.
  - `n < 0`: unlimited chunks.
  - `n == 0`: return nil.

# Key Features

  - UTF-8 safety: never cuts in the middle of a multi-byte rune.
  - Readability-aware splitting: prefers spaces and punctuation over arbitrary
    byte boundaries.
  - Newline-first semantics in [Chunk]: preserves natural paragraph structure.
  - Bounded output control via `n`, useful for APIs with strict item limits.
  - Deterministic trimming of leading/trailing whitespace in produced chunks.

# Usage

	chunks := strsplit.Chunk(text, 280, -1)    // split full text block
	lineParts := strsplit.ChunkLine(line, 64, 3) // at most 3 chunks

This package is ideal for any Go application that needs robust,
Unicode-aware message segmentation under byte-size constraints.
*/
package strsplit

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Chunk splits text block into substrings of max size at newline/separator boundaries, trimming whitespace and returning at most n chunks.
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

// ChunkLine splits single line into substrings of max byte size at UTF-8 boundaries, preferring whitespace/punctuation separators; returns at most n chunks.
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
