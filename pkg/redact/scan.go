package redact

// Trigger classes for the bulk-copy scan: most bytes are trigNone and cost a
// single table load; the rare candidate classes run their cheap prefilter
// before breaking out to the rule dispatch.
const (
	trigNone byte = iota
	trigStop
	trigColon
	trigJWT
	trigDash
	trigVendor
)

// bulkTrigger classifies every byte for bulkTextEnd: hard stops (rule bytes
// and digits), and the prefiltered candidate starts of the userinfo (':'),
// JWT ('e'), inline-PEM ('-'), and vendor-token rules. An escaped JSON key
// ('\"') is caught at its '"', already a hard stop, so '\' needs no class here.
var bulkTrigger = [256]byte{ //nolint:gochecknoglobals
	'"': trigStop, '=': trigStop, '\n': trigStop, '<': trigStop,
	'0': trigStop, '1': trigStop, '2': trigStop, '3': trigStop, '4': trigStop,
	'5': trigStop, '6': trigStop, '7': trigStop, '8': trigStop, '9': trigStop,
	':': trigColon,
	'e': trigJWT,
	'-': trigDash,
	'g': trigVendor, 'x': trigVendor, 's': trigVendor, 'r': trigVendor,
	'w': trigVendor, 'd': trigVendor, 'A': trigVendor, 'h': trigVendor,
	'S': trigVendor,
}

// bulkTextEnd returns the end of the plain-text run starting at src[i]: the
// scan stops at bytes that begin a redaction rule. Ordinary bytes cost one
// table load; candidate bytes run a short prefilter so the bulk copy stays
// tight for ordinary text.
//
//nolint:gocognit,gocyclo,cyclop // Deliberately flat hot loop: prefilters must stay inline.
func bulkTextEnd(src []byte, i int) int {
	j := i + 1
	for j < len(src) {
		switch bulkTrigger[src[j]] {
		case trigStop:
			return j
		case trigColon:
			if j+2 < len(src) && src[j+1] == '/' && src[j+2] == '/' {
				return j
			}
		case trigJWT:
			if j+2 < len(src) && src[j+1] == 'y' && src[j+2] == 'J' && !isWordChar(src[j-1]) {
				return j
			}
		case trigDash:
			if hasPrefixAt(src, j, "-----B") {
				return j
			}
		case trigVendor:
			if !isWordChar(src[j-1]) && isVendorTokenStart(src, j) {
				return j
			}
		}

		j++
	}

	return j
}

// Byte-class predicates and scans shared by the engine and the rule files
// (credit cards, URL-encoded pairs, JWTs, vendor tokens). They are deliberately
// ASCII-only and branch-free enough to stay inlinable on the hot path.

func isWordChar(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isDigitByte(c byte) bool {
	return c >= '0' && c <= '9'
}

// scanDigits returns the index just past the run of ASCII digits starting at i.
func scanDigits(src []byte, i int) int {
	for i < len(src) && isDigitByte(src[i]) {
		i++
	}

	return i
}
