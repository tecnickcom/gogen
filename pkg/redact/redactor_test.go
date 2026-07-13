package redact

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultReturnsPackageInstance(t *testing.T) {
	t.Parallel()

	require.Same(t, defaultRedactor, Default())
	require.Equal(t, String(benchmarkHTTPDataInput), Default().String(benchmarkHTTPDataInput))
}

// TestNewWithoutOptionsMatchesPackageFunctions verifies a plain New() instance
// behaves exactly like the package-level API on representative inputs.
func TestNewWithoutOptionsMatchesPackageFunctions(t *testing.T) {
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
		want := String(in)
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
