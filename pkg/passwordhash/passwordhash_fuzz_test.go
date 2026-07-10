package passwordhash

import (
	"errors"
	"testing"

	"github.com/tecnickcom/gogen/pkg/encode"
)

// requireSentinel fails the test if err is not classified by one of the
// package's exported sentinel errors, so callers can rely on errors.Is routing
// for every possible failure.
func requireSentinel(t *testing.T, err error) {
	t.Helper()

	for _, sentinel := range []error{
		ErrPasswordTooShort,
		ErrPasswordTooLong,
		ErrInvalidParams,
		ErrInvalidHashData,
		ErrAlgoMismatch,
		ErrVersionMismatch,
		ErrInvalidPepperKey,
	} {
		if errors.Is(err, sentinel) {
			return
		}
	}

	t.Fatalf("error not classified by a package sentinel: %v", err)
}

// assertRobust asserts the failure invariants shared by the verification and
// rehash-check paths: every error is matchable with a package sentinel, and a
// failed call never reports a positive result.
func assertRobust(t *testing.T, what string, positive bool, err error) {
	t.Helper()

	if err == nil {
		return
	}

	requireSentinel(t, err)

	if positive {
		t.Fatalf("%s reported a positive result together with an error", what)
	}
}

// fuzzCostAboveBudget reports whether hash decodes to Argon2 parameters more
// expensive than the fuzz budget. The verify bounds are deliberately generous
// (up to 4 GiB and 1024 passes); running those costs during fuzzing would only
// throttle throughput, and the bounds themselves are exercised by the unit
// tests. Threads are not bounded: the Argon2 cost is driven by memory and
// passes, not lanes.
func fuzzCostAboveBudget(hash string) bool {
	// Mirror the production ordering: the entry points reject oversized strings
	// before decoding, so an oversized input is cheap for them and must not be
	// pre-decoded here just to decide whether to skip it.
	if len(hash) > maxHashLen {
		return false
	}

	// Probe with the same format-detecting decoder the entry points use, so a
	// costly PHC blob is skipped just like a costly JSON one.
	probe, err := decodeHash(hash)
	if err != nil || probe.Params == nil {
		return false
	}

	return probe.Params.Memory > DefaultMemory || probe.Params.Time > 2*DefaultTime
}

// FuzzPasswordVerify feeds arbitrary candidate passwords and stored-hash blobs
// to the verification and rehash-check paths — plain and pepper-encrypted —
// asserting the package's core robustness invariants: no input may cause a
// panic, a failed call never reports a positive result, and every error is
// matchable with a package sentinel.
func FuzzPasswordVerify(f *testing.F) {
	p := New()

	// A fixed valid 16-byte AES key for the pepper (Encrypt*) paths. The fuzz
	// engine cannot forge a ciphertext that authenticates under it, so arbitrary
	// input fails AES-GCM fast; only the seeded ciphertext decrypts, keeping the
	// argon2 cost of the Encrypt paths bounded.
	pepperKey := []byte("0123456789012345")

	// Well-formed encoding with out-of-range embedded parameters.
	badParamsBlob, err := encode.Serialize(&Hashed{Params: &Params{}})
	if err != nil {
		f.Fatal(err)
	}

	// A valid pepper-encrypted hash of a known password, minted cheaply so the
	// Encrypt success path stays fast during fuzzing.
	cheap := New(WithMemory(minMemory), WithTime(minTime), WithThreads(1))

	encHash, err := cheap.EncryptPasswordHash(pepperKey, "Test-Password-01234")
	if err != nil {
		f.Fatal(err)
	}

	f.Add("test", testRefHash)
	f.Add("Test-Password-01234", "wrong-hash")
	f.Add("", "")
	f.Add("Test-Password-01234", badParamsBlob)
	f.Add("Test-Password-01234", testRefHash[:len(testRefHash)/2])
	f.Add("Test-Password-01234", encHash)
	// PHC-format seeds: a valid reference and a truncated one, so the fuzzer
	// explores the auto-detected '$'-prefixed decode path as well.
	f.Add("test", testRefHashPHC)
	f.Add("Test-Password-01234", testRefHashPHC[:len(testRefHashPHC)/2])

	f.Fuzz(func(t *testing.T, password, hash string) {
		if fuzzCostAboveBudget(hash) {
			t.Skip("argon2 cost above fuzz budget")
		}

		ok, verr := p.PasswordVerify(password, hash)
		assertRobust(t, "PasswordVerify", ok, verr)

		need, rerr := p.PasswordNeedsRehash(hash)
		assertRobust(t, "PasswordNeedsRehash", need, rerr)

		eok, everr := p.EncryptPasswordVerify(pepperKey, password, hash)
		assertRobust(t, "EncryptPasswordVerify", eok, everr)

		eneed, ererr := p.EncryptPasswordNeedsRehash(pepperKey, hash)
		assertRobust(t, "EncryptPasswordNeedsRehash", eneed, ererr)
	})
}
