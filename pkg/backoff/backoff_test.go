package backoff

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	s := New(Config{
		Base:     100 * time.Millisecond,
		Factor:   2,
		Jitter:   50 * time.Millisecond,
		MaxDelay: 30 * time.Second,
	})

	require.NotNil(t, s)
	require.InDelta(t, float64(100*time.Millisecond), s.next, 0)
	require.InDelta(t, 2.0, s.factor, 0)
	require.Equal(t, 50*time.Millisecond, s.jitter)
	require.Equal(t, 30*time.Second, s.maxDelay)
}

func TestSchedule_Next_progressionAndClamp(t *testing.T) {
	t.Parallel()

	// Jitter 0 makes the returned delays fully deterministic.
	s := New(Config{
		Base:     100 * time.Millisecond,
		Factor:   2,
		Jitter:   0,
		MaxDelay: 350 * time.Millisecond,
	})

	require.Equal(t, 100*time.Millisecond, s.Next()) // not clamped
	require.Equal(t, 200*time.Millisecond, s.Next()) // not clamped
	require.Equal(t, 350*time.Millisecond, s.Next()) // 400ms computed -> clamped
	require.Equal(t, 350*time.Millisecond, s.Next()) // 800ms computed -> clamped
}

func TestSchedule_Next_constantFactor(t *testing.T) {
	t.Parallel()

	s := New(Config{
		Base:     40 * time.Millisecond,
		Factor:   1, // no growth
		Jitter:   0,
		MaxDelay: time.Second,
	})

	require.Equal(t, 40*time.Millisecond, s.Next())
	require.Equal(t, 40*time.Millisecond, s.Next())
	require.Equal(t, 40*time.Millisecond, s.Next())
}

func TestSchedule_Next_maxDelayUnsetUsesSafetyCap(t *testing.T) {
	t.Parallel()

	// MaxDelay <= 0 exercises the safety-cap branch; the small base stays well
	// under the cap so the values remain deterministic.
	s := New(Config{
		Base:     100 * time.Millisecond,
		Factor:   2,
		Jitter:   0,
		MaxDelay: 0,
	})

	require.Equal(t, 100*time.Millisecond, s.Next())
	require.Equal(t, 200*time.Millisecond, s.Next())
	require.Equal(t, 400*time.Millisecond, s.Next())
}

func TestSchedule_Next_growthIsCapped(t *testing.T) {
	t.Parallel()

	// A huge factor pushes the stored progression past maxSafeDelay after the
	// first step, exercising the internal cap and proving the delay never
	// overflows into a negative duration.
	s := New(Config{
		Base:     1 << 40, // ~18 minutes in ns, exactly representable in float64
		Factor:   1e18,
		Jitter:   0,
		MaxDelay: 0,
	})

	first := s.Next()
	require.Equal(t, time.Duration(1<<40), first)

	// After the huge multiply, next is capped, so subsequent delays sit at the
	// safety cap and stay positive.
	capped := s.Next()
	require.Positive(t, capped)
	require.Equal(t, time.Duration(int64(float64(maxSafeDelay))), capped)
	require.LessOrEqual(t, capped, maxSafeDelay+2)
}

func TestSchedule_Next_overlargeMaxDelayNeverOverflows(t *testing.T) {
	t.Parallel()

	// Base and MaxDelay both at the int64 ceiling. The pre-jitter clamp must be
	// pulled down to the internal safety cap before the float64->int64
	// conversion, otherwise int64(float64(MaxInt64)) would overflow into a
	// negative duration. Every returned delay must stay strictly positive.
	s := New(Config{
		Base:     time.Duration(math.MaxInt64),
		Factor:   2,
		Jitter:   0,
		MaxDelay: time.Duration(math.MaxInt64),
	})

	for range 5 {
		d := s.Next()
		require.Positive(t, d)
		require.LessOrEqual(t, d, maxSafeDelay+2)
	}
}

func TestSchedule_Next_neverNegativeForBadInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
	}{
		{"negative base", Config{Base: -100 * time.Millisecond, Factor: 2, Jitter: 10 * time.Millisecond, MaxDelay: time.Second}},
		{"negative factor", Config{Base: 100 * time.Millisecond, Factor: -2, Jitter: 0, MaxDelay: time.Second}},
		{"nan factor", Config{Base: 100 * time.Millisecond, Factor: math.NaN(), Jitter: 0, MaxDelay: time.Second}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := New(tt.cfg)
			for range 6 {
				require.GreaterOrEqual(t, s.Next(), time.Duration(0), "delay must never be negative")
			}
		})
	}
}

func TestSchedule_Next_withJitterStaysInRange(t *testing.T) {
	t.Parallel()

	const (
		base   = 100 * time.Millisecond
		jitter = 50 * time.Millisecond
	)

	s := New(Config{
		Base:     base,
		Factor:   1, // keep the base constant so the bound is easy to assert
		Jitter:   jitter,
		MaxDelay: time.Second,
	})

	for range 200 {
		d := s.Next()
		require.GreaterOrEqual(t, d, base)
		require.Less(t, d, base+jitter)
	}
}

func TestJitterStrategy_Valid(t *testing.T) {
	t.Parallel()

	require.True(t, JitterAdditive.Valid())
	require.True(t, JitterFull.Valid())
	require.True(t, JitterEqual.Valid())
	require.False(t, JitterStrategy(-1).Valid())
	require.False(t, JitterStrategy(99).Valid())
}

func TestSchedule_Next_fullJitter(t *testing.T) {
	t.Parallel()

	const base = 100 * time.Millisecond

	s := New(Config{
		Base:     base,
		Factor:   1,                      // constant delay so the bound is easy to assert
		Jitter:   500 * time.Millisecond, // must be ignored under full jitter
		MaxDelay: time.Second,
		Strategy: JitterFull,
	})

	for range 200 {
		d := s.Next()
		require.GreaterOrEqual(t, d, time.Duration(0))
		require.Less(t, d, base, "full jitter must stay below the delay (Jitter field ignored)")
	}
}

func TestSchedule_Next_equalJitter(t *testing.T) {
	t.Parallel()

	const base = 100 * time.Millisecond

	s := New(Config{
		Base:     base,
		Factor:   1,
		Jitter:   500 * time.Millisecond, // must be ignored under equal jitter
		MaxDelay: time.Second,
		Strategy: JitterEqual,
	})

	for range 200 {
		d := s.Next()
		require.GreaterOrEqual(t, d, base/2, "equal jitter guarantees at least half the delay")
		require.Less(t, d, base)
	}
}

func TestSchedule_Next_unknownStrategyFallsBackToAdditive(t *testing.T) {
	t.Parallel()

	const (
		base   = 100 * time.Millisecond
		jitter = 50 * time.Millisecond
	)

	s := New(Config{
		Base:     base,
		Factor:   1,
		Jitter:   jitter,
		MaxDelay: time.Second,
		Strategy: JitterStrategy(99), // undefined -> additive fallback
	})

	for range 200 {
		d := s.Next()
		require.GreaterOrEqual(t, d, base)
		require.Less(t, d, base+jitter)
	}
}

func TestAddJitter_disabledCeil(t *testing.T) {
	t.Parallel()

	require.Equal(t, 100*time.Millisecond, AddJitter(100*time.Millisecond, 0))
	require.Equal(t, 100*time.Millisecond, AddJitter(100*time.Millisecond, -5))
}

func TestAddJitter_inRange(t *testing.T) {
	t.Parallel()

	const (
		base = 100 * time.Millisecond
		ceil = 50 * time.Millisecond
	)

	for range 200 {
		d := AddJitter(base, ceil)
		require.GreaterOrEqual(t, d, base)
		require.Less(t, d, base+ceil)
	}
}

func TestAddJitterValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		base   int64
		jitter int64
		want   int64
	}{
		{
			name:   "no overflow",
			base:   int64(100 * time.Millisecond),
			jitter: int64(50 * time.Millisecond),
			want:   int64(150 * time.Millisecond),
		},
		{
			name:   "zero jitter",
			base:   int64(100 * time.Millisecond),
			jitter: 0,
			want:   int64(100 * time.Millisecond),
		},
		{
			name:   "overflow saturates at MaxInt64",
			base:   math.MaxInt64 - 5,
			jitter: 10,
			want:   math.MaxInt64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, addJitterValue(tt.base, tt.jitter))
		})
	}
}
