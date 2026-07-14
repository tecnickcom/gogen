package periodic

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithInitialJitter(t *testing.T) {
	t.Parallel()

	p, err := New(10*time.Millisecond, 5*time.Millisecond, time.Millisecond, func(_ context.Context) {}, WithInitialJitter())
	require.NoError(t, err)
	require.True(t, p.jitterFirst)

	// The default (no option) leaves the first tick eager.
	def, err := New(10*time.Millisecond, 5*time.Millisecond, time.Millisecond, func(_ context.Context) {})
	require.NoError(t, err)
	require.False(t, def.jitterFirst)
}

func Test_firstDelay_eagerByDefault(t *testing.T) {
	t.Parallel()

	// Without WithInitialJitter the first tick is eager regardless of the jitter ceiling.
	p := &Periodic{interval: 10 * time.Millisecond, jitter: 5 * time.Millisecond}

	require.Equal(t, time.Nanosecond, p.firstDelay())
}

func Test_firstDelay_withInitialJitter_zeroJitter(t *testing.T) {
	t.Parallel()

	// WithInitialJitter is a no-op when jitter is disabled: the first tick stays eager
	// (and no rand draw on a zero ceiling, so it can never panic).
	p := &Periodic{interval: 10 * time.Millisecond, jitter: 0, jitterFirst: true}

	require.Equal(t, time.Nanosecond, p.firstDelay())
}

func Test_firstDelay_withInitialJitter(t *testing.T) {
	t.Parallel()

	const jitter = 5 * time.Millisecond

	p := &Periodic{interval: 10 * time.Millisecond, jitter: jitter, jitterFirst: true}

	// The first delay is spread across [0, jitter) — the interval is not added.
	var maxSeen time.Duration

	for range 200 {
		d := p.firstDelay()
		require.GreaterOrEqual(t, d, time.Duration(0))
		require.Less(t, d, jitter)

		if d > maxSeen {
			maxSeen = d
		}
	}

	// The draws must actually span the range, not collapse to the eager 1 ns: over
	// 200 uniform draws in [0, jitter) at least one exceeding jitter/2 is a certainty.
	require.Greater(t, maxSeen, jitter/2, "first delay is not being spread across [0, jitter)")
}

func Test_Start_Stop_withInitialJitter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	task := func(_ context.Context) { calls.Add(1) }

	// A short jitter keeps the first fire quick; the loop must still run the task.
	p, err := New(5*time.Millisecond, 2*time.Millisecond, time.Millisecond, task, WithInitialJitter())
	require.NoError(t, err)

	p.Start(t.Context())

	time.Sleep(50 * time.Millisecond)
	p.Stop()

	require.Positive(t, calls.Load())
}
