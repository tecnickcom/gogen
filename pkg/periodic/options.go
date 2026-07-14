package periodic

// Option configures optional [Periodic] behavior. Pass options as the trailing
// arguments to [New].
type Option func(*Periodic)

// WithInitialJitter delays the first invocation by a random duration in
// [0, jitter) instead of firing it immediately. Use it when many instances of a
// service start at once (a rolling deploy, a fleet-wide restart): with the eager
// default every replica's first call lands on the target simultaneously, which is
// the very thundering herd the per-tick jitter exists to prevent. It costs up to
// jitter of extra startup latency, once, and is a no-op when jitter is 0.
func WithInitialJitter() Option {
	return func(p *Periodic) {
		p.jitterFirst = true
	}
}
