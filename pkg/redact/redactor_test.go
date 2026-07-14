package redact

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultReturnsSharedInstance verifies Default() hands every caller the one
// shared instance rather than a fresh one, which is what keeps the key-memo
// cache shared and bounded across the packages that fall back to it.
func TestDefaultReturnsSharedInstance(t *testing.T) {
	t.Parallel()

	first, second := Default(), Default()

	require.Same(t, defaultRedactor, first)
	require.Same(t, first, second)
}

// TestNewWithoutOptionsMatchesDefault verifies a plain New() instance behaves
// exactly like the shared [Default] one on representative inputs.
func TestNewWithoutOptionsMatchesDefault(t *testing.T) {
	t.Parallel()

	re := New()

	inputs := []string{
		benchmarkHTTPDataInput,
		string(benchmarkDigitHeavyInput),
		"dial error: postgres://app:hunter2@10.0.0.5/db",
		"state=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----\n",
		"<password>SECRET</password>",
		"push failed: ghp_AbCdEfGhIjKlMnOpQrStUvWxYz012345",
	}

	for _, in := range inputs {
		want := Default().String(in)
		require.Equal(t, want, re.String(in), "String mismatch for %q", in)
		require.Equal(t, want, string(re.Bytes([]byte(in))))
		require.Equal(t, want, re.BytesToString([]byte(in)))

		var dst []byte

		dst = re.AppendTo(dst, []byte(in))
		require.Equal(t, want, string(dst))

		var pooled string

		re.Pooled([]byte(in), func(out []byte) { pooled = string(out) })
		require.Equal(t, want, pooled)
	}

	require.NotPanics(t, func() { re.Pooled([]byte("x"), nil) })
}

// TestZeroValueRedactorDegradesGracefully verifies the exported zero value does
// not panic or emit an empty marker; it redacts correctly (with the default
// marker) on every rule path, including the non-ASCII key path that used to
// dereference a nil memo.
func TestZeroValueRedactorDegradesGracefully(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"password":"x"}`, `{"password":"***"}`}, // ASCII key: used to emit an empty marker
		{`{"pässword":"x"}`, `{"pässword":"x"}`},   // non-ASCII key: used to panic on the nil memo
		{"4012888888881881", "***"},                // card path
		{"Authorization: Bearer S\n", "Authorization: ***\n"},
		{"token=SECRET&note=ok", "token=***&note=ok"},
	}

	for _, tc := range cases {
		var zero Redactor

		require.NotPanicsf(t, func() {
			require.Equal(t, tc.want, zero.String(tc.input), "input: %s", tc.input)
		}, "input: %s", tc.input)
	}
}

// TestNewDoesNotEagerlyAllocateMemo verifies the ~200 KB key-memo map is
// allocated only when a non-ASCII key is actually classified, so a fresh
// instance driven over ASCII-only input carries no cache.
func TestNewDoesNotEagerlyAllocateMemo(t *testing.T) {
	t.Parallel()

	re := New()
	require.Nil(t, re.keyMemo.data, "New() must not pre-allocate the memo map")

	re.String(`{"password":"x","user":"bob","trace":"abc123"}`)
	require.Nil(t, re.keyMemo.data, "an ASCII-only workload must never touch the memo")

	re.String(`{"pässword":"x"}`)
	require.NotNil(t, re.keyMemo.data, "a non-ASCII key must lazily allocate the memo")
}

func TestRedactorConcurrentUse(t *testing.T) {
	t.Parallel()

	re := New(WithMarker("[X]"), WithExtraTokens("floof"))

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 100 {
				assert.Equal(t, "floof=[X]&päss=ok", re.String("floof=SECRET&päss=ok"))
			}
		})
	}

	wg.Wait()
}

// TestRedactorConvergenceWithCustomMarker verifies the convergence property
// holds for a non-default marker on the pathological inputs from the fuzz
// corpus.
func TestRedactorConvergenceWithCustomMarker(t *testing.T) {
	t.Parallel()

	re := New(WithMarker("#REDACTED#"))

	docs := []string{
		`sid=0"sid":0`,
		"\"pass\":{\",\"Card\":\"{\"\"}",
		`token=""=`,
		"pass=/<pass=",
	}

	for _, doc := range docs {
		once := re.String(doc)
		twice := re.String(once)
		require.Equal(t, twice, re.String(twice), "no fixed point after two passes for input: %q", doc)
	}
}

// TestUsableRedactorPartialZeroValue covers the individual field-defaulting
// branches of a partially-initialized Redactor.
func TestUsableRedactorPartialZeroValue(t *testing.T) {
	t.Parallel()

	// Only the marker set: the key memo is defaulted so a non-ASCII key does
	// not panic, and redaction still works.
	markerOnly := &Redactor{marker: []byte("XX")}
	require.Equal(t, "token=XX", markerOnly.String("token=SECRET"))
	require.NotPanics(t, func() { markerOnly.String(`{"pässword":"x"}`) })

	// Only the key memo set: the marker is defaulted to the standard one.
	memoOnly := &Redactor{keyMemo: newSensitiveKeyMemo()}
	require.Equal(t, "token=***", memoOnly.String("token=SECRET"))
}

// TestUsableRedactorReturnsCompleteReceiver covers the fast path: an instance
// built by New already has both the marker and the key memo, so it is returned
// as-is and keeps its shared (caching) memo instead of a per-call copy.
func TestUsableRedactorReturnsCompleteReceiver(t *testing.T) {
	t.Parallel()

	re := New(WithMarker("#"))
	require.Same(t, re, re.usableRedactor())
}

// TestRedactorEntryPointsAgree pins the contract that the five entry points are
// the same engine differing only in input and output handling: String is the
// reference, and every other one must produce identical bytes.
func TestRedactorEntryPointsAgree(t *testing.T) {
	t.Parallel()

	input := benchmarkHTTPDataInput
	want := Default().String(input)

	require.Equal(t, want, string(Default().Bytes([]byte(input))))
	require.Equal(t, want, Default().BytesToString([]byte(input)))

	var dst []byte

	dst = Default().AppendTo(dst, []byte(input))
	require.Equal(t, want, string(dst))

	var pooled string

	Default().Pooled([]byte(input), func(out []byte) { pooled = string(out) })
	require.Equal(t, want, pooled)

	require.NotPanics(t, func() { Default().Pooled([]byte(input), nil) })
}

func TestRedactorString(t *testing.T) {
	t.Parallel()

	input := "Authorization: Bearer SECRET\npassword=SECRET&reference=VISIBLE"
	want := expectedRedaction("Authorization: ***\npassword=***&reference=VISIBLE")

	require.Equal(t, want, Default().String(input))
}

func TestRedactorBytes(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\napiKey=SECRET&reference=VISIBLE\n{\"password\":\"SECRET\",\"reference\":\"VISIBLE\"}")
	want := []byte(expectedRedaction("Authorization: ***\napiKey=***&reference=VISIBLE\n{\"password\":\"***\",\"reference\":\"VISIBLE\"}"))

	got := Default().Bytes(input)
	require.Equal(t, want, got)
}

func TestRedactorBytesMatchesString(t *testing.T) {
	t.Parallel()

	input := benchmarkHTTPDataInput
	require.Equal(t, Default().String(input), string(Default().Bytes([]byte(input))))
}

func TestRedactorBytesToString(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\npassword=SECRET&reference=VISIBLE")
	want := "Authorization: ***\npassword=***&reference=VISIBLE"

	got := Default().BytesToString(input)
	require.Equal(t, want, got)
}

func TestRedactorAppendTo(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\napiKey=SECRET&reference=VISIBLE\n{\"password\":\"SECRET\",\"reference\":\"VISIBLE\"}")
	want := []byte(expectedRedaction("Authorization: ***\napiKey=***&reference=VISIBLE\n{\"password\":\"***\",\"reference\":\"VISIBLE\"}"))

	dst := make([]byte, 0, len(input))
	got := Default().AppendTo(dst, input)
	require.Equal(t, want, got)
}

func TestRedactorAppendToResetsDestination(t *testing.T) {
	t.Parallel()

	input := []byte("password=SECRET")
	dst := []byte("prefix should be overwritten")

	got := Default().AppendTo(dst, input)
	require.Equal(t, []byte("password=***"), got)
}

// TestRedactorAppendToInPlaceAliasing covers backingOverlap: an in-place
// AppendTo(b, b) — where the destination and source share storage — must produce
// the same output as a clean redaction, instead of corrupting the buffer and
// leaking a secret whose key the write cursor overran.
func TestRedactorAppendToInPlaceAliasing(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"pin=1&password=SUPERSECRET",
		"a=1&b=2&password=SUPERSECRET",
		"cvv=1&cvv=2&token=SUPERSECRET&x=9",
		benchmarkHTTPDataInput,
	}

	for _, in := range inputs {
		buf := make([]byte, 0, len(in)+16)
		buf = append(buf, in...)

		require.Equal(t, Default().String(in), string(Default().AppendTo(buf, buf)), "in-place alias for %q", in)
	}
}

func TestRedactorPooled(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\napiKey=SECRET&reference=VISIBLE")
	want := []byte("Authorization: ***\napiKey=***&reference=VISIBLE")

	var got []byte

	Default().Pooled(input, func(out []byte) {
		got = append([]byte(nil), out...)
	})

	require.Equal(t, want, got)
}

func TestRedactorPooledNilConsumer(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		Default().Pooled([]byte("password=SECRET"), nil)
	})
}
