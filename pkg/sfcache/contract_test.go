// The CONTRACT of this package: everything it promises, expressed through the PUBLIC
// API only.
//
// Nothing in here reaches inside the cache. Every other test file pins the current
// implementation — the queues, the entry, the eviction levels — and so has to be
// rewritten when the implementation is; this one does not, and must stay green against
// any implementation that keeps the promises below.
//
// The properties:
//
//   - single flight: exactly one upstream call per cold key under a stampede
//   - the working set survives a cold burst: lookups in flight cost no values
//   - a merely-attempted lookup never costs a live value, and a failing one destroys
//     neither another key's stale value nor a valid entry
//   - failing keys stay bounded, and a failing lookup still sheds dead values
//   - the victim is the value closest to expiring, and among stale values it is the
//     one no caller has asked for
//   - a stale revive exceeds the capacity by exactly one value, and self-heals
//   - MaxStaleOnFailure protects a cold key; MaxStale does not
//   - a revived value keeps refreshing, and the first success replaces it
//   - a NaN key is rejected, not leaked
//   - PurgeExpired removes the expired and keeps the valid
package sfcache_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/sfcache"
)

var errDown = errors.New("upstream down")

// counter counts the upstream calls per key: the only way to observe, from the
// outside, whether a value was still cached.
type counter struct {
	mux   sync.Mutex
	calls map[string]int
}

func (c *counter) get(key string) int {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.calls[key]
}

func (c *counter) hit(key string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.calls == nil {
		c.calls = map[string]int{}
	}

	c.calls[key]++
}

// the in-flight lookups must not count against the capacity, or a burst
// of cold keys empties the cache of the values it is holding for everyone else.
func Test_safety_working_set_survives_a_cold_burst(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const (
		size    = 64
		inFlood = 128
	)

	var cnt counter

	park := make(chan struct{})

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		cnt.hit(key)

		if key[0] == 'c' { // cold keys park in flight
			<-park
		}

		return "v" + key, nil
	}, sfcache.Config{Size: size, TTL: time.Hour})

	var wg sync.WaitGroup

	for i := range inFlood {
		wg.Go(func() {
			_, _ = c.Lookup(ctx, "cold"+strconv.Itoa(i))
		})
	}

	require.Eventually(t, func() bool { return c.Len() >= inFlood }, 5*time.Second, time.Millisecond)

	// Store a full working set while the cold keys are still in flight.
	for i := range size {
		_, err := c.Lookup(ctx, "hot"+strconv.Itoa(i))
		require.NoError(t, err)
	}

	// Every hot key must still be a hit: no cold key in flight may have cost one.
	for i := range size {
		_, err := c.Lookup(ctx, "hot"+strconv.Itoa(i))
		require.NoError(t, err)
		require.Equalf(t, 1, cnt.get("hot"+strconv.Itoa(i)),
			"hot%d was evicted by a lookup that was merely in flight", i)
	}

	close(park)
	wg.Wait()
}

// a lookup that is merely attempted, and then FAILS, must not cost the
// cache a healthy value.
func Test_safety_failed_lookup_does_not_evict_a_valid_entry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var cnt counter

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		cnt.hit(key)

		if key == "bad" {
			return "", errDown
		}

		return "v" + key, nil
	}, sfcache.Config{Size: 2, TTL: time.Hour})

	_, _ = c.Lookup(ctx, "good1")
	_, _ = c.Lookup(ctx, "good2")

	_, err := c.Lookup(ctx, "bad")
	require.Error(t, err)

	_, _ = c.Lookup(ctx, "good1")
	_, _ = c.Lookup(ctx, "good2")

	require.Equal(t, 1, cnt.get("good1"), "a failing lookup evicted good1")
	require.Equal(t, 1, cnt.get("good2"), "a failing lookup evicted good2")
}

// a stream of distinct failing keys must stay bounded.
func Test_safety_failing_keys_stay_bounded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	c := sfcache.New(func(_ context.Context, _ string) (string, error) {
		return "", errDown
	}, sfcache.Config{Size: 2, TTL: time.Hour})

	for i := range 20 {
		_, _ = c.Lookup(ctx, "bad"+strconv.Itoa(i))
		require.LessOrEqualf(t, c.Len(), 3, "the residue of failing lookups grew past Size+1 at %d", i)
	}
}

// an always-failing key must not destroy another key's stale value.
func Test_safety_failing_key_does_not_destroy_a_stale_value(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	down := false

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		if key == "C" {
			return "", errDown
		}

		if down {
			return "", errDown
		}

		return "v" + key, nil
	}, sfcache.Config{Size: 2, TTL: time.Millisecond, MaxStaleOnFailure: time.Hour})

	_, err := c.Lookup(ctx, "A")
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)

	down = true

	got, err := c.Lookup(ctx, "A") // failed refresh -> revived stale
	require.NoError(t, err)
	require.Equal(t, "vA", got)

	down = false

	_, err = c.Lookup(ctx, "B") // fills the cache to capacity
	require.NoError(t, err)

	_, err = c.Lookup(ctx, "C") // never produces a value
	require.Error(t, err)

	down = true

	got, err = c.Lookup(ctx, "A")
	require.NoError(t, err, "a key that never produced a value destroyed A's stale protection")
	require.Equal(t, "vA", got)
}

// the value being actively served stale must survive in preference to the one nobody
// has asked for, under EVERY stale configuration.
//
// A value is revived because a caller asked for it during the outage, so ranking stale
// values by their remaining window ranks them by how little anyone wants them. The
// windows are covered separately and together, because each computes the deadline
// differently.
func Test_safety_stale_victim_is_the_one_nobody_asked_for(t *testing.T) {
	t.Parallel()

	for name, cfg := range map[string]sfcache.Config{
		"MaxStale":          {Size: 2, TTL: time.Millisecond, MaxStale: time.Hour},
		"MaxStaleOnFailure": {Size: 2, TTL: time.Millisecond, MaxStaleOnFailure: time.Hour},
		"both":              {Size: 2, TTL: time.Millisecond, MaxStale: time.Hour, MaxStaleOnFailure: time.Hour},
	} {
		ctx := context.Background()
		down := false

		c := sfcache.New(func(_ context.Context, key string) (string, error) {
			if down && (key != "new") {
				return "", errDown
			}

			return "v" + key, nil
		}, cfg)

		_, _ = c.Lookup(ctx, "asked")   // will be revived: someone asks for it
		_, _ = c.Lookup(ctx, "ignored") // will just sit there, expired

		time.Sleep(5 * time.Millisecond)

		down = true

		got, err := c.Lookup(ctx, "asked") // revived: evidence of demand
		require.NoErrorf(t, err, "%s", name)
		require.Equalf(t, "vasked", got, "%s", name)

		// A new value arrives and the cache is at capacity: something must go.
		_, err = c.Lookup(ctx, "new")
		require.NoErrorf(t, err, "%s", name)

		got, err = c.Lookup(ctx, "asked")
		require.NoErrorf(t, err, "%s: the value being actively served was sacrificed for the one nobody asked for", name)
		require.Equalf(t, "vasked", got, "%s", name)
	}
}

// at capacity, a successful store takes the valid value closest to expiring.
// Test_safety_a_failing_upstream_cannot_make_a_value_immortal pins the promise of
// [sfcache.Config.MaxStaleOnFailure]: the window is anchored ONCE, by the first failed
// refresh, so later failures keep serving the same value until that deadline but never
// push it back.
//
// It runs under every window combination that can anchor on a failure. The deadline is a
// different computation under each — with MaxStale also set it is the LATER of the two —
// so exercising one window cannot pin the other.
// hammerUntilTheWindowShuts keeps looking the key up against a dead upstream, and
// reports whether the stale window ever shut within the budget. The error that shuts it
// must be the upstream's own.
func hammerUntilTheWindowShuts(t *testing.T, c *sfcache.Cache[string, string], budget, every time.Duration) bool {
	t.Helper()

	until := time.Now().Add(budget)

	for time.Now().Before(until) {
		_, err := c.Lookup(t.Context(), "k")
		if err != nil {
			require.ErrorIs(t, err, errDown, "the window shut, but with the wrong error")

			return true // the window shut: the value is mortal, as promised
		}

		time.Sleep(every)
	}

	return false
}

func Test_safety_a_failing_upstream_cannot_make_a_value_immortal(t *testing.T) {
	t.Parallel()

	const (
		ttl    = 20 * time.Millisecond
		window = 60 * time.Millisecond
	)

	configs := map[string]sfcache.Config{
		"MaxStaleOnFailure": {Size: 4, TTL: ttl, MaxStaleOnFailure: window},
		"both":              {Size: 4, TTL: ttl, MaxStale: window, MaxStaleOnFailure: window},
	}

	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var down atomic.Bool

			c := sfcache.New(func(_ context.Context, _ string) (string, error) {
				if down.Load() {
					return "", errDown
				}

				return "v1", nil
			}, cfg)

			val, err := c.Lookup(t.Context(), "k")
			require.NoError(t, err)
			require.Equal(t, "v1", val)

			time.Sleep(ttl + (5 * time.Millisecond)) // the value expires

			down.Store(true)

			// Hammer the dead upstream well past the window. Every call re-attempts the
			// refresh, and every one of them fails: if a failure could re-anchor the
			// window, the value would go on being served for as long as the upstream
			// stays down — forever, which is precisely what the promise forbids.
			require.Truef(t, hammerUntilTheWindowShuts(t, c, 10*window, window/6),
				"still served stale after %v of continuous failure against a %v window: a failed"+
					" refresh is pushing the deadline back", 10*window, window)
		})
	}
}

// Test_safety_evicts_the_value_closest_to_expiring pins the victim of an ordinary store
// at capacity.
//
// The INSERTION ORDER is load-bearing. Values are stored in DESCENDING order of
// expiration, so every insert belongs at the head and each of the three evictions reads
// its victim off a structure the previous one re-shaped. Storing them in ascending
// deadline order would need no ordering structure at all, being sorted by construction,
// and would still pass with the heap's sifting removed.
func Test_safety_evicts_the_value_closest_to_expiring(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const size = 8

	var cnt counter

	// "k<n>" expires in n+1 minutes; anything else outlives the lot, so the values
	// stored AT capacity are never themselves the victim.
	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		cnt.hit(key)

		return "v" + key, nil
	}, sfcache.Config{Size: size, TTL: time.Hour},
		sfcache.WithTTLFunc(func(key string, _ string) time.Duration {
			n, err := strconv.Atoi(strings.TrimPrefix(key, "k"))
			if err != nil {
				return time.Hour // the values stored AT capacity outlive every k<n>
			}

			return time.Duration(n+1) * time.Minute
		}))

	for i := size - 1; i >= 0; i-- { // descending: each new value belongs at the head
		_, _ = c.Lookup(ctx, "k"+strconv.Itoa(i))
	}

	const evictions = 3

	for i := range evictions { // at capacity: k0, k1, k2 must go, closest to expiring first
		_, _ = c.Lookup(ctx, "new"+strconv.Itoa(i))
	}

	// The survivors first: they are still cached, so they cost no second upstream call.
	// (Checking them BEFORE the evicted ones matters — a miss below stores at capacity
	// and would evict one of these in turn.)
	for i := evictions; i < size; i++ {
		key := "k" + strconv.Itoa(i)

		_, _ = c.Lookup(ctx, key)

		require.Equalf(t, 1, cnt.get(key),
			"%s was evicted, but %d values were closer to expiring than it", key, evictions)
	}

	// And the three closest to expiring are gone: each costs a second upstream call.
	for i := range evictions {
		key := "k" + strconv.Itoa(i)

		_, _ = c.Lookup(ctx, key)

		require.Equalf(t, 2, cnt.get(key),
			"%s survived, but it was among the %d values closest to expiring", key, evictions)
	}
}

// a stale revive that can take nothing exceeds the capacity by exactly
// one value, and the next successful store reclaims it.
func Test_safety_revive_overruns_by_one_and_self_heals(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	down := false

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		if down && (key != "fresh") {
			return "", errDown
		}

		return "v" + key, nil
	}, sfcache.Config{Size: 2, TTL: time.Millisecond, MaxStaleOnFailure: time.Hour})

	_, _ = c.Lookup(ctx, "A")
	_, _ = c.Lookup(ctx, "B")

	time.Sleep(5 * time.Millisecond)

	down = true

	for range 5 {
		_, _ = c.Lookup(ctx, "A")
		_, _ = c.Lookup(ctx, "B")
		require.LessOrEqual(t, c.Len(), 3, "the revives grew the cache past Size+1")
	}

	down = false

	_, _ = c.Lookup(ctx, "fresh")
	require.LessOrEqual(t, c.Len(), 2, "the successful store did not reclaim the overrun")
}

// a key that is not equal to itself is rejected, not leaked.
func Test_safety_nan_key_is_rejected(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	type key struct {
		f float64
	}

	calls := atomic.Int64{}

	c := sfcache.New(func(_ context.Context, _ key) (string, error) {
		calls.Add(1)

		return "v", nil
	}, sfcache.Config{Size: 8, TTL: time.Hour})

	nan := key{f: func() float64 { var z float64; return z / z }()}

	for range 100 {
		_, err := c.Lookup(ctx, nan)
		require.ErrorIs(t, err, sfcache.ErrInvalidKey)
	}

	require.Zero(t, calls.Load(), "the lookup ran for a key that can never be cached")
	require.Zero(t, c.Len(), "a NaN key leaked a record")
}

// PurgeExpired removes the expired entries, including revived stale values, and
// keeps the valid ones.
func Test_safety_purge_expired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var cnt counter

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		cnt.hit(key)

		return "v" + key, nil
	}, sfcache.Config{Size: 8, TTL: time.Hour},
		sfcache.WithTTLFunc(func(key string, _ string) time.Duration {
			if key[0] == 'e' {
				return time.Millisecond
			}

			return time.Hour
		}))

	_, _ = c.Lookup(ctx, "e1")
	_, _ = c.Lookup(ctx, "e2")
	_, _ = c.Lookup(ctx, "keep")

	time.Sleep(5 * time.Millisecond)

	require.Equal(t, 2, c.PurgeExpired())
	require.Equal(t, 1, c.Len())

	_, _ = c.Lookup(ctx, "keep")

	require.Equal(t, 1, cnt.get("keep"), "PurgeExpired removed a valid value")
}

// single flight — exactly one upstream call per cold key under a stampede.
func Test_safety_single_flight(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var (
		cnt     counter
		running atomic.Int64
		maxSeen atomic.Int64
	)

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		cnt.hit(key)

		if n := running.Add(1); n > maxSeen.Load() {
			maxSeen.Store(n)
		}

		defer running.Add(-1)

		time.Sleep(time.Millisecond)

		return "v" + key, nil
	}, sfcache.Config{Size: 128, TTL: time.Hour})

	for round := range 20 {
		key := "k" + strconv.Itoa(round)

		var wg sync.WaitGroup

		for range 64 {
			wg.Go(func() {
				_, err := c.Lookup(ctx, key)
				assert.NoError(t, err)
			})
		}

		wg.Wait()
		require.Equalf(t, 1, cnt.get(key), "the lookup ran more than once for %s", key)
	}

	require.LessOrEqual(t, maxSeen.Load(), int64(1), "two producers ran at once")
}

// MaxStaleOnFailure protects a COLD key, MaxStale does not — the
// distinction stale-if-error exists for.
func Test_safety_stale_windows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	build := func(cfg sfcache.Config) (*sfcache.Cache[string, string], *bool) {
		down := false

		return sfcache.New(func(_ context.Context, key string) (string, error) {
			if down {
				return "", errDown
			}

			return "v" + key, nil
		}, cfg), &down
	}

	rfc, rfcDown := build(sfcache.Config{Size: 4, TTL: 10 * time.Millisecond, MaxStale: 20 * time.Millisecond})
	fail, failDown := build(sfcache.Config{Size: 4, TTL: 10 * time.Millisecond, MaxStaleOnFailure: time.Hour})

	_, _ = rfc.Lookup(ctx, "k")
	_, _ = fail.Lookup(ctx, "k")

	time.Sleep(80 * time.Millisecond) // idle past TTL+MaxStale

	*rfcDown, *failDown = true, true

	_, err := rfc.Lookup(ctx, "k")
	require.Error(t, err, "MaxStale must not protect a key idle past TTL+MaxStale")

	got, err := fail.Lookup(ctx, "k")
	require.NoError(t, err, "MaxStaleOnFailure must protect a cold key")
	require.Equal(t, "vk", got)
}

// a revived value is served with a NIL error and every call keeps retrying the
// refresh, so the first success replaces it.
func Test_safety_revived_value_keeps_refreshing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var cnt counter

	down := false
	served := "vk"

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		cnt.hit(key)

		if down {
			return "", errDown
		}

		return served, nil
	}, sfcache.Config{Size: 4, TTL: time.Millisecond, MaxStaleOnFailure: time.Hour})

	_, _ = c.Lookup(ctx, "k")

	time.Sleep(5 * time.Millisecond)

	down = true

	for range 3 {
		got, err := c.Lookup(ctx, "k")
		require.NoError(t, err)
		require.Equal(t, "vk", got)
	}

	require.Equal(t, 4, cnt.get("k"), "a revived value must still attempt a refresh on every call")

	down = false
	served = "fresh"

	got, err := c.Lookup(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, "fresh", got, "the first success must replace the stale value")
}

// a stale revive stores a value, so it makes room like any other
// store — but a refresh that FAILED must never cost the cache a live value. The
// cache is left over capacity instead.
//
// It takes an interleaving to reach the state where the choice is even offered: the
// refresh must be in flight (so its own key has vacated the map) while a successful
// store fills the map to capacity with valid entries. Only then does the revive land
// on a full cache with nothing expendable in it.
func Test_safety_revive_does_not_evict_a_valid_entry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var (
		cnt     counter
		down    atomic.Bool
		arrived = make(chan struct{})
		release = make(chan struct{})
		refresh sync.WaitGroup
	)

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		cnt.hit(key)

		if (key == "A") && down.Load() {
			close(arrived)
			<-release

			return "", errDown
		}

		return "v" + key, nil
	}, sfcache.Config{Size: 2, TTL: time.Hour, MaxStaleOnFailure: time.Hour},
		sfcache.WithTTLFunc(func(key string, _ string) time.Duration {
			if key == "A" {
				return 5 * time.Millisecond
			}

			return time.Hour
		}))

	_, _ = c.Lookup(ctx, "A")  // expires in 5ms
	_, _ = c.Lookup(ctx, "V1") // valid for an hour; the cache is now full

	time.Sleep(20 * time.Millisecond) // A expires

	down.Store(true)

	var (
		refreshed  string
		refreshErr error
	)

	refresh.Go(func() {
		refreshed, refreshErr = c.Lookup(ctx, "A") // refresh fails -> revive
	})

	<-arrived // A is in flight, so A has vacated the map

	_, err := c.Lookup(ctx, "V2") // a second valid value: the map is at capacity
	require.NoError(t, err)

	close(release) // now the revive must make room on a cache holding only valid entries

	refresh.Wait()

	require.NoError(t, refreshErr)
	require.Equal(t, "vA", refreshed)

	_, _ = c.Lookup(ctx, "V1")
	_, _ = c.Lookup(ctx, "V2")

	require.Equal(t, 1, cnt.get("V1"), "a FAILED refresh evicted the valid entry V1")
	require.Equal(t, 1, cnt.get("V2"), "a FAILED refresh evicted the valid entry V2")
}

// a failing lookup must still SHED the values that have turned worthless — the
// ones that expired with no stale window left to serve them. They can never be
// returned to anybody, so a store that may take nothing else may still take them,
// and a cache under a total outage must not go on holding a full Size of dead values.
func Test_safety_failing_lookup_sheds_worthless_values(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	down := false

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		if down {
			return "", errDown
		}

		return "v" + key, nil
	}, sfcache.Config{Size: 4, TTL: 5 * time.Millisecond}) // no stale window at all

	for i := range 4 {
		_, err := c.Lookup(ctx, "k"+strconv.Itoa(i))
		require.NoError(t, err)
	}

	require.Equal(t, 4, c.Len())

	time.Sleep(20 * time.Millisecond) // every value is now worthless: expired, no stale window

	down = true

	for i := range 8 {
		_, err := c.Lookup(ctx, "bad"+strconv.Itoa(i))
		require.Error(t, err)
	}

	// The residue of each failing lookup must fit INSIDE Size, which it can only do
	// by shedding a dead value: a cache that could not take them would sit at Size+1.
	require.LessOrEqualf(t, c.Len(), 4, "the failing lookups did not reclaim the dead values: Len=%d", c.Len())
}

// Reset clears every entry and releases every parked caller.
//
// It must leave NOTHING behind — not in the map, and not in the eviction machinery. A
// queue that survives a Reset names keys the cache no longer holds, and the eviction
// that later offers one of those ghosts as its victim drops nothing and gives up, so the
// capacity silently stops being enforced.
func Test_safety_reset_clears_everything(t *testing.T) {
	t.Parallel()

	const size = 4

	var down atomic.Bool

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		if down.Load() {
			return "", errDown
		}

		return "v" + key, nil
	}, sfcache.Config{Size: size, TTL: time.Millisecond, MaxStaleOnFailure: time.Hour})

	// Fill it, expire it, then drive every key into the revived-stale state, so that the
	// eviction machinery is holding entries of every kind when the Reset lands.
	for i := range size {
		_, err := c.Lookup(t.Context(), strconv.Itoa(i))
		require.NoError(t, err)
	}

	time.Sleep(5 * time.Millisecond)

	down.Store(true)

	for i := range size {
		_, err := c.Lookup(t.Context(), strconv.Itoa(i)) // failed refresh -> revived stale
		require.NoError(t, err)
	}

	_, _ = c.Lookup(t.Context(), "cold") // and some error residue

	c.Reset()

	require.Zero(t, c.Len(), "Reset must clear every entry")

	// The capacity must still be enforced afterwards. If a queue kept its entries, the
	// eviction now names keys the map does not hold, drops nothing, and gives up.
	down.Store(false)

	for i := range size * 8 {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("post%d", i))
		require.NoError(t, err)
		require.LessOrEqualf(t, c.Len(), size+2,
			"the cache grew past its capacity after a Reset: the eviction is naming keys that are"+
				" no longer held, so it drops nothing (Len=%d at store %d)", c.Len(), i)
	}
}

// Remove deletes the entry and releases the callers parked on a lookup for it. A
// released caller must RETRY — taking the result of a fresh lookup — rather than inherit
// the outcome of the lookup that Remove invalidated.
func Test_safety_remove_releases_the_parked_callers(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	var (
		entered = make(chan struct{}) // the producer is inside the lookup
		release = make(chan struct{}) // ...and stays there until this closes
	)

	c := sfcache.New(func(_ context.Context, _ string) (string, error) {
		if calls.Add(1) == 1 {
			close(entered)

			<-release // hold the first lookup open, so a caller can park on it

			return "first", nil
		}

		return "second", nil
	}, sfcache.Config{Size: 4, TTL: time.Hour})

	var wg sync.WaitGroup

	wg.Go(func() { _, _ = c.Lookup(context.Background(), "k") }) // the producer

	// The producer must be IN the lookup before the other caller starts, or the two race
	// for the producer's slot and the one meant to park can end up holding the flight.
	<-entered

	got := make(chan string, 1)

	wg.Go(func() {
		v, err := c.Lookup(context.Background(), "k") // parks on the producer's flight
		assert.NoError(t, err)

		got <- v
	})

	requireParkedOnLookup(t, "Test_safety_remove_releases_the_parked_callers")

	c.Remove("k") // invalidates the flight and releases whoever is parked on it

	select {
	case v := <-got:
		require.Equal(t, "second", v,
			"the released caller took the result of the lookup Remove invalidated, instead of retrying")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the caller parked on the removed key was never released")
	}

	require.EqualValues(t, 2, calls.Load(), "the released caller must run a FRESH lookup")

	close(release)
	wg.Wait()
}

// requireParkedOnLookup blocks until a goroutine whose stack names marker is parked
// inside Lookup, waiting on a flight. It reads only goroutine stacks, so it reaches
// nothing inside the package.
func requireParkedOnLookup(t *testing.T, marker string) {
	t.Helper()

	require.Eventually(t, func() bool {
		buf := make([]byte, 1<<20)
		buf = buf[:runtime.Stack(buf, true)]

		for g := range strings.SplitSeq(string(buf), "\n\n") {
			if strings.Contains(g, marker) && strings.Contains(g, "[select") &&
				strings.Contains(g, ").Lookup(") {
				return true
			}
		}

		return false
	}, 5*time.Second, time.Millisecond, "no caller ever parked on the in-flight lookup")
}

// No code the CALLER supplies may run while the cache holds its lock: not the TTL
// function, and not the Unwrap/Is methods of the caller's error, which errors.Is calls
// while deciding whether a failure was context-induced.
//
// Both are ordinary things to write with a lock inside them. Run under the cache's
// exclusive lock, either one deadlocks against a goroutine that holds that lock and calls
// into the cache, and the whole cache wedges for good — every key, including a cache hit.
func Test_safety_no_caller_code_runs_under_the_cache_lock(t *testing.T) {
	t.Parallel()

	t.Run("the TTL function", func(t *testing.T) {
		t.Parallel()

		var appMux sync.Mutex

		entered := make(chan struct{})

		c := sfcache.New(func(_ context.Context, _ string) (string, error) {
			return "v", nil
		}, sfcache.Config{Size: 4, TTL: time.Minute},
			sfcache.WithTTLFunc(func(_ string, _ string) time.Duration {
				close(entered)

				appMux.Lock() // must not be reached while the cache holds its lock
				defer appMux.Unlock()

				return time.Minute
			}))

		appMux.Lock() // the application holds ITS lock first

		go func() { _, _ = c.Lookup(context.Background(), "k") }()

		requireNotWedged(t, c, &appMux, entered)
	})

	t.Run("the error chain", func(t *testing.T) {
		t.Parallel()

		var appMux sync.Mutex

		entered := make(chan struct{})

		// ONE error value, reused: a retry must not close the channel twice.
		failure := &lockingError{mux: &appMux, entered: entered}

		// The producer's OWN context must end, or errors.Is short-circuits on a nil
		// target and never walks the chain at all.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := sfcache.New(func(_ context.Context, _ string) (string, error) {
			cancel()

			return "", failure
		}, sfcache.Config{Size: 4, TTL: time.Minute})

		appMux.Lock() // the application holds ITS lock first

		go func() { _, _ = c.Lookup(ctx, "k") }()

		requireNotWedged(t, c, &appMux, entered)
	})
}

// lockingError is an application error whose Is method takes an application lock. An
// error with a mutable field behind a mutex is ordinary Go, and errors.Is calls Is while
// walking the chain.
type lockingError struct {
	mux     *sync.Mutex
	entered chan struct{}
	once    sync.Once
}

func (e *lockingError) Error() string { return "upstream failed" }

func (e *lockingError) Is(_ error) bool {
	e.once.Do(func() { close(e.entered) })

	e.mux.Lock()
	defer e.mux.Unlock()

	return false
}

// requireNotWedged waits until the caller-supplied code is running and blocked on the
// application's lock, then requires that the cache is still usable meanwhile. The caller
// must already hold appMux and have started the lookup.
func requireNotWedged(t *testing.T, c *sfcache.Cache[string, string], appMux *sync.Mutex, entered <-chan struct{}) {
	t.Helper()

	select {
	case <-entered: // the caller's code is running, and is about to block on appMux
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the caller-supplied code was never reached: the fixture is wrong")
	}

	// If that code is running under the cache's lock, this cannot complete: it needs the
	// cache's lock, which the caller's code is holding, and the caller's code cannot
	// return until appMux is free, which this goroutine is holding. Deadlock.
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer appMux.Unlock()

		_ = c.Len()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the cache is wedged: caller-supplied code was run under the cache's lock")
	}
}

// With BOTH windows configured, a value whose MaxStale window has already shut is still
// NOT worthless: the next failed refresh opens a MaxStaleOnFailure window on it. A
// failing lookup may reclaim only what is worthless, so it must not destroy that value —
// and the refresh that follows must still be able to serve it.
func Test_safety_a_shut_max_stale_window_is_not_worthless_under_both(t *testing.T) {
	t.Parallel()

	const (
		ttl      = 10 * time.Millisecond
		maxStale = 20 * time.Millisecond
	)

	var down atomic.Bool

	c := sfcache.New(func(_ context.Context, key string) (string, error) {
		if down.Load() || strings.HasPrefix(key, "bad") {
			return "", errDown
		}

		return "v" + key, nil
	}, sfcache.Config{Size: 1, TTL: ttl, MaxStale: maxStale, MaxStaleOnFailure: time.Hour})

	_, err := c.Lookup(t.Context(), "keep")
	require.NoError(t, err)

	// Past the expiry AND past the MaxStale window: only MaxStaleOnFailure can still
	// revive this value, so it is not worthless and may not be reclaimed.
	time.Sleep(ttl + maxStale + (10 * time.Millisecond))

	// A failing lookup at capacity. It may take only a worthless entry, and there is
	// none: it must leave "keep" alone.
	_, err = c.Lookup(t.Context(), "bad")
	require.Error(t, err)

	// The upstream is down, so the only way to answer is to revive "keep" — which is
	// possible only if the failing lookup above did not destroy it.
	down.Store(true)

	got, err := c.Lookup(t.Context(), "keep")
	require.NoError(t, err,
		"a failing lookup reclaimed a value whose MaxStale window had shut, but which"+
			" MaxStaleOnFailure could still have revived")
	require.Equal(t, "vkeep", got)
}
