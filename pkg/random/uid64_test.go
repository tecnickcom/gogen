package random

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRnd_UID64(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64()
	b := r.UID64()

	require.NotEqual(t, a, b)
}

func TestRnd_UID64_Hex(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64().Hex()
	b := r.UID64().Hex()

	require.NotEqual(t, a, b)
	require.Len(t, a, 16)
	require.Len(t, b, 16)
}

func TestRnd_UID64_Format(t *testing.T) {
	t.Parallel()

	r := New(nil)
	u := r.UID64()

	// A pre-filled buffer must be fully overwritten.
	b := [16]byte{'x'}
	u.Format(&b)

	require.Equal(t, u.Hex(), string(b[:]))
	require.Len(t, b, 16)
}

func TestRnd_UID64_Byte(t *testing.T) {
	t.Parallel()

	r := New(nil)
	u := r.UID64()

	require.Equal(t, u.Hex(), string(u.Byte()))
	require.Len(t, u.Byte(), 16)
}

func TestRnd_UID64_String(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64().String()
	b := r.UID64().String()

	require.NotEqual(t, a, b)
}

// TestRnd_UID64_Golden pins the exact hexadecimal layout against a hand-computed
// vector: the value must render most-significant nibble first, which is what makes
// the hexadecimal form time-ordered. Comparing Format against Hex only compares
// Format with itself, since Hex is built on it.
func TestRnd_UID64_Golden(t *testing.T) {
	t.Parallel()

	u := TUID64(0x0123456789abcdef)

	const want = "0123456789abcdef"

	var dst [16]byte

	u.Format(&dst)

	require.Equal(t, want, string(dst[:]), "Format")
	require.Equal(t, want, string(u.Byte()), "Byte")
	require.Equal(t, want, u.Hex(), "Hex")
	require.Equal(t, strconv.FormatUint(uint64(u), 36), u.String(), "String")
}

// TestRnd_UID64_Layout asserts the documented split: the upper 32 bits are the
// seconds since the start of the current decade, and the lower 32 bits come from
// the configured reader. A century offset, a wrong shift, or an ignored reader
// would all pass the other tests.
func TestRnd_UID64_Layout(t *testing.T) {
	t.Parallel()

	entropy := []byte{0x01, 0x02, 0x03, 0x04}

	before := time.Now().UTC()
	u := New(bytes.NewReader(entropy)).UID64()
	after := time.Now().UTC()

	decadeStart := func(t time.Time) int64 {
		y := t.Year()

		return time.Date(y-y%10, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	}

	lo := uint64(before.Unix() - decadeStart(before))
	hi := uint64(after.Unix() - decadeStart(after))

	require.GreaterOrEqual(t, uint64(u)>>32, lo, "decade offset must not predate the call")
	require.LessOrEqual(t, uint64(u)>>32, hi, "decade offset must not postdate the call")

	// The offset must be a decade offset, not a raw Unix timestamp: a decade is
	// ~3.16e8 seconds, so it must stay far below 2^32.
	require.Less(t, uint64(u)>>32, uint64(1)<<32, "decade offset must fit in 32 bits")

	require.Equal(t, uint64(binary.LittleEndian.Uint32(entropy)), uint64(u)&0xFFFFFFFF,
		"low 32 bits must come from the configured reader")
}

func TestRnd_UID64_Hex_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fn := func() string {
		return r.UID64().Hex()
	}

	// Deliberately a small sample. UID64 has only 32 random bits per second, so
	// uniqueness is probabilistic, not guaranteed: at 1,000 identifiers this test
	// would fail spuriously about once in 10,000 runs (birthday bound n^2/2N over
	// 2^32). 100 identifiers puts that near 1e-6 while still exercising concurrent
	// generation and formatting. The collision behavior itself is documented on
	// [Rnd.UID64]; UID128 and UUIDv7 carry the real uniqueness guarantees and are
	// collision-tested at higher volume.
	collisionTest(t, fn, 10, 10)
}

func collisionTest(t *testing.T, f func() string, concurrency, iterations int) {
	t.Helper()

	total := concurrency * iterations

	idCh := make(chan string, total)
	defer close(idCh)

	// generators
	genWg := &sync.WaitGroup{}
	genWg.Add(concurrency)

	for range concurrency {
		go func() {
			defer genWg.Done()

			for range iterations {
				idCh <- f()
			}
		}()
	}

	// wait for generators to finish
	genWg.Wait()

	ids := make(map[string]bool, total)

	for range total {
		id, ok := <-idCh
		if !ok {
			t.Errorf("unexpected closed id channel")
			return
		}

		if _, exists := ids[id]; exists {
			t.Errorf("unexpected duplicate ID detected")
			return
		}

		// store generated id for duplicate detection
		ids[id] = true
	}
}
