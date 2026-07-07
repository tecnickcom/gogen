package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPDataCreditCardWordBoundary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// standalone card number (no surrounding punctuation)
		{"4012888888881881", "***"},
		// card adjacent to spaces
		{"card 4012888888881881 end", "card *** end"},
		// card at end of line followed by newline
		{"ref: 371449635398431\n", "ref: ***\n"},
		// card in parentheses (old behavior preserved)
		{"(4222222222222)", "(***)"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestHTTPDataCreditCardsKeepsDigitsAdjacentToWordChar(t *testing.T) {
	t.Parallel()

	input := []byte("prefix 4012888888881881x suffix")
	got := HTTPDataBytes(input)

	require.Equal(t, input, got)
}

func TestMatchesCardPatternReturnsFalseForUnknownPrefix(t *testing.T) {
	t.Parallel()

	require.False(t, matchesCardPattern([]byte("9111111111111111")))
}

func TestHTTPDataCreditCardsAdditionalBranches(t *testing.T) {
	t.Parallel()

	// Match and redact a standalone valid card number.
	require.Equal(t, []byte("***"), HTTPDataBytes([]byte("4012888888881881")))

	// Keep non-matching numeric run unchanged.
	require.Equal(t, []byte("9111111111111111"), HTTPDataBytes([]byte("9111111111111111")))

	// Exercise branch where current digit follows a word character.
	require.Equal(t, []byte("x123"), HTTPDataBytes([]byte("x123")))
}

func TestHTTPDataDigitWordBoundaryBranch(t *testing.T) {
	t.Parallel()

	input := []byte("(123x)")
	require.Equal(t, input, HTTPDataBytes(input))
}

func TestPassesLuhn(t *testing.T) {
	t.Parallel()

	// Known-valid card numbers satisfy the Luhn checksum.
	require.True(t, passesLuhn([]byte("4012888888881881")))
	require.True(t, passesLuhn([]byte("371449635398431")))
	require.True(t, passesLuhn([]byte("4222222222222")))

	// Altering the final check digit breaks the checksum.
	require.False(t, passesLuhn([]byte("4012888888881882")))
}

func TestLuhnCheckDefaultDisabled(t *testing.T) {
	t.Parallel()

	// The Luhn gate is off by default.
	require.False(t, LuhnCheckEnabled())

	digits := []byte("4012888888881882") // valid Visa prefix/length, invalid Luhn.

	// With the gate disabled, prefix match alone is enough to flag as a card.
	require.True(t, defaultRedactor.isCreditCard(digits))
}

//nolint:paralleltest // Mutates the process-wide Luhn toggle.
func TestSetLuhnCheckGatesRedaction(t *testing.T) {
	// Not parallel: SetLuhnCheck mutates process-wide state.
	t.Cleanup(func() { SetLuhnCheck(false) })

	// A run that matches a card prefix/length but fails Luhn.
	invalidLuhn := []byte("4012888888881882")

	// A run that matches a card prefix/length and passes Luhn.
	validLuhn := []byte("4012888888881881")

	SetLuhnCheck(true)
	require.True(t, LuhnCheckEnabled())

	// Enabled: only the Luhn-valid number is treated as a card.
	require.False(t, defaultRedactor.isCreditCard(invalidLuhn))
	require.True(t, defaultRedactor.isCreditCard(validLuhn))

	// And through the public redaction path.
	require.Equal(t, invalidLuhn, HTTPDataBytes(invalidLuhn))
	require.Equal(t, []byte("***"), HTTPDataBytes(validLuhn))

	SetLuhnCheck(false)
	require.False(t, LuhnCheckEnabled())

	// Disabled (default): both are redacted because prefix match alone suffices.
	require.Equal(t, []byte("***"), HTTPDataBytes(invalidLuhn))
	require.Equal(t, []byte("***"), HTTPDataBytes(validLuhn))
}

// TestHTTPDataGroupedCardNumbers covers card numbers written as digit groups
// separated by single spaces or dashes.
func TestHTTPDataGroupedCardNumbers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// Space- and dash-grouped cards are redacted as a whole span.
		{"4012 8888 8888 1881", "***"},
		{"4012-8888-8888-1881", "***"},
		{"pan: 4012 8888 8888 1881 end", "pan: *** end"},
		// Amex 4-6-5 grouping.
		{"3782 822463 10005", "***"},
		// Grouped card followed by further digit groups (e.g. an expiry): the
		// windowed scan still finds the card prefix span.
		{"4012 8888 8888 1881 12 26", "*** 12 26"},
		{"card 4111 1111 1111 1111 06 24 end", "card *** 06 24 end"},
		{"4222 2222 2222 2 99", "*** 99"}, // grouped 13-digit Visa + noise group
		// Grouped 19-digit UnionPay.
		{"6212 3456 7890 1234 567", "***"},
		// Wrong prefix: joined digits are not a card, left visible.
		{"1234 5678 9012 3456", "1234 5678 9012 3456"},
		// Group glued to a trailing word character is treated as an identifier.
		{"4012 8888 8888 1881x", "4012 8888 8888 1881x"},
		{"4222 2222 2222 2x", "4222 2222 2222 2x"},
		// No group boundary with 13-19 digits matches a card pattern.
		{"1234567890123 4567", "1234567890123 4567"},
		// First run alone exceeds the maximum card length (first-group overflow).
		{"12345678901234567890 1234", "12345678901234567890 1234"},
		// Joining the next group would exceed the maximum card length.
		{"1234567890123456 78901", "1234567890123456 78901"},
		// Double separator is not treated as a group join.
		{"4012  8888  8888  1881", "4012  8888  8888  1881"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestMatchesCardPatternExtendedPrefixes(t *testing.T) {
	t.Parallel()

	// UnionPay 62 (16-19 digits).
	require.True(t, matchesCardPattern([]byte("6212345678901234")))
	require.True(t, matchesCardPattern([]byte("62123456789012345")))
	require.True(t, matchesCardPattern([]byte("6212345678901234567")))
	// Discover 644-649 and 65 (16-19 digits).
	require.True(t, matchesCardPattern([]byte("6445000000000000")))
	require.True(t, matchesCardPattern([]byte("6490000000000000")))
	require.True(t, matchesCardPattern([]byte("6500000000000000002")))
	// JCB 35 (16-19 digits).
	require.True(t, matchesCardPattern([]byte("35660020203605055")))
	// Visa 19 digits (but not 17 or 18).
	require.True(t, matchesCardPattern([]byte("4111111111111111117")))
	require.False(t, matchesCardPattern([]byte("41111111111111111")))
	require.False(t, matchesCardPattern([]byte("411111111111111111")))
	// Mastercard is 16 digits only.
	require.False(t, matchesCardPattern([]byte("55555555555544445")))
	// 643 is below the 644-649 range and not otherwise a known prefix.
	require.False(t, matchesCardPattern([]byte("6430000000000000")))
	// 61 is not a known prefix.
	require.False(t, matchesCardPattern([]byte("6112345678901237")))
	// 15-digit run with no matching network prefix.
	require.False(t, matchesCardPattern([]byte("955555555554444")))
	// Length bounds.
	require.False(t, matchesCardPattern([]byte("411111111111")))         // 12 digits
	require.False(t, matchesCardPattern([]byte("41111111111111111174"))) // 20 digits
}

func TestHTTPDataLongCardNumbers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// 17-19 digit UnionPay and 19-digit Visa are redacted.
		{"6212345678901234567", "***"},
		{"62123456789012345", "***"},
		{"4111111111111111117", "***"},
		// Unknown prefixes and unsupported lengths stay visible.
		{"5212345678901234567", "5212345678901234567"},
		{"41234567890123456", "41234567890123456"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestMaestroCards(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// 16-digit Maestro numbers with well-known IINs are redacted by default.
		{"6759649826438453", "***"},
		{"5018000000000000", "***"},
		{"6304990000000000", "***"},
		// Grouped Maestro.
		{"6759 6498 2643 8453", "***"},
		// Broader Maestro ranges are NOT matched on prefix alone: 5099/6712 are
		// not well-known IINs (56xx 16-digit runs match the Mastercard range).
		{"5099999999999999", "5099999999999999"},
		{"6712345678901234", "6712345678901234"},
		// Short Maestro (12-15 digits) is not detected with the Luhn gate off.
		{"501800000009", "501800000009"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

//nolint:paralleltest // Mutates the process-wide Luhn toggle.
func TestMaestroShortLuhnGated(t *testing.T) {
	// Not parallel: SetLuhnCheck mutates process-wide state.
	t.Cleanup(func() { SetLuhnCheck(false) })

	shortValid12 := []byte("501800000009")    // 12-digit Maestro IIN, Luhn-valid
	shortValid15 := []byte("501800000000007") // 15-digit Maestro IIN, Luhn-valid
	shortInvalid := []byte("501800000008")    // 12-digit Maestro IIN, Luhn-invalid

	// Sanity-check the fixtures.
	require.True(t, passesLuhn(shortValid12))
	require.True(t, passesLuhn(shortValid15))
	require.False(t, passesLuhn(shortInvalid))

	// Gate off (default): short Maestro numbers stay visible.
	require.Equal(t, shortValid12, HTTPDataBytes(shortValid12))

	SetLuhnCheck(true)

	// Gate on: Luhn-valid short Maestro numbers are redacted...
	require.Equal(t, []byte("***"), HTTPDataBytes(shortValid12))
	require.Equal(t, []byte("***"), HTTPDataBytes(shortValid15))

	// ...while Luhn-invalid ones and non-Maestro short runs stay visible.
	require.Equal(t, shortInvalid, HTTPDataBytes(shortInvalid))
	require.Equal(t, []byte("991800000009"), HTTPDataBytes([]byte("991800000009")))
}

// TestGroupedCardLegacyPrefixesExcluded verifies the grouped-format scan does
// not match the 14-digit Diners and legacy 15-digit JCB (1800/2131) ranges,
// which collide with common phone-number formats, while contiguous runs and
// grouped Amex keep matching.
func TestGroupedCardLegacyPrefixesExcluded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// Phone-number formats must stay visible.
		{"call 1 800 555 0199 1234 now", "call 1 800 555 0199 1234 now"},
		{"1-800-555-0199-1234", "1-800-555-0199-1234"},
		{"ref 36 1234 5678 9012 x", "ref 36 1234 5678 9012 x"},
		{"2131 5550 1991 234", "2131 5550 1991 234"},
		// Amex stays grouped-detectable (active network, printed 4-6-5).
		{"3782 822463 10005", "***"},
		// Legacy ranges keep matching as contiguous runs.
		{"213100000000008", "***"},
		{"180000000000002", "***"},
		{"30569309025904", "***"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}
