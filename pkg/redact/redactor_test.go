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
