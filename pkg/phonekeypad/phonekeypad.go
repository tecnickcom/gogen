/*
Package phonekeypad converts alphabetic strings and phone number literals to
their numeric equivalents on a standard 12-key telephony keypad (ITU E.161 /
ITU T.9).

# Problem

Many telephony applications, vanity-number look-ups, DTMF generators, and
T9-style input systems need to translate letters to the digits printed on a
phone keypad. Doing this correctly—handling both upper- and lower-case input,
skipping punctuation and separators, and producing either a []int slice or a
plain string—requires repetitive boilerplate that this package encapsulates.

# Keypad Layout

The standard keypad mapping (ITU E.161) used by this package:

	+-----+-----+-----+
	|  1  |  2  |  3  |
	|     | ABC | DEF |
	+-----+-----+-----+
	|  4  |  5  |  6  |
	| GHI | JKL | MNO |
	+-----+-----+-----+
	|  7  |  8  |  9  |
	| PQRS| TUV | WXYZ|
	+-----+-----+-----+
	|     |  0  |     |
	|     |     |     |
	+-----+-----+-----+

# Features

  - Single-rune conversion ([KeypadDigit]): convert one character at a time;
    ideal for streaming input or custom filtering logic.
  - Slice output ([KeypadNumber]): convert a full string to []int, preserving
    each digit as an integer for arithmetic or comparison.
  - String output ([KeypadNumberString]): convert a full string directly to a
    digit string, ready for storage, display, or dialing.
  - Case-insensitive: accepts both upper- and lower-case letters with no
    pre-processing required by the caller.
  - Separator-tolerant: hyphens, spaces, parentheses, and any other
    non-alphanumeric characters are silently skipped, so formatted phone
    numbers like "1-800-FLOWERS" work without sanitisation.
  - Zero dependencies: uses only the Go standard library.

# Quick Start

Translate a vanity phone number to its dialable form:

	numStr := phonekeypad.KeypadNumberString("1-800-FLOWERS")
	// numStr == "18003569377"

Or get the digit sequence as a slice for programmatic use:

	digits := phonekeypad.KeypadNumber("1-800-FLOWERS")
	// digits == []int{1, 8, 0, 0, 3, 5, 6, 9, 3, 7, 7}

# Benefits

This package eliminates boilerplate in any Go application that deals with
telephony, DTMF, vanity numbers, or T9 text encoding. Its simple, allocation-
minimal design makes it suitable for both low-latency request handlers and
batch-processing pipelines.
*/
package phonekeypad

import (
	"fmt"
	"strings"
)

// KeypadDigit maps a single rune to its ITU E.161 phone-keypad digit.
//
// Digits '0'–'9' map to themselves. Letters 'A'–'Z' and 'a'–'z' are mapped
// case-insensitively according to the standard keypad layout:
//
//	'A','B','C'         → 2
//	'D','E','F'         → 3
//	'G','H','I'         → 4
//	'J','K','L'         → 5
//	'M','N','O'         → 6
//	'P','Q','R','S'     → 7
//	'T','U','V'         → 8
//	'W','X','Y','Z'     → 9
//
// Any other rune (punctuation, whitespace, symbols) returns (-1, false).
// This makes it safe to range over formatted phone strings like "(800) 555-1234"
// and simply skip characters where ok is false.
func KeypadDigit(r rune) (int, bool) {
	if r >= '0' && r <= '9' {
		return int(r - '0'), true
	}

	if r >= 'a' && r <= 'z' {
		r -= ('a' - 'A')
	}

	return keypadAlphaToDigit(r)
}

// keypadAlphaToDigit maps uppercase ASCII letters to ITU E.161 keypad digits.
func keypadAlphaToDigit(r rune) (int, bool) {
	switch r {
	case 'A', 'B', 'C':
		return 2, true
	case 'D', 'E', 'F':
		return 3, true
	case 'G', 'H', 'I':
		return 4, true
	case 'J', 'K', 'L':
		return 5, true
	case 'M', 'N', 'O':
		return 6, true
	case 'P', 'Q', 'R', 'S':
		return 7, true
	case 'T', 'U', 'V':
		return 8, true
	case 'W', 'X', 'Y', 'Z':
		return 9, true
	}

	return -1, false
}

// KeypadNumber converts a string to a []int of keypad digits.
//
// Each rune in s is passed to [KeypadDigit]. Characters that do not map to a
// keypad digit (separators, punctuation, etc.) are silently skipped, so
// formatted inputs such as "1-800-EXAMPLE" or "(555) 123-4567" are handled
// without pre-sanitisation.
//
// The returned slice has the same length as the number of valid keypad
// characters in s. Use [KeypadNumberString] when a plain digit string is
// preferred over a slice.
func KeypadNumber(num string) []int {
	seq := make([]int, 0, len(num))

	for _, r := range num {
		v, status := KeypadDigit(r)
		if status {
			seq = append(seq, v)
		}
	}

	return seq
}

// KeypadNumberString converts a string to a keypad digit string.
//
// It calls [KeypadNumber] and joins the resulting
// digit slice into a plain string (e.g. "18003569377"), suitable for storage,
// display, or passing directly to a dialler. Non-keypad characters are skipped
// just as in [KeypadNumber].
func KeypadNumberString(num string) string {
	seq := KeypadNumber(num)

	return strings.Trim(
		strings.Join(
			strings.Fields(
				fmt.Sprint(seq),
			),
			"",
		),
		"[]",
	)
}
