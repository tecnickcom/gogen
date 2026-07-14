package redact

import (
	"slices"
)

// maxCardDigits is the longest supported PAN length (19-digit Visa, UnionPay,
// Discover, JCB, and Maestro ranges exist alongside the classic 13-16 digit
// ones).
const maxCardDigits = 19

// minShortMaestroDigits is the shortest Maestro PAN length. Maestro numbers
// shorter than 16 digits are only detected when the Luhn gate is enabled (see
// isCreditCard).
const minShortMaestroDigits = 12

// matchesCardPattern reports whether a run of ASCII digits (guaranteed by the
// callers) has a known card-network prefix and a valid length for that
// network. Maestro is only matched on its well-known 4-digit IINs at 16-19
// digits (see matchesMaestroIIN); the broader Maestro ranges (50, 56-69) and
// its short 12-15 digit forms are excluded here because they collide too
// easily with ordinary numeric identifiers.
func matchesCardPattern(digits []byte) bool {
	n := len(digits)

	switch {
	case n < 13 || n > maxCardDigits:
		return false
	case digits[0] == '4':
		// Visa: 13, 16 or 19 digits.
		return n == 13 || n == 16 || n == 19
	case n >= 16:
		return matchesCard16To19(digits, n)
	case n == 15:
		return matchesCard15(digits)
	case n == 14:
		return matchesCard14(digits)
	default:
		return false
	}
}

// matchesCard15 checks the 15-digit networks: Amex 34/37, JCB 2131, Diners 1800.
func matchesCard15(digits []byte) bool {
	switch digits[0] {
	case '3':
		return digits[1] == '4' || digits[1] == '7'
	case '2':
		return digits[1] == '1' && digits[2] == '3' && digits[3] == '1'
	case '1':
		return digits[1] == '8' && digits[2] == '0' && digits[3] == '0'
	default:
		return false
	}
}

// matchesCard14 checks the 14-digit network: Diners Club 300-305, 36, 38.
func matchesCard14(digits []byte) bool {
	return digits[0] == '3' && ((digits[1] == '0' && digits[2] >= '0' && digits[2] <= '5') ||
		digits[1] == '6' || digits[1] == '8')
}

// matchesCard16To19 checks the 16-19 digit networks (16 <= n <= 19).
func matchesCard16To19(digits []byte, n int) bool {
	// Mastercard 2-series (2221-2720, approximated) and 51-57: 16 digits only.
	if n == 16 && (digits[0] == '2' || digits[0] == '5') && digits[1] >= '1' && digits[1] <= '7' {
		return true
	}

	// JCB 35: 16-19 digits.
	if digits[0] == '3' {
		return digits[1] == '5'
	}

	// Maestro well-known IINs: 16-19 digits (shorter Maestro forms are
	// handled separately behind the Luhn gate in isCreditCard).
	if matchesMaestroIIN(digits) {
		return true
	}

	return digits[0] == '6' && matchesCard6Series(digits)
}

// matchesCard6Series checks the '6'-prefix 16-19 digit ranges: UnionPay 62
// (also covers Discover 622126-622925), Discover 65, 6011 and 644-649.
func matchesCard6Series(digits []byte) bool {
	return digits[1] == '2' || digits[1] == '5' ||
		(digits[1] == '0' && digits[2] == '1' && digits[3] == '1') ||
		(digits[1] == '4' && digits[2] >= '4' && digits[2] <= '9')
}

// matchesMaestroIIN reports whether digits (len >= 4, guaranteed by the
// callers) begin with a well-known Maestro issuer prefix: 5018, 5020, 5038,
// 5893, 6304, 6759, 6761, 6762 or 6763. The broader Maestro ranges (50, 56-69)
// are deliberately not matched: they collide too easily with ordinary numeric
// identifiers.
func matchesMaestroIIN(digits []byte) bool {
	iin := int(digits[0]-'0')*1000 + int(digits[1]-'0')*100 + int(digits[2]-'0')*10 + int(digits[3]-'0')

	switch iin {
	case 5018, 5020, 5038, 5893, 6304, 6759, 6761, 6762, 6763:
		return true
	default:
		return false
	}
}

// isCreditCard reports whether a run of ASCII digits should be redacted as a
// credit-card number. It always requires a known card prefix; when the
// instance's Luhn gate ([WithLuhnCheck]) is enabled it additionally requires a
// valid Luhn checksum.
//
// Short Maestro numbers (12-15 digits, well-known IINs only) are a special
// case: they are detected ONLY when the Luhn gate is enabled, because a short
// prefix-and-length match alone would over-redact far too many ordinary
// identifiers.
func (re *Redactor) isCreditCard(digits []byte) bool {
	if matchesCardPattern(digits) {
		return !re.luhn || passesLuhn(digits)
	}

	if re.luhn && len(digits) >= minShortMaestroDigits && len(digits) < 16 && matchesMaestroIIN(digits) {
		return passesLuhn(digits)
	}

	return false
}

// passesLuhn reports whether a run of ASCII digits satisfies the Luhn checksum.
func passesLuhn(digits []byte) bool {
	sum := 0
	double := false

	for _, c := range slices.Backward(digits) {
		d := int(c - '0')

		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		double = !double
	}

	return sum%10 == 0
}

// scanGroupedCardSpan detects a credit-card number written as several digit
// groups joined by single space or '-' separators (e.g. "4012 8888 8888 1881"
// or "4012-8888-8888-1881"), given the first group src[start:firstGroupEnd].
// It returns the index just past the span to redact and true when a prefix of
// the joined groups, ending on a group boundary that is not glued to a word
// character, forms a card number. Candidates are evaluated at every group
// boundary (windowed, longest match wins) rather than only after joining
// greedily, so a card followed by further digit groups — e.g. a PAN with a
// trailing "12 26" expiry — is still caught.
func (re *Redactor) scanGroupedCardSpan(src []byte, start, firstGroupEnd int) (int, bool) {
	var digits [maxCardDigits]byte

	n := copyDigits(digits[:], src[start:firstGroupEnd])
	end := firstGroupEnd
	best := 0

	for n >= 0 && hasSeparatedDigit(src, end) {
		groupEnd := scanDigits(src, end+1)

		n = appendDigits(digits[:], n, src[end+1:groupEnd])
		if n < 0 {
			break
		}

		end = groupEnd
		if (end >= len(src) || !isWordChar(src[end])) && re.isGroupedCardCandidate(digits[:n]) {
			best = end
		}
	}

	if best == 0 {
		return 0, false
	}

	return best, true
}

// isGroupedCardCandidate reports whether joined digit groups form a card
// number eligible for grouped-format detection. The 14-digit Diners range and
// the legacy 15-digit ranges (JCB 2131, Diners 1800) are excluded: those cards
// are never printed in spaced groups, and their prefixes collide with common
// phone formats ("1 800 555 0199 1234", "36 1234 5678 9012"). Amex (15 digits,
// prefix 3) stays eligible — it is an active network physically printed in
// 4-6-5 groups. Contiguous runs are unaffected and keep matching all networks.
func (re *Redactor) isGroupedCardCandidate(digits []byte) bool {
	n := len(digits)
	if n == 14 || (n == 15 && digits[0] != '3') {
		return false
	}

	return re.isCreditCard(digits)
}

// hasSeparatedDigit reports whether src[i] is a single space or '-' separator
// immediately followed by another digit group.
func hasSeparatedDigit(src []byte, i int) bool {
	return i+1 < len(src) && (src[i] == ' ' || src[i] == '-') && isDigitByte(src[i+1])
}

// copyDigits copies src into dst and returns the number of bytes copied, or -1
// when src does not fit, signaling more digits than any card can hold.
func copyDigits(dst, src []byte) int {
	if len(src) > len(dst) {
		return -1
	}

	copy(dst, src)

	return len(src)
}

// appendDigits appends group after the first n (>= 0) bytes of dst, returning
// the new length, or -1 when group does not fit. Callers guard the loop on a
// non-negative length so a -1 result terminates the scan.
func appendDigits(dst []byte, n int, group []byte) int {
	m := copyDigits(dst[n:], group)
	if m < 0 {
		return -1
	}

	return n + m
}
