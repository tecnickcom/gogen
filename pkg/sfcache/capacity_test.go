// Tests for the capacity: eviction, the victim it chooses, and what it may not take.

package sfcache

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_evict_expired(t *testing.T) {
	t.Parallel()

	r := New(nopLookupFn, Config{Size: 3, TTL: 1 * time.Minute})

	seed(r, map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now().Add(-2 * time.Second),
		},
		"example.org": {
			expireAt: time.Now().Add(11 * time.Second),
		},
		"example.net": {
			expireAt: time.Now().Add(13 * time.Second),
		},
	})

	require.Equal(t, 3, r.Len())

	evictOne(r, evictValue)

	require.Equal(t, 2, r.Len())
	require.Contains(t, r.keymap, "example.org")
	require.Contains(t, r.keymap, "example.net")
}

func Test_evict_oldest(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 3, TTL: 1 * time.Second})

	seed(c, map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now().Add(11 * time.Second),
		},
		"example.org": {
			expireAt: time.Now().Add(7 * time.Second),
		},
		"example.net": {
			expireAt: time.Now().Add(13 * time.Second),
		},
	})

	evictOne(c, evictValue)

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.net")
}

func Test_set(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 2, TTL: 10 * time.Second})

	c.set("example.com", []string{"192.0.2.1"}, nil, c.ttl)
	time.Sleep(1 * time.Second)
	c.set("example.org", []string{"192.0.2.2", "198.51.100.2"}, nil, c.ttl)

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.org")

	c.set("example.net", []string{"192.0.2.3", "198.51.100.3", "203.0.113.3"}, nil, c.ttl)

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.org")
	require.Contains(t, c.keymap, "example.net")

	c.set("example.net", []string{"198.51.100.4"}, nil, c.ttl)

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.org")
	require.Contains(t, c.keymap, "example.net")
	require.Equal(t, []string{"198.51.100.4"}, c.keymap["example.net"].val)
}

// Test_New_effectively_unbounded_size pins that an enormous Size is a valid way to
// configure a cache that never evicts. Size is a bound, not a reservation: the
// queues must grow into it rather than reserve it up front, which for a Size near
// math.MaxInt would not merely waste memory but fail outright.
func Test_New_effectively_unbounded_size(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: math.MaxInt, TTL: time.Minute})

	for i := range 100 {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("key%d", i))
		require.NoError(t, err)
	}

	require.Equal(t, 100, c.Len(), "nothing may be evicted from a cache that cannot be full")

	requireConsistentAccounting(t, c)
}

// Test_evict_cannot_touch_an_inflight_lookup pins that eviction can never break
// single-flight deduplication: a lookup in flight is not an entry, so there is
// nothing for evict to take.
func Test_evict_cannot_touch_an_inflight_lookup(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 2, TTL: 1 * time.Minute})

	seed(c, map[string]*entry[any]{
		"cached.example.com": {
			expireAt: time.Now().Add(30 * time.Second),
		},
	})

	fl := seedFlight(c, "inflight.example.com")

	require.True(t, evictOne(c, evictValue))
	require.NotContains(t, c.keymap, "cached.example.com")

	// Nothing is left to evict: the in-flight lookup is not an entry.
	require.False(t, evictOne(c, evictValue), "an in-flight lookup must not be evictable")
	require.True(t, inFlight(c, "inflight.example.com"), "eviction must not disturb a flight")

	requireConsistentAccounting(t, c)

	fl.finish()
}

func Test_Lookup_capacity_preserves_inflight(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var (
		mu    sync.Mutex
		calls = map[string]int{}
	)

	lookupFn := func(_ context.Context, key string) (any, error) {
		mu.Lock()
		calls[key]++
		mu.Unlock()

		if key == "slow.example.com" {
			<-release // hold the in-flight lookup open
		}

		return key, nil
	}

	c := New(lookupFn, Config{Size: 1, TTL: 1 * time.Minute})

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(context.Background(), "slow.example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "slow.example.com", v)
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		return inFlight(c, "slow.example.com")
	}, time.Second, time.Millisecond)

	// Fill the cache beyond capacity with another key:
	// the in-flight lookup must survive it.
	v, err := c.Lookup(t.Context(), "fast.example.com")
	require.NoError(t, err)
	require.Equal(t, "fast.example.com", v)

	require.True(t, inFlight(c, "slow.example.com"), "the in-flight lookup must survive eviction at capacity")

	// A duplicate call must coalesce onto the in-flight lookup.
	dupDone := make(chan struct{})

	go func() {
		defer close(dupDone)

		v, err := c.Lookup(context.Background(), "slow.example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "slow.example.com", v)
	}()

	waitForParkedLookupWaiter(t, "capacity_preserves_inflight")
	close(release)

	<-prodDone
	<-dupDone

	mu.Lock()
	defer mu.Unlock()

	require.Equal(t, 1, calls["slow.example.com"], "duplicate in-flight lookups must be coalesced")
}

// Test_Lookup_capacity_overshoot_reclaimed verifies that the capacity
// overshoot caused by more concurrent distinct in-flight keys than `size` is
// reclaimed once the lookups complete, instead of pinning the map at the
// burst high-water mark forever.
func Test_Lookup_capacity_overshoot_reclaimed(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, key string) (any, error) {
		<-release // hold all burst lookups open concurrently

		return key, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	const burst = 8

	wg := &sync.WaitGroup{}

	for i := range burst {
		wg.Go(func() {
			_, _ = c.Lookup(context.Background(), fmt.Sprintf("burst%d", i))
		})
	}

	// Wait until all in-flight lookups exceed the capacity.
	require.Eventually(t, func() bool {
		return c.Len() == burst
	}, 5*time.Second, time.Millisecond)

	close(release)
	wg.Wait()

	// The overshoot must be reclaimed as the lookups publish their results.
	require.LessOrEqual(t, c.Len(), 2, "capacity overshoot must be reclaimed after the burst")

	// And the cache must stay within capacity on subsequent inserts.
	for i := range 5 {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("post%d", i))
		require.NoError(t, err)
		require.LessOrEqual(t, c.Len(), 2)
	}
}

// Test_Lookup_evicts_error_residue_first verifies through the public API that
// at capacity the expired residue of a failed lookup is evicted before an
// older but still valid entry.
func Test_Lookup_evicts_error_residue_first(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, key string) (any, error) {
		if key == "bad.example.com" {
			return nil, errors.New("mock error")
		}

		return key, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	_, err := c.Lookup(t.Context(), "bad.example.com") // leaves an expired residue
	require.Error(t, err)

	v, err := c.Lookup(t.Context(), "good.example.com")
	require.NoError(t, err)
	require.Equal(t, "good.example.com", v)

	// At capacity, the expired residue must be evicted before the valid entry.
	_, err = c.Lookup(t.Context(), "new.example.com")
	require.NoError(t, err)

	c.mux.RLock()
	_, hasGood := c.keymap["good.example.com"]
	_, hasBad := c.keymap["bad.example.com"]
	c.mux.RUnlock()

	require.True(t, hasGood, "the valid entry must survive eviction")
	require.False(t, hasBad, "the expired error residue must be evicted first")
}

// Test_Lookup_failed_lookup_does_not_evict_a_valid_entry pins that a key which
// is merely attempted cannot cost the cache a healthy value: neither the
// in-flight lookup nor the error residue may evict a valid entry, so the
// cache goes over capacity instead and reclaims the excess when the next value
// is stored.
func Test_Lookup_failed_lookup_does_not_evict_a_valid_entry(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, key string) (any, error) {
		calls.Add(1)

		if key == "bad.example.com" {
			<-release // hold the failing flight open

			return nil, errors.New("mock error")
		}

		return key, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	for _, key := range []string{"good1.example.com", "good2.example.com"} {
		_, err := c.Lookup(t.Context(), key)
		require.NoError(t, err)
	}

	require.Equal(t, 2, c.Len())

	failed := make(chan struct{})

	go func() {
		defer close(failed)

		_, err := c.Lookup(context.Background(), "bad.example.com")
		assert.Error(t, err) // assert (not require) must be used off the test goroutine
	}()

	// While the failing lookup is still in flight, and its outcome is
	// therefore unknown, both valid entries must still be there.
	require.Eventually(t, func() bool {
		return c.Len() == 3
	}, time.Second, time.Millisecond, "the flight must overshoot the capacity, not evict a valid entry")

	c.mux.RLock()
	_, hasGood1 := c.keymap["good1.example.com"]
	_, hasGood2 := c.keymap["good2.example.com"]
	c.mux.RUnlock()

	require.True(t, hasGood1, "an in-flight lookup must not evict a valid entry")
	require.True(t, hasGood2, "an in-flight lookup must not evict a valid entry")

	close(release)
	<-failed

	// The error residue must not have evicted a valid entry either.
	c.mux.RLock()
	_, hasGood1 = c.keymap["good1.example.com"]
	_, hasGood2 = c.keymap["good2.example.com"]
	c.mux.RUnlock()

	require.True(t, hasGood1, "a failed lookup must not evict a valid entry")
	require.True(t, hasGood2, "a failed lookup must not evict a valid entry")

	// Both keys are still served from the cache: no lookup is re-run.
	before := calls.Load()

	for _, key := range []string{"good1.example.com", "good2.example.com"} {
		v, err := c.Lookup(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key, v)
	}

	require.Equal(t, before, calls.Load(), "the valid entries must still be cached")

	// Storing a new value reclaims the excess: the expired error residue is
	// evicted first, and the cache is back within capacity.
	_, err := c.Lookup(t.Context(), "good3.example.com")
	require.NoError(t, err)

	require.Equal(t, 2, c.Len(), "the next stored value must reclaim the excess")

	c.mux.RLock()
	_, hasBad := c.keymap["bad.example.com"]
	c.mux.RUnlock()

	require.False(t, hasBad, "the error residue must be evicted before a valid entry")
}

// Test_Lookup_failing_keys_stay_bounded verifies that a stream of distinct
// failing keys cannot grow the cache without bound, even though a failed
// lookup never evicts a valid entry: expired residue is still reclaimed.
func Test_Lookup_failing_keys_stay_bounded(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, _ string) (any, error) {
		return nil, errors.New("mock error")
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	for i := range 20 {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("bad%d.example.com", i))
		require.Error(t, err)

		require.LessOrEqual(t, c.Len(), 3, "error residue must be reclaimed")
	}
}

// Test_Lookup_inflight_lookups_do_not_evict_values pins that the capacity
// bounds the values the cache holds, not the lookups it has in flight: keys
// being resolved concurrently must overshoot the capacity, never push values
// out of it.
func Test_Lookup_inflight_lookups_do_not_evict_values(t *testing.T) {
	t.Parallel()

	const (
		size = 4
		cold = 16 // distinct keys held in flight, far past the capacity
	)

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, key string) (any, error) {
		calls.Add(1)

		if strings.HasPrefix(key, "cold") {
			<-release // park the cold flights

			return nil, errors.New("mock error")
		}

		return key, nil
	}

	c := New(lookupFn, Config{Size: size, TTL: time.Minute})

	var wg sync.WaitGroup

	for i := range cold {
		wg.Go(func() {
			_, _ = c.Lookup(context.Background(), fmt.Sprintf("cold%d.example.com", i))
		})
	}

	require.Eventually(t, func() bool {
		return c.Len() == cold
	}, time.Second, time.Millisecond, "every cold lookup must be in flight")

	// Store a full capacity worth of values while the cold lookups are in flight.
	for i := range size {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("hot%d.example.com", i))
		require.NoError(t, err)
	}

	require.Equal(t, size+cold, c.Len(), "in-flight lookups must overshoot the capacity, not evict")

	close(release)
	wg.Wait()

	// Every value must still be served from the cache: no lookup is re-run.
	before := calls.Load()

	for i := range size {
		key := fmt.Sprintf("hot%d.example.com", i)

		v, err := c.Lookup(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key, v)
	}

	require.Equal(t, before, calls.Load(), "concurrent lookups must not have evicted any value")
	require.LessOrEqual(t, c.Len(), size+1, "the excess must be reclaimed as the flights complete")

	requireConsistentAccounting(t, c)
}

// Test_evict_prefers_worthless_entries pins the victim order: an expired entry
// holding nothing worth keeping goes first, then a value that is only being
// served stale, and only then a valid entry. Each eviction level draws its own
// line across that order, and a store may never cross the line of its level.
func Test_evict_prefers_worthless_entries(t *testing.T) {
	t.Parallel()

	seeded := func() *Cache[string, any] {
		c := New(nopLookupFn, Config{Size: 3, TTL: time.Minute})

		seed(c, map[string]*entry[any]{
			"residue.example.com": {err: errors.New("mock error")},
			"stale.example.com":   {val: "stale", staleUntil: time.Now().Add(time.Minute)},
			"valid.example.com":   {val: "valid", expireAt: time.Now().Add(time.Minute)},
		})

		return c
	}

	// Repeated because the map iteration order is random: a victim chosen by
	// preference wins every time, one chosen by accident would not.
	for range 64 {
		c := seeded()

		require.True(t, evictOne(c, evictValue))
		require.NotContains(t, c.keymap, "residue.example.com", "worthless error residue must go first")
		require.Contains(t, c.keymap, "stale.example.com")
		require.Contains(t, c.keymap, "valid.example.com")

		require.True(t, evictOne(c, evictValue))
		require.NotContains(t, c.keymap, "stale.example.com", "a stale value must go before a valid entry")
		require.Contains(t, c.keymap, "valid.example.com")

		require.True(t, evictOne(c, evictValue))
		require.Empty(t, c.keymap)

		require.False(t, evictOne(c, evictValue), "an empty cache has nothing to evict")
	}

	// evictWorthless: the residue of a failed lookup adds no value, so it may
	// cost the cache none.
	c := seeded()

	require.True(t, evictOne(c, evictWorthless))
	require.NotContains(t, c.keymap, "residue.example.com")

	require.False(t, evictOne(c, evictWorthless), "evictWorthless must not take a stale or a valid entry")
	require.Contains(t, c.keymap, "stale.example.com")
	require.Contains(t, c.keymap, "valid.example.com")

	// evictStale: a stale revive does store a value, so it must pay for it — but
	// the refresh it comes from FAILED, so it may not take a live value.
	for range 64 {
		c = seeded()

		require.True(t, evictOne(c, evictStale))
		require.NotContains(t, c.keymap, "residue.example.com", "worthless error residue must go first")

		require.True(t, evictOne(c, evictStale), "evictStale must be able to take a merely-stale value")
		require.NotContains(t, c.keymap, "stale.example.com")

		require.False(t, evictOne(c, evictStale), "evictStale must never take a valid entry")
		require.Contains(t, c.keymap, "valid.example.com")
	}
}

// Test_Lookup_stale_revive_does_not_evict_a_valid_entry pins that reviving a
// stale value is a failure outcome, not a cacheable one: at capacity it must
// leave the cache over capacity rather than take a valid entry.
func Test_Lookup_stale_revive_does_not_evict_a_valid_entry(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, key string) (any, error) {
		if key != "stale.example.com" {
			return key, nil
		}

		if calls.Add(1) == 1 {
			return "good", nil // prime the value that will be served stale
		}

		<-release // hold the failing refresh open while the values are cached

		return nil, errors.New("mock refresh failure")
	}

	// The values to protect never expire; the stale key expires immediately.
	ttlFn := func(key string, _ any) time.Duration {
		if key == "stale.example.com" {
			return time.Millisecond
		}

		return time.Hour
	}

	c := New(lookupFn, Config{Size: 2, MaxStaleOnFailure: time.Minute}, WithTTLFunc(ttlFn))

	v, err := c.Lookup(t.Context(), "stale.example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	time.Sleep(10 * time.Millisecond) // let it expire

	done := make(chan struct{})

	go func() {
		defer close(done)

		v, err := c.Lookup(context.Background(), "stale.example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "good", v, "the failed refresh must serve the stale value")
	}()

	// Wait until the refresh flight has registered its flight, so that the
	// values below are stored while it is still in the air.
	require.Eventually(t, func() bool {
		return inFlight(c, "stale.example.com")
	}, time.Second, time.Millisecond)

	for _, key := range []string{"valid1.example.com", "valid2.example.com"} {
		_, err := c.Lookup(t.Context(), key)
		require.NoError(t, err)
	}

	close(release)
	<-done

	c.mux.RLock()
	_, hasValid1 := c.keymap["valid1.example.com"]
	_, hasValid2 := c.keymap["valid2.example.com"]
	c.mux.RUnlock()

	require.True(t, hasValid1, "reviving a stale value must not evict a valid entry")
	require.True(t, hasValid2, "reviving a stale value must not evict a valid entry")
	require.Equal(t, 3, c.Len(), "the revive must overshoot the capacity instead")

	requireConsistentAccounting(t, c)
}

// Test_worthlessVictim pins the only entry a FAILED lookup is allowed to take: one
// that nobody can ever be served. It must find one whenever the cache holds one — a
// false negative would let error residue accumulate without bound — and must name
// nothing otherwise, because naming a value a caller could still be served would
// destroy it.
//
// The three questions it answers are three queue heads (residue, stale, values), so
// each answer is exact and costs one comparison — where the machinery this replaced
// approximated them with conservative bounds, and the machinery THAT replaced walked
// the whole cache.
func Test_worthlessVictim(t *testing.T) {
	t.Parallel()

	requireNoVictim := func(t *testing.T, c *Cache[string, any], msg string) {
		t.Helper()

		c.mux.Lock()
		defer c.mux.Unlock()

		key, ok := c.vic.worthlessVictim(time.Now())
		require.Falsef(t, ok, "%s, but %v was named as worthless", msg, key)
	}

	requireVictim := func(t *testing.T, c *Cache[string, any], want, msg string) {
		t.Helper()

		c.mux.Lock()
		defer c.mux.Unlock()

		key, ok := c.vic.worthlessVictim(time.Now())
		require.Truef(t, ok, "%s", msg)
		require.Equalf(t, want, key, "%s", msg)
	}

	c := New(nopLookupFn, Config{Size: 4, TTL: time.Minute})

	requireNoVictim(t, c, "an empty cache has nothing to reclaim")

	seed(c, map[string]*entry[any]{
		"valid.example.com": {expireAt: time.Now().Add(time.Minute)},
	})
	requireNoVictim(t, c, "a cache of valid entries has nothing to reclaim")

	// An in-flight lookup is not an entry, so it cannot be named a victim either.
	seedFlight(c, "inflight.example.com")
	requireNoVictim(t, c, "an in-flight lookup holds no entry to reclaim")

	seed(c, map[string]*entry[any]{
		"valid.example.com":   {expireAt: time.Now().Add(time.Minute)},
		"residue.example.com": {err: errors.New("mock error")},
	})
	requireVictim(t, c, "residue.example.com", "error residue is always the first victim")

	seed(c, map[string]*entry[any]{
		"valid.example.com":   {expireAt: time.Now().Add(time.Minute)},
		"expired.example.com": {expireAt: time.Now().Add(-time.Second)},
	})
	requireVictim(t, c, "expired.example.com", "an expired entry with no stale window is worthless")

	// With stale-if-error enabled, an expired value is NOT worthless while a window
	// can still serve it: naming it would destroy a value a caller can still be given.
	expiredValue := map[string]*entry[any]{
		"stale.example.com": {expireAt: time.Now().Add(-time.Second)},
	}

	sc := New(nopLookupFn, Config{Size: 4, TTL: time.Minute, MaxStale: time.Hour})
	seed(sc, expiredValue)
	requireNoVictim(t, sc, "an open MaxStale window is not worthless")

	sf := New(nopLookupFn, Config{Size: 4, TTL: time.Minute, MaxStaleOnFailure: time.Hour})
	seed(sf, expiredValue)
	requireNoVictim(t, sf, "a value MaxStaleOnFailure can still revive is not worthless")

	// A revived stale value is not worthless while its anchored window is open, and
	// is worthless once it closes.
	seed(sf, map[string]*entry[any]{
		"revived.example.com": {val: "good", staleUntil: time.Now().Add(time.Hour)},
	})
	requireNoVictim(t, sf, "an open revived stale window is not worthless")

	seed(sf, map[string]*entry[any]{
		"revived.example.com": {val: "good", staleUntil: time.Now().Add(-time.Second)},
	})
	requireVictim(t, sf, "revived.example.com", "a closed revived stale window is worthless")

	// Past its MaxStale deadline, the value is worthless again.
	seed(sc, map[string]*entry[any]{
		"stale.example.com": {expireAt: time.Now().Add(-2 * time.Hour)},
	})
	requireVictim(t, sc, "stale.example.com", "a closed MaxStale window is worthless")
}

// Test_worthlessAt pins what makes an entry a reclaim victim. It is a property
// of the entry AND of the cache's stale settings, not of the entry alone: this
// is what stops a failing lookup from destroying a value that a stale window can
// still serve.
func Test_worthlessAt(t *testing.T) {
	t.Parallel()

	expireAt := time.Now().Add(-time.Second) // expired one second ago

	tests := []struct {
		name    string
		cfg     Config
		item    *entry[any]
		wantAt  time.Time
		wantOk  bool
		wantBad bool // worthless now?
	}{
		{
			name:    "error residue is worthless from the start",
			cfg:     Config{Size: 2, MaxStale: time.Hour, MaxStaleOnFailure: time.Hour},
			item:    &entry[any]{err: errors.New("mock error")},
			wantAt:  time.Time{},
			wantOk:  true,
			wantBad: true,
		},
		{
			name:    "no stale-if-error: worthless once expired",
			cfg:     Config{Size: 2},
			item:    &entry[any]{expireAt: expireAt},
			wantAt:  expireAt,
			wantOk:  true,
			wantBad: true,
		},
		{
			name:    "an open MaxStale window keeps an expired value",
			cfg:     Config{Size: 2, MaxStale: time.Hour},
			item:    &entry[any]{expireAt: expireAt},
			wantAt:  expireAt.Add(time.Hour),
			wantOk:  true,
			wantBad: false,
		},
		{
			name:    "a closed MaxStale window does not",
			cfg:     Config{Size: 2, MaxStale: time.Millisecond},
			item:    &entry[any]{expireAt: expireAt},
			wantAt:  expireAt.Add(time.Millisecond),
			wantOk:  true,
			wantBad: true,
		},
		{
			name:    "MaxStaleOnFailure: the next failure can still revive it, so it never turns worthless",
			cfg:     Config{Size: 2, MaxStaleOnFailure: time.Hour},
			item:    &entry[any]{expireAt: expireAt},
			wantAt:  time.Time{},
			wantOk:  false,
			wantBad: false,
		},
		{
			name:    "a revived value keeps its anchored deadline",
			cfg:     Config{Size: 2, MaxStaleOnFailure: time.Hour},
			item:    &entry[any]{val: "good", staleUntil: expireAt.Add(time.Hour)},
			wantAt:  expireAt.Add(time.Hour),
			wantOk:  true,
			wantBad: false,
		},
		{
			// The case ORDER inside worthlessAt is load-bearing, and this is the
			// only state that observes it: with BOTH windows set, a value whose
			// MaxStale window has already closed is still not worthless, because
			// the next failed refresh opens a MaxStaleOnFailure window on it.
			// Test MaxStaleOnFailure FIRST or this value is destroyed by a lookup
			// that produced nothing — a caller then gets an error where
			// stale-if-error promised the last known good value.
			name:    "both windows: a closed MaxStale window does not make a value worthless",
			cfg:     Config{Size: 2, MaxStale: time.Millisecond, MaxStaleOnFailure: time.Hour},
			item:    &entry[any]{val: "good", expireAt: expireAt},
			wantAt:  time.Time{},
			wantOk:  false,
			wantBad: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := New(nopLookupFn, tt.cfg)

			at, ok := c.vic.worthlessAt(tt.item)
			require.Equal(t, tt.wantOk, ok)

			if ok {
				require.True(t, tt.wantAt.Equal(at), "want %s, got %s", tt.wantAt, at)
			}

			require.Equal(t, tt.wantBad, c.vic.worthless(tt.item, time.Now()))
		})
	}
}

// Test_Lookup_inflight_does_no_capacity_work pins that installing an in-flight
// flight evicts nothing at all — not even a worthless entry. It holds no
// value, so it does not count against the capacity and has no room to make.
func Test_Lookup_inflight_does_no_capacity_work(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, key string) (any, error) {
		if strings.HasPrefix(key, "slow") {
			<-release
		}

		return nil, errors.New("mock error")
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	// Fill the capacity with worthless error residue.
	for _, key := range []string{"bad1.example.com", "bad2.example.com"} {
		_, err := c.Lookup(t.Context(), key)
		require.Error(t, err)
	}

	require.Equal(t, 2, c.Len())

	done := make(chan struct{})

	go func() {
		defer close(done)

		_, _ = c.Lookup(context.Background(), "slow.example.com")
	}()

	require.Eventually(t, func() bool {
		return c.Len() == 3
	}, time.Second, time.Millisecond, "the flight must overshoot the capacity, evicting nothing")

	c.mux.RLock()
	_, hasBad1 := c.keymap["bad1.example.com"]
	_, hasBad2 := c.keymap["bad2.example.com"]
	c.mux.RUnlock()

	require.True(t, hasBad1, "an in-flight lookup must not reclaim even a worthless entry")
	require.True(t, hasBad2, "an in-flight lookup must not reclaim even a worthless entry")

	requireConsistentAccounting(t, c)

	close(release)
	<-done
}

// Test_Lookup_stale_revive_reclaims_worthless_excess pins the other half of the
// revive's capacity contract: it may not take a value (see
// Test_Lookup_stale_revive_does_not_evict_a_valid_entry), but it must still
// reclaim worthless entries, or the cache stays permanently over capacity.
func Test_Lookup_stale_revive_reclaims_worthless_excess(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, key string) (any, error) {
		if strings.HasPrefix(key, "bad") {
			return nil, errors.New("mock error")
		}

		if calls.Add(1) == 1 {
			return "good", nil // prime the value that will be served stale
		}

		<-release // hold the failing refresh open

		return nil, errors.New("mock refresh failure")
	}

	c := New(lookupFn, Config{Size: 2, TTL: 5 * time.Millisecond, MaxStaleOnFailure: time.Minute})

	_, err := c.Lookup(t.Context(), "stale.example.com")
	require.NoError(t, err)

	_, err = c.Lookup(t.Context(), "bad1.example.com") // leaves worthless residue
	require.Error(t, err)

	time.Sleep(20 * time.Millisecond) // let the stale key expire

	done := make(chan struct{})

	go func() {
		defer close(done)

		v, err := c.Lookup(context.Background(), "stale.example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "good", v, "the failed refresh must serve the stale value")
	}()

	require.Eventually(t, func() bool {
		return inFlight(c, "stale.example.com")
	}, time.Second, time.Millisecond)

	// A second failing lookup, while the refresh is in flight, leaves more
	// worthless residue behind than the capacity allows.
	_, err = c.Lookup(t.Context(), "bad2.example.com")
	require.Error(t, err)

	close(release)
	<-done

	require.LessOrEqual(t, c.Len(), 2, "the stale revive must reclaim the worthless excess")
	requireConsistentAccounting(t, c)
}

// Test_Lookup_panicking_ttl_func_costs_no_value pins that a panicking ttlFn
// cannot cost the cache a value. entryTTL runs the caller-supplied function, so
// it must run BEFORE makeRoom: otherwise the eviction is paid for a value that
// the panic then prevents from ever being stored.
func Test_Lookup_panicking_ttl_func_costs_no_value(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, key string) (any, error) {
		return key, nil
	}

	ttlFn := func(key string, _ any) time.Duration {
		if key == "panic.example.com" {
			panic("mock ttlFn panic")
		}

		return time.Hour
	}

	c := New(lookupFn, Config{Size: 2}, WithTTLFunc(ttlFn))

	for _, key := range []string{"valid1.example.com", "valid2.example.com"} {
		_, err := c.Lookup(t.Context(), key)
		require.NoError(t, err)
	}

	require.Equal(t, 2, c.Len(), "the cache must be at capacity")

	func() {
		defer func() {
			require.NotNil(t, recover(), "the ttlFn panic must reach the caller")
		}()

		_, _ = c.Lookup(t.Context(), "panic.example.com")
	}()

	c.mux.RLock()
	_, hasValid1 := c.keymap["valid1.example.com"]
	_, hasValid2 := c.keymap["valid2.example.com"]
	c.mux.RUnlock()

	require.True(t, hasValid1, "a panicking ttlFn must not evict a value it never stores")
	require.True(t, hasValid2, "a panicking ttlFn must not evict a value it never stores")
	require.Equal(t, 2, c.Len())

	requireConsistentAccounting(t, c)
}

// Test_Lookup_stale_revive_respects_the_capacity pins that reviving a stale
// value pays for the value it stores.
//
// A revive is a failed refresh, so it may take no valid entry — but it DOES
// store a value, and a store that adds a value must make room for it. Reclaiming
// only worthless entries is not enough: stale-if-error is precisely the setting
// under which nothing is worthless, so the revive would find no victim it may take
// and store on top of a full map anyway. That is one unreclaimable value per
// concurrent failing refresh, for as long as the outage lasts — the cache would
// silently ignore Size exactly when the upstream is down.
func Test_Lookup_stale_revive_respects_the_capacity(t *testing.T) {
	t.Parallel()

	const (
		size = 4
		keys = 20
	)

	for _, cfg := range []Config{
		{Size: size, MaxStale: time.Hour},
		{Size: size, MaxStaleOnFailure: time.Hour},
		{Size: size, MaxStale: time.Hour, MaxStaleOnFailure: time.Hour},
	} {
		release := make(chan struct{})

		var (
			mu       sync.Mutex
			attempts = map[string]int{}
		)

		// The first lookup of a key succeeds; its refresh parks until released and
		// then fails. A TTL of zero stores the value already expired, so the very
		// next call refreshes it: no sleeping needed anywhere in this test.
		lookupFn := func(_ context.Context, key string) (string, error) {
			mu.Lock()
			attempts[key]++
			first := attempts[key] == 1
			mu.Unlock()

			if first {
				return "good-" + key, nil
			}

			<-release

			return "", errors.New("upstream down")
		}

		c := New(lookupFn, cfg)

		// Prime each key, then immediately park its failing refresh, BEFORE
		// priming the next one. A flight is not an entry, so the value it carries
		// (captured into its staleState) cannot be evicted: this is how every one
		// of the 20 keys ends up holding a last known good value to revive,
		// against a cache that can only store 4 of them.
		for i := range keys {
			key := fmt.Sprintf("key%d", i)

			_, err := c.Lookup(t.Context(), key)
			require.NoError(t, err)

			go func() {
				_, _ = c.Lookup(context.Background(), key)
			}()

			require.Eventually(t, func() bool { return inFlight(c, key) }, time.Second, time.Millisecond)
		}

		// Every refresh now fails, and every one of them revives a stale value.
		// Each revive STORES a value, so each must make room for it: reclaiming
		// only worthless entries would find none — stale-if-error is exactly the
		// setting under which nothing is worthless — and would store on top of a
		// full map, once per key, unreclaimable until the outage ends.
		close(release)

		require.Eventually(t, func() bool { return c.Len() > 0 && !anyInFlight(c) }, 5*time.Second, time.Millisecond)

		require.LessOrEqualf(t, values(c), size,
			"the values held (%d) must never exceed Size (%d) with cfg %+v:"+
				" a stale revive stores a value and must pay for it",
			values(c), size, cfg)

		requireConsistentAccounting(t, c)
	}
}

// Test_set_anchors_the_ttl_after_making_room pins that a value's lifetime starts
// when it is stored, not when the store began.
//
// Making room runs under the exclusive write lock, and while a single eviction is
// now cheap, a store may still have to make MANY of them. Anchoring expireAt before
// that work charges its cost to the value's own TTL: the value is born short by the
// cost of storing it, and a value whose TTL is shorter than the work is born ALREADY
// EXPIRED — never cacheable, so every lookup of that key misses and pays it again.
//
// The cache is deliberately left far over its capacity, so that the store must evict
// its way back down and the cost is large enough to observe. That is the same shape
// as a store on a cache whose capacity was just reduced, and it is the only way left
// to make the work take long enough to time: an eviction no longer walks the cache.
func Test_set_anchors_the_ttl_after_making_room(t *testing.T) {
	t.Parallel()

	const (
		overrun = 100_000
		ttl     = time.Minute
	)

	c := New(nopLookupFn, Config{Size: 1, TTL: ttl})

	c.mux.Lock()

	for i := range overrun {
		c.store(fmt.Sprintf("fill%d", i), &entry[any]{expireAt: time.Now().Add(time.Hour)})
	}

	before := time.Now()

	// This store must evict its way from 100,000 entries down to one.
	c.set("target", "value", nil, c.ttl)

	work := time.Since(before)
	item := c.keymap["target"]

	c.mux.Unlock()

	require.NotNil(t, item)
	require.Equal(t, 1, c.Len(), "the store must have evicted its way back to the capacity")

	// Anchored after the work: expireAt is about (before + work + ttl).
	// Anchored before it (the bug): expireAt is exactly (before + ttl).
	// Require at least half the observed cost to have been kept out of the TTL,
	// which no timing jitter can account for.
	require.Truef(t, item.expireAt.After(before.Add(ttl+work/2)),
		"making room (%v) was charged to the new value's TTL: it has only %v of its %v left",
		work, time.Until(item.expireAt), ttl)
}

// Test_store_replacing_an_entry_keeps_the_accounting pins that store accounts for
// the entry it replaces. Overwriting error residue with a value must take it out of
// the residue index too: a leaked index entry names a key the map no longer holds,
// so every eviction would offer that ghost as its victim, drop nothing, and give up
// — which is the state makeRoom's own termination guard exists to survive.
func Test_store_replacing_an_entry_keeps_the_accounting(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 4, TTL: time.Minute})

	c.mux.Lock()

	c.store("key", &entry[any]{err: errors.New("mock error")})
	require.Equal(t, 1, c.vic.residue.len())

	c.store("key", &entry[any]{val: "value", expireAt: time.Now().Add(time.Minute)})

	c.mux.Unlock()

	require.Zero(t, c.vic.residue.len(), "replacing error residue with a value must take it out of the residue queue")
	requireConsistentAccounting(t, c)
}

// Test_makeRoom_terminates_when_the_accounting_drifts pins makeRoom's
// termination guard, and reclaim's repair of a residue index that disagrees with
// the map.
//
// The residue index names victims without consulting the map, so if it ever drifts
// it claims a victim exists where none does. One thing then stands between the cache
// and an infinite loop holding the exclusive write lock: evict must report what drop
// ACTUALLY did, not what it meant to do, so that makeRoom stops instead of asking
// again forever. Without it the cache wedges — which is worse than a wrong result.
func Test_makeRoom_terminates_when_the_accounting_drifts(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 2, TTL: time.Minute})

	seed(c, map[string]*entry[any]{
		"a.example.com": {val: "a", expireAt: time.Now().Add(time.Hour)},
		"b.example.com": {val: "b", expireAt: time.Now().Add(time.Hour)},
	})

	// Drift the accounting: the residue queue names an entry that is not stored, so
	// eviction is handed a victim it cannot drop, while the map holds nothing but
	// valid entries that evictWorthless may not take.
	c.mux.Lock()
	c.vic.residue.push("ghost.example.com", &entry[any]{err: errors.New("ghost")})
	c.mux.Unlock()

	done := make(chan struct{})

	go func() {
		defer close(done)

		c.mux.Lock()
		defer c.mux.Unlock()

		c.makeRoom("new.example.com", evictWorthless)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "makeRoom spun forever holding the write lock: its termination guard is gone")
	}
}

// Test_Lookup_failing_key_at_capacity_costs_no_value pins that a failing lookup against
// a full cache of valid entries takes nothing: it may only reclaim what is worthless,
// and nothing here is.
func Test_Lookup_failing_key_at_capacity_costs_no_value(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, key string) (string, error) {
		if strings.HasPrefix(key, "bad") {
			return "", errors.New("upstream down")
		}

		return "good-" + key, nil
	}

	// Every one of these fails and finds the cache full of valid entries it may not take.
	// The residue of each is removed before the next, so that nothing IS worthless and
	// the failing lookup has no victim available to it at all.
	failAtCapacity := func(size int) {
		c := New(lookupFn, Config{Size: size, TTL: time.Hour})

		for i := range size {
			_, err := c.Lookup(t.Context(), fmt.Sprintf("key%d", i))
			require.NoError(t, err)
		}

		for i := range 20 {
			key := fmt.Sprintf("bad%d", i)

			_, err := c.Lookup(t.Context(), key)
			require.Error(t, err)

			c.Remove(key)
		}

		require.Equalf(t, size, values(c),
			"a failing lookup destroyed a valid value it was never allowed to take (Size=%d)", size)
		requireConsistentAccounting(t, c)
	}

	failAtCapacity(16)
	failAtCapacity(1024)
}

// failingRefreshFn serves every key but "refreshed", which it holds in flight until
// released and then fails, so that a refresh can be made to land on a chosen state.
func failingRefreshFn(down *atomic.Bool, release <-chan struct{}) LookupFunc[string, string] {
	return func(_ context.Context, key string) (string, error) {
		if down.Load() && (key == "refreshed") {
			<-release // hold the failing refresh in flight

			return "", errors.New("upstream down")
		}

		return "good-" + key, nil
	}
}

// hotKeyTTL keeps the "hot" keys valid for an hour, and gives every other key a zero
// TTL so that it is stored already expired and every call refreshes it.
func hotKeyTTL(key string, _ string) time.Duration {
	if strings.HasPrefix(key, "hot") {
		return time.Hour
	}

	return 0
}

// Test_Lookup_failed_refresh_examines_a_constant_number_of_entries is the same
// property on the STALE REVIVE path, which the test above does not reach because it
// configures no stale window. The cost of a scan here would land during an outage,
// exactly when stale-if-error is the thing keeping the service up.
func Test_Lookup_failed_refresh_at_capacity_serves_stale_within_the_bound(t *testing.T) {
	t.Parallel()

	refreshAtCapacity := func(size int) {
		release := make(chan struct{})

		var down atomic.Bool

		c := New(failingRefreshFn(&down, release), Config{Size: size, MaxStaleOnFailure: time.Hour},
			WithTTLFunc(hotKeyTTL))

		for i := range size - 1 {
			_, err := c.Lookup(t.Context(), fmt.Sprintf("hot%d", i))
			require.NoError(t, err)
		}

		_, err := c.Lookup(t.Context(), "refreshed")
		require.NoError(t, err)

		down.Store(true)

		go func() {
			// The refresh supersedes the entry it is refreshing, vacating its slot.
			_, _ = c.Lookup(context.Background(), "refreshed")
		}()

		require.Eventually(t, func() bool { return inFlight(c, "refreshed") }, time.Second, time.Millisecond)

		// Refill the vacated slot while the refresh is still in the air, so that the
		// revive lands on a cache that is FULL of valid entries — the state in which
		// it has no victim it may take.
		_, err = c.Lookup(t.Context(), fmt.Sprintf("hot%d", size-1))
		require.NoError(t, err)

		close(release)

		require.Eventually(t, func() bool { return !inFlight(c, "refreshed") }, time.Second, time.Millisecond)

		for range 20 {
			v, err := c.Lookup(t.Context(), "refreshed")
			require.NoError(t, err, "the stale value must be served")
			require.Equal(t, "good-refreshed", v)
		}

		require.LessOrEqual(t, values(c), size+1, "a revive may exceed the capacity by one value, and only one")
		requireConsistentAccounting(t, c)
	}

	refreshAtCapacity(16)
	refreshAtCapacity(1024)
}

// Test_Lookup_failing_lookup_stream_costs_no_value pins the steady state of an outage: a
// stream of distinct failing lookups reclaims the residue the previous one left, and
// never a value. Residue is worthless under every configuration, so it is always the
// victim available to a failing lookup — which is the only kind of victim one may take.
func Test_Lookup_failing_lookup_stream_costs_no_value(t *testing.T) {
	t.Parallel()

	stream := func(cfg Config) {
		c := New(func(_ context.Context, key string) (string, error) {
			if strings.HasPrefix(key, "bad") {
				return "", errors.New("upstream down")
			}

			return "good-" + key, nil
		}, cfg)

		for i := range cfg.Size {
			_, err := c.Lookup(t.Context(), fmt.Sprintf("hot%d", i))
			require.NoError(t, err)
		}

		// A stream of failing lookups, each leaving residue for the next to reclaim.
		// No Remove in between: this is the steady state of a real outage.
		for i := range 50 {
			_, err := c.Lookup(t.Context(), fmt.Sprintf("bad%d", i))
			require.Error(t, err)
		}

		require.LessOrEqualf(t, values(c), cfg.Size,
			"a failing lookup cost the cache a value (Size=%d)", cfg.Size)
		requireConsistentAccounting(t, c)
	}

	// Every stale configuration: what counts as worthless differs under each, and the
	// failing lookup must still find its residue and take nothing else.
	for _, cfg := range []Config{
		{Size: 16, TTL: time.Hour},
		{Size: 1024, TTL: time.Hour},
		{Size: 16, TTL: time.Hour, MaxStale: time.Hour},
		{Size: 1024, TTL: time.Hour, MaxStale: time.Hour},
		{Size: 16, TTL: time.Hour, MaxStaleOnFailure: time.Hour},
		{Size: 1024, TTL: time.Hour, MaxStaleOnFailure: time.Hour},
	} {
		stream(cfg)
	}
}

// Test_makeRoom_stale_revive_takes_nothing_when_it_may_take_nothing is the same
// guard pinned directly: with nothing expired, a revive has no victim it is allowed
// to take, so it must take none and leave the cache over capacity instead.
func Test_makeRoom_stale_revive_takes_nothing_when_it_may_take_nothing(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 2, MaxStaleOnFailure: time.Hour})

	seed(c, map[string]*entry[any]{
		"a.example.com": {val: "a", expireAt: time.Now().Add(time.Hour)},
		"b.example.com": {val: "b", expireAt: time.Now().Add(time.Hour)},
	})

	c.mux.Lock()

	// A new key at capacity: the eviction loop DOES run (2 entries, room for 1),
	// but every entry is valid and a revive may take none of them.
	c.makeRoom("revived.example.com", evictStale)

	c.mux.Unlock()

	require.Len(t, c.keymap, 2, "a revive must take nothing rather than a valid entry")

	requireConsistentAccounting(t, c)
}

// Test_evict_examines_an_entry_when_it_has_a_victim_to_find is the positive control
// for the probe counter. Without it, a counter that is never incremented at all
// satisfies every assertion that two of its readings are EQUAL, and the property above
// could be deleted with the suite still green.
func Test_makeRoom_evicts_when_it_has_a_victim_it_may_take(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 2, TTL: time.Minute})

	seed(c, map[string]*entry[any]{
		"a.example.com": {val: "a", expireAt: time.Now().Add(time.Hour)},
		"b.example.com": {val: "b", expireAt: time.Now().Add(time.Hour)},
	})

	c.mux.Lock()

	// Every entry is valid, but a cacheable value may take the one closest to expiring:
	// this store HAS a victim it is allowed to take, so it must actually take it.
	c.makeRoom("c.example.com", evictValue)

	c.mux.Unlock()

	require.Equal(t, 1, c.Len(), "a store that has a victim it may take must evict one")
	requireConsistentAccounting(t, c)
}

// Test_evict_takes_the_stale_value_closest_to_its_deadline pins the victim choice
// among the values that are only being served stale. Taking whichever one the map
// iterator happened to hit would destroy a value with its whole window ahead of it
// and keep one about to lapse.
//
// Among the DATED ones — the values a failed refresh revived — the one closest to
// its deadline goes first, because it is nearly worthless anyway.
func Test_evict_takes_the_stale_value_closest_to_its_deadline(t *testing.T) {
	t.Parallel()

	// Repeated because the map iteration order is random: a victim chosen by
	// preference wins every time, one chosen by accident would not.
	for range 64 {
		c := New(nopLookupFn, Config{Size: 3, MaxStale: time.Hour})

		seed(c, map[string]*entry[any]{
			"doomed.example.com":  {val: "doomed", staleUntil: time.Now().Add(10 * time.Millisecond)},
			"healthy.example.com": {val: "healthy", staleUntil: time.Now().Add(time.Hour)},
			"valid.example.com":   {val: "valid", expireAt: time.Now().Add(time.Hour)},
		})

		c.mux.Lock()
		require.True(t, evictOne(c, evictStale))
		c.mux.Unlock()

		require.NotContains(t, c.keymap, "doomed.example.com",
			"among the revived values, the one closest to its deadline must go first")
		require.Contains(t, c.keymap, "healthy.example.com")
		require.Contains(t, c.keymap, "valid.example.com")

		requireConsistentAccounting(t, c)
	}
}

// Test_evict_sacrifices_a_stale_value_nobody_asked_for pins the victim choice that
// the deadline order alone gets exactly backwards.
//
// An entry becomes DATED by being revived, and it is revived because a caller asked
// for it during the outage. So under MaxStaleOnFailure a dated entry is one the
// cache is actively serving, while an UNDATED one (a plain expired value no failure
// has revived) has not been asked for since it expired. Ranking by remaining window
// would keep the cold value — which never becomes worthless on its own, so it looks
// the most valuable — and destroy the working set, deterministically. The value
// nobody has asked for goes first.
func Test_evict_sacrifices_a_stale_value_nobody_asked_for(t *testing.T) {
	t.Parallel()

	for range 64 {
		c := New(nopLookupFn, Config{Size: 3, MaxStaleOnFailure: time.Hour})

		cold := &entry[any]{val: "cold", expireAt: time.Now().Add(-time.Second)}

		_, dated := c.vic.worthlessAt(cold)
		require.False(t, dated, "the cold fixture must really be undated, or this test proves nothing")

		hot := &entry[any]{val: "hot", staleUntil: time.Now().Add(time.Hour)}

		_, dated = c.vic.worthlessAt(hot)
		require.True(t, dated, "a revived value must be dated")

		seed(c, map[string]*entry[any]{
			"cold.example.com": cold,
			"hot.example.com":  hot,
		})

		c.mux.Lock()
		require.True(t, evictOne(c, evictStale))
		c.mux.Unlock()

		require.NotContains(t, c.keymap, "cold.example.com",
			"the value nobody has asked for during the outage must go first")
		require.Contains(t, c.keymap, "hot.example.com",
			"a revived value is the one a caller is actively being served")

		requireConsistentAccounting(t, c)
	}
}

// Test_evict_takes_an_undated_stale_value_when_that_is_all_there_is pins that an
// undated stale value is takeable at all. If it were not, a revive against a cache
// holding only undated stale values would find no victim and overrun the capacity
// every single time.
func Test_evict_takes_an_undated_stale_value_when_that_is_all_there_is(t *testing.T) {
	t.Parallel()

	for range 64 {
		c := New(nopLookupFn, Config{Size: 3, MaxStaleOnFailure: time.Hour})

		seed(c, map[string]*entry[any]{
			"a.example.com": {val: "a", expireAt: time.Now().Add(-time.Second)},
			"b.example.com": {val: "b", expireAt: time.Now().Add(-time.Second)},
			"c.example.com": {val: "c", expireAt: time.Now().Add(-time.Second)},
		})

		c.mux.Lock()

		for _, item := range c.keymap {
			_, dated := c.vic.worthlessAt(item)
			require.False(t, dated, "every fixture entry must be undated")
		}

		took := evictOne(c, evictStale)

		c.mux.Unlock()

		require.True(t, took, "an undated stale value must be takeable when it is all there is")
		require.Len(t, c.keymap, 2)

		requireConsistentAccounting(t, c)
	}
}

// Test_Lookup_eviction_cost_does_not_grow_with_the_capacity pins the cost of an eviction
// on the FAILING path: it must not grow with [Config.Size].
//
// This is a wall-clock guard because nothing else can be one. Coverage, -race, lint and
// an in-code counter are all blind to an O(Size) pass under the exclusive write lock: a
// counter only counts the work the code routes through it.
//
// The state it builds is the one an outage leaves behind: a residue index that HAS been
// large.
func Test_Lookup_eviction_cost_does_not_grow_with_the_capacity(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, key string) (string, error) {
		if strings.HasPrefix(key, "bad") {
			return "", errors.New("upstream down")
		}

		return "good-" + key, nil
	}

	// The cost of one failing lookup at capacity, against a cache whose residue index
	// was grown to Size by an outage and then drained by the recovery.
	afterAnOutage := func(size int) time.Duration {
		c := New(lookupFn, Config{Size: size, TTL: time.Hour})

		// The outage: distinct cold keys, all failing, on a cache below capacity — so
		// nothing is reclaimed and the residue index grows to the capacity.
		for i := range size {
			_, _ = c.Lookup(t.Context(), fmt.Sprintf("bad%d", i))
		}

		// The recovery: good values evict the residue back down, but the index keeps
		// whatever room it took.
		for i := range size {
			_, _ = c.Lookup(t.Context(), fmt.Sprintf("good%d", i))
		}

		const iter = 300

		start := time.Now()

		// The timed keys must FAIL. A successful store leaves no residue, so timing one
		// would leave the residue index empty for the whole measured region — and an
		// empty index is cheap to search whatever room it once took. What has to be timed
		// is the operation the index is on the hot path of: a failing lookup at capacity,
		// reclaiming the residue the last one left.
		for i := range iter {
			_, _ = c.Lookup(t.Context(), fmt.Sprintf("bad-post%d", i))
		}

		elapsed := time.Since(start) // BEFORE the invariant check, which walks the map

		requireConsistentAccounting(t, c)

		return elapsed / iter
	}

	const factor = 64 // the capacities differ by this much

	small, large := afterAnOutage(1_000), afterAnOutage(factor*1_000)

	// Generous: a linear eviction is ~64x here, anything flat is ~1x. Ten leaves room
	// for hash scatter and a loaded CI box, but not for a linear pass.
	require.Lessf(t, large, 10*small,
		"a failing lookup cost %v at Size=1000 but %v at Size=%d: the eviction is walking"+
			" something proportional to the capacity, under the exclusive write lock",
		small, large, factor*1_000)
}

// Test_Lookup_revive_cost_does_not_grow_with_the_capacity pins the cost of an eviction
// on the REVIVE path: it must not grow with [Config.Size].
//
// A stale revive has its own eviction level (evictStale) and its own queue, and neither
// is reached unless a stale window is configured, so this needs a cache configured for
// one. The state it builds is an outage in progress: the cache full, the upstream down,
// every known key served stale and every cold one failing.
func Test_Lookup_revive_cost_does_not_grow_with_the_capacity(t *testing.T) {
	t.Parallel()

	var down atomic.Bool

	lookupFn := func(_ context.Context, key string) (string, error) {
		if down.Load() {
			return "", errors.New("upstream down")
		}

		return "good-" + key, nil
	}

	midOutage := func(size int) time.Duration {
		down.Store(false)

		// A TTL that runs out as the value is stored: every later call attempts a
		// refresh, which is what serving a value stale requires.
		c := New(lookupFn, Config{Size: size, TTL: time.Nanosecond, MaxStaleOnFailure: time.Hour})

		for i := range size {
			_, _ = c.Lookup(t.Context(), fmt.Sprintf("k%d", i))
		}

		down.Store(true)

		// The outage: every key fails its refresh and is revived, so the stale queue
		// grows to the capacity.
		for i := range size {
			_, _ = c.Lookup(t.Context(), fmt.Sprintf("k%d", i))
		}

		const iter = 300

		start := time.Now()

		for i := range iter {
			// A cold key that fails, leaving residue and putting the cache over
			// capacity...
			_, _ = c.Lookup(t.Context(), fmt.Sprintf("cold%d", i))

			// ...so that reviving a known one has to make room, and its eviction picks
			// a victim for real.
			_, _ = c.Lookup(t.Context(), fmt.Sprintf("k%d", i%size))
		}

		elapsed := time.Since(start) // BEFORE the invariant check, which walks the map

		requireConsistentAccounting(t, c)

		return elapsed / iter
	}

	const factor = 64 // the capacities differ by this much

	small, large := midOutage(1_000), midOutage(factor*1_000)

	require.Lessf(t, large, 10*small,
		"a stale revive cost %v at Size=1000 but %v at Size=%d: the eviction is walking"+
			" something proportional to the capacity, under the exclusive write lock",
		small, large, factor*1_000)
}

// Test_file_and_unfile_agree_on_every_entry_state pins the equivalence of the two
// switches that decide which queue an entry belongs to.
//
// [victims.file] chooses the queue by the entry's fields, and [victims.unfile] must
// re-derive the SAME queue from the SAME fields. They are not adjacent in the source,
// and nothing else asserts that they agree: for every state the cache actually
// constructs, testing the fields in a different ORDER is an equivalent mutant, so a
// desynchronised pair leaves an entry filed in one queue and taken out of another —
// corrupting both — while the rest of the suite stays green.
//
// The states below therefore include one the cache never builds today (an entry holding
// an error AND carrying a revived deadline). Nothing forbids one, and it is the only
// state that tells the two orders apart.
func Test_file_and_unfile_agree_on_every_entry_state(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]*entry[string]{
		"a value":                  {val: "v", expireAt: now.Add(time.Hour)},
		"error residue":            {err: errors.New("mock error")},
		"a revived stale value":    {val: "v", staleUntil: now.Add(time.Hour)},
		"an error AND revived":     {err: errors.New("mock error"), val: "v", staleUntil: now.Add(time.Hour)},
		"an expired value":         {val: "v", expireAt: now.Add(-time.Hour)},
		"a revived, shut deadline": {val: "v", staleUntil: now.Add(-time.Hour)},
	}

	for name, item := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var vic victims[string, string]

			vic.file("key", item)

			held := vic.values.len() + vic.stale.len() + vic.residue.len()
			require.Equal(t, 1, held, "file must put the entry in exactly one queue")

			vic.unfile(item)

			left := vic.values.len() + vic.stale.len() + vic.residue.len()
			require.Zero(t, left,
				"unfile took the entry out of a different queue than file put it in: the two"+
					" switches disagree, so an entry is left behind in one queue and removed from another")
		})
	}
}
