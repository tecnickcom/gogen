package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPDataJWTLiterals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// JWS in free text, query strings, and JSON string values.
		{"jwt eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.dQw4w9WgXcQ end", "jwt *** end"},
		{"GET /cb?state=eyJhbGciOiJIUzI1NiJ9.abc.def", "GET /cb?state=***"},
		{`{"trace":"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig"}`, `{"trace":"***"}`},
		// Unsigned token (two segments, no signature).
		{"eyJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0", "***"},
		// JWE compact serialization (5 segments): all segments consumed.
		{"eyJhbGciOiJSU0EifQ.aaaa.bbbb.cccc.dddd end", "*** end"},
		// Base64url alphabet including '-' and '_' is consumed.
		{"eyJhbGciOiJIUzI1NiJ9.a-b_c.d-e_f", "***"},
		// Glued to a preceding word character: an identifier, not a JWT.
		{"xeyJhbGciOi.abc.def", "xeyJhbGciOi.abc.def"},
		// No dot structure or segments too short: prose stays visible.
		{"eyJ only", "eyJ only"},
		{"eyJhbGciOiJIUzI1NiJ9 nodot", "eyJhbGciOiJIUzI1NiJ9 nodot"},
		{"eyJhbGciOiJIUzI1NiJ9.ab x", "eyJhbGciOiJIUzI1NiJ9.ab x"}, // payload < min
		{"eyJab.cdef.ghij", "eyJab.cdef.ghij"},                     // header < min
		// Sentence punctuation after a token is preserved.
		{"token eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0. Next", "token ***. Next"},
		// 'e' at end of input.
		{"the e", "the e"},
		// "dir" JWE compact form: empty middle (encrypted-key) segment, further
		// segments follow — must be consumed whole.
		{"eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIn0..48V1iv.5eym8ct.XFBotag end", "*** end"},
		// Detached-payload JWS: empty middle, one signature segment follows.
		{"eyJhbGciOiJSUzI1NiJ9..signatureABCDEF end", "*** end"},
		// An empty middle with NO following segment is not a token.
		{"eyJhbGciOiJIUzI1NiJ9.. x", "eyJhbGciOiJIUzI1NiJ9.. x"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestHTTPDataJWTBoundaryAfterDigits(t *testing.T) {
	t.Parallel()

	// An "eyJ" glued to a preceding digit run reaches the dispatcher directly
	// (digits are scanned separately) and must be rejected as an identifier.
	input := "0eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.x"
	require.Equal(t, input, HTTPData(input))
}
