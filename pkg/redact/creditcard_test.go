package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactCreditCardWordBoundary(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

func TestRedactCreditCardsKeepsDigitsAdjacentToWordChar(t *testing.T) {
	t.Parallel()

	input := []byte("prefix 4012888888881881x suffix")
	got := Default().Bytes(input)

	require.Equal(t, input, got)
}

func TestMatchesCardPatternReturnsFalseForUnknownPrefix(t *testing.T) {
	t.Parallel()

	require.False(t, matchesCardPattern([]byte("9111111111111111")))
}

func TestRedactCreditCardsAdditionalBranches(t *testing.T) {
	t.Parallel()

	// Match and redact a standalone valid card number.
	require.Equal(t, []byte("***"), Default().Bytes([]byte("4012888888881881")))

	// Keep non-matching numeric run unchanged.
	require.Equal(t, []byte("9111111111111111"), Default().Bytes([]byte("9111111111111111")))

	// Exercise branch where current digit follows a word character.
	require.Equal(t, []byte("x123"), Default().Bytes([]byte("x123")))
}

func TestRedactDigitWordBoundaryBranch(t *testing.T) {
	t.Parallel()

	input := []byte("(123x)")
	require.Equal(t, input, Default().Bytes(input))
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

	digits := []byte("4012888888881882") // valid Visa prefix/length, invalid Luhn.

	// The gate is off by default, so prefix match alone flags a card, both on
	// the package default and on a plain instance.
	require.True(t, defaultRedactor.isCreditCard(digits))
	require.True(t, New().isCreditCard(digits))
}

// TestWithLuhnCheckGatesRedaction covers the gate on a configured instance:
// with it enabled, a card prefix alone is no longer enough.
func TestWithLuhnCheckGatesRedaction(t *testing.T) {
	t.Parallel()

	// A run that matches a card prefix/length but fails Luhn.
	invalidLuhn := []byte("4012888888881882")

	// A run that matches a card prefix/length and passes Luhn.
	validLuhn := []byte("4012888888881881")

	strict := New(WithLuhnCheck(true))

	// Enabled: only the Luhn-valid number is treated as a card.
	require.False(t, strict.isCreditCard(invalidLuhn))
	require.True(t, strict.isCreditCard(validLuhn))

	// And through the public redaction path.
	require.Equal(t, invalidLuhn, strict.Bytes(invalidLuhn))
	require.Equal(t, []byte("***"), strict.Bytes(validLuhn))

	// Disabled (default): both are redacted because prefix match alone suffices.
	require.Equal(t, []byte("***"), Default().Bytes(invalidLuhn))
	require.Equal(t, []byte("***"), Default().Bytes(validLuhn))
}

// TestRedactGroupedCardNumbers covers card numbers written as digit groups
// separated by single spaces or dashes.
func TestRedactGroupedCardNumbers(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
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

func TestRedactLongCardNumbers(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

func TestMaestroShortLuhnGated(t *testing.T) {
	t.Parallel()

	shortValid12 := []byte("501800000009")    // 12-digit Maestro IIN, Luhn-valid
	shortValid15 := []byte("501800000000007") // 15-digit Maestro IIN, Luhn-valid
	shortInvalid := []byte("501800000008")    // 12-digit Maestro IIN, Luhn-invalid

	// Sanity-check the fixtures.
	require.True(t, passesLuhn(shortValid12))
	require.True(t, passesLuhn(shortValid15))
	require.False(t, passesLuhn(shortInvalid))

	// Gate off (default): short Maestro numbers stay visible.
	require.Equal(t, shortValid12, Default().Bytes(shortValid12))

	strict := New(WithLuhnCheck(true))

	// Gate on: Luhn-valid short Maestro numbers are redacted...
	require.Equal(t, []byte("***"), strict.Bytes(shortValid12))
	require.Equal(t, []byte("***"), strict.Bytes(shortValid15))

	// ...while Luhn-invalid ones and non-Maestro short runs stay visible.
	require.Equal(t, shortInvalid, strict.Bytes(shortInvalid))
	require.Equal(t, []byte("991800000009"), strict.Bytes([]byte("991800000009")))
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

// TestMastercard16Digits covers the 16-digit-only Mastercard ranges: the classic
// 51-57 series and the 2-series (2221-2720, approximated as a 21-27 second digit).
func TestMastercard16Digits(t *testing.T) {
	t.Parallel()

	// 51-57 series.
	require.True(t, matchesCardPattern([]byte("5555555555554444")))
	require.True(t, matchesCardPattern([]byte("5105105105105100")))
	require.True(t, matchesCardPattern([]byte("5712345678901234")))
	// 2-series.
	require.True(t, matchesCardPattern([]byte("2221000000000009")))
	require.True(t, matchesCardPattern([]byte("2720999999999996")))
	// Second digit outside 1-7 is not a Mastercard prefix.
	require.False(t, matchesCardPattern([]byte("5899999999999999")))
	require.False(t, matchesCardPattern([]byte("2099999999999999")))
	require.False(t, matchesCardPattern([]byte("2899999999999999")))
	// Both series are 16 digits only.
	require.False(t, matchesCardPattern([]byte("22210000000000098")))
	require.False(t, matchesCardPattern([]byte("571234567890123")))

	// End-to-end, in free text with no sensitive key to trigger the key rules.
	cases := []struct {
		input string
		want  string
	}{
		{"paid with 5555555555554444 today", "paid with *** today"},
		{"paid with 2221000000000009 today", "paid with *** today"},
		{"5555 5555 5555 4444", "***"},                         // grouped
		{"ref 5899999999999999 x", "ref 5899999999999999 x"},   // not a card prefix
		{"ref 22210000000000098 x", "ref 22210000000000098 x"}, // 17 digits
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}
