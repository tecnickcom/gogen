package sfcache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	testLookup := func(_ context.Context, key string) (any, error) {
		return key, nil
	}

	got := New(testLookup, 3, 5*time.Second)
	require.NotNil(t, got)

	require.NotNil(t, got.lookupFn)
	require.NotNil(t, got.mux)

	require.Equal(t, 3, got.size)
	require.Equal(t, 5*time.Second, got.ttl)

	require.NotNil(t, got.keymap)
	require.Empty(t, got.keymap)

	got = New(nil, 0, 1*time.Second)
	require.Equal(t, 1, got.size)
}

func Test_Reset(t *testing.T) {
	t.Parallel()

	c := New(nil, 1, 1*time.Second)

	c.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Unix(),
		},
	}

	c.Reset()

	require.Empty(t, c.keymap)
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	c := New(nil, 3, 1*time.Second)

	c.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Unix(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Unix(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Unix(),
		},
	}

	c.Remove("example.net")

	require.Len(t, c.keymap, 2)
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.org")
}

func Test_evict_expired(t *testing.T) {
	t.Parallel()

	r := New(nil, 3, 1*time.Minute)

	r.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Add(-2 * time.Second).Unix(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Add(11 * time.Second).Unix(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Add(13 * time.Second).Unix(),
		},
	}

	require.Len(t, r.keymap, 3)

	r.evict()

	require.Len(t, r.keymap, 2)
	require.Contains(t, r.keymap, "example.org")
	require.Contains(t, r.keymap, "example.net")
}

func Test_evict_oldest(t *testing.T) {
	t.Parallel()

	c := New(nil, 3, 1*time.Second)

	c.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Add(11 * time.Second).Unix(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Add(7 * time.Second).Unix(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Add(13 * time.Second).Unix(),
		},
	}

	c.evict()

	require.Len(t, c.keymap, 2)
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.net")
}

/*
NOTE:
The IP blocks 192.0.2.0/24 (TEST-NET-1), 198.51.100.0/24 (TEST-NET-2),
and 203.0.113.0/24 (TEST-NET-3) are provided for use in documentation.
*/

func Test_set(t *testing.T) {
	t.Parallel()

	c := New(nil, 2, 10*time.Second)

	c.set("example.com", []string{"192.0.2.1"}, nil, nil)
	time.Sleep(1 * time.Second)
	c.set("example.org", []string{"192.0.2.2", "198.51.100.2"}, nil, nil)

	require.Len(t, c.keymap, 2)
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.org")

	c.set("example.net", []string{"192.0.2.3", "198.51.100.3", "203.0.113.3"}, nil, nil)

	require.Len(t, c.keymap, 2)
	require.Contains(t, c.keymap, "example.org")
	require.Contains(t, c.keymap, "example.net")

	c.set("example.net", []string{"198.51.100.4"}, nil, nil)

	require.Len(t, c.keymap, 2)
	require.Contains(t, c.keymap, "example.org")
	require.Contains(t, c.keymap, "example.net")
	require.Equal(t, []string{"198.51.100.4"}, c.keymap["example.net"].val)
}

func Test_Lookup(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		ip := fmt.Sprintf("192.0.2.%d", i)

		return []string{ip}, nil
	}

	c := New(lookupFn, 1, 1*time.Second)

	// cache miss
	val, err := c.Lookup(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, val)

	// cache hit
	val, err = c.Lookup(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, val)

	time.Sleep(1 * time.Second)

	// cache expired
	val, err = c.Lookup(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.2"}, val)

	// cache miss with eviction
	val, err = c.Lookup(context.TODO(), "example.net")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.3"}, val)

	// deleted entry on duplicate lookup
	wait := make(chan struct{})

	c.mux.Lock()
	c.set("example.org", nil, nil, wait)
	c.mux.Unlock()

	go func() {
		time.Sleep(5 * time.Millisecond)
		c.Remove("example.org")
		close(wait)
	}()

	val, err = c.Lookup(context.TODO(), "example.org")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.4"}, val)

	// context expired on duplicate lookup
	wait = make(chan struct{})

	c.mux.Lock()
	c.set("example.org", nil, nil, wait)
	c.mux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	c.set("example.org", nil, nil, wait)

	val, err = c.Lookup(ctx, "example.org")
	require.Error(t, err)
	require.Nil(t, val)
}

func Test_Lookup_concurrent_slow(t *testing.T) {
	t.Parallel()

	const nlookup = 10

	type retval struct {
		err error
		val []string
	}

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		time.Sleep(300 * time.Millisecond) // simulate slow lookup

		i++
		ip := fmt.Sprintf("192.0.2.%d", i)

		return []string{ip}, nil
	}

	c := New(lookupFn, 2, 0)
	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Add(1)

		go func() {
			defer wg.Done()

			val, err := c.Lookup(context.TODO(), "example.org")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.NoError(t, v.err)
		require.NotNil(t, v.val)
		require.Len(t, v.val, 1)
		require.Equal(t, []string{"192.0.2.1"}, v.val)
	}
}

func Test_Lookup_concurrent_fast(t *testing.T) {
	t.Parallel()

	const nlookup = 1234

	type retval struct {
		err error
		val []string
	}

	lookupFn := func(_ context.Context, _ string) (any, error) {
		return []string{"192.0.2.13"}, nil
	}

	// With ttl = 0 the items expires immediately causing stress on the concurrent lookups.
	// This covers the case when the cache entry was updated during the wait.
	// This should not happen in real world scenarios, but it's good to have it covered.

	c := New(lookupFn, 2, 0)
	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Add(1)

		go func() {
			defer wg.Done()

			val, err := c.Lookup(context.TODO(), "example.org")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.NoError(t, v.err)
		require.NotNil(t, v.val)
		require.Len(t, v.val, 1)
		require.Equal(t, []string{"192.0.2.13"}, v.val)
	}
}

func Test_Lookup_error(t *testing.T) {
	t.Parallel()

	const nlookup = 10

	type retval struct {
		err error
		val []string
	}

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		time.Sleep(300 * time.Millisecond) // simulate slow lookup

		i++

		return nil, fmt.Errorf("mock error: %d", i)
	}

	c := New(lookupFn, 2, 10*time.Second)

	val, err := c.Lookup(context.TODO(), "example.com")
	require.Error(t, err)
	require.Nil(t, val)

	// test concurrent lookups

	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Add(1)

		go func() {
			defer wg.Done()

			val, err := c.Lookup(context.TODO(), "example.net")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "mock error: 2", v.err.Error())
		require.Nil(t, v.val)
	}
}

func Test_Lookup_error_concurrent_fast(t *testing.T) {
	t.Parallel()

	const nlookup = 100

	type retval struct {
		err error
		val []string
	}

	lookupFn := func(_ context.Context, _ string) (any, error) {
		return nil, errors.New("mock error")
	}

	c := New(lookupFn, 2, 0)

	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Add(1)

		go func() {
			defer wg.Done()

			val, err := c.Lookup(context.TODO(), "example.net")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "mock error", v.err.Error())
		require.Nil(t, v.val)
	}
}
