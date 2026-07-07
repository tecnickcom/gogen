package dnscache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// refNormalizeHost is a byte-oriented reference implementation of
// normalizeHost: strip one trailing dot (unless the host is just "."), then
// fold only ASCII A-Z.
func refNormalizeHost(host string) string {
	b := []byte(host)

	if len(b) > 1 && b[len(b)-1] == '.' {
		b = b[:len(b)-1]
	}

	for i, ch := range b {
		if ch >= 'A' && ch <= 'Z' {
			b[i] = ch + ('a' - 'A')
		}
	}

	return string(b)
}

func FuzzNormalizeHost(f *testing.F) {
	for _, seed := range []string{
		"",
		".",
		"..",
		"example.com",
		"EXAMPLE.com.",
		"Example.COM",
		"xn--80ak6aa92e.com",
		"192.0.2.1",
		"2001:DB8::1",
		"A.",
		"\xff\xfe",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, host string) {
		got := normalizeHost(host)

		// Differential check against the reference implementation.
		require.Equal(t, refNormalizeHost(host), got)

		// The result must be a stable cache key for its own re-normalization
		// unless it still carries (invalid) extra trailing dots.
		if len(got) < 2 || got[len(got)-1] != '.' {
			require.Equal(t, got, normalizeHost(got))
		}
	})
}

// assertNoDuplicateDestinations fails if any two candidates identify the same
// dial destination.
func assertNoDuplicateDestinations(t *testing.T, cands []dialCandidate) {
	t.Helper()

	for i := range cands {
		for j := i + 1; j < len(cands); j++ {
			require.False(t, sameDestination(cands[i], cands[j]),
				"duplicate destination: %q and %q", cands[i].raw, cands[j].raw)
		}
	}
}

// assertFamilyFilter fails if a candidate violates the family restriction of
// the given network ('4' or '6' suffix).
func assertFamilyFilter(t *testing.T, network string, cands []dialCandidate) {
	t.Helper()

	if network == "" {
		return
	}

	suffix := network[len(network)-1]

	for _, cand := range cands {
		if !cand.addr.IsValid() {
			continue // invalid entries are always kept
		}

		switch suffix {
		case '4':
			require.False(t, isIPv6Addr(cand.addr), "IPv6 candidate %q on %q", cand.raw, network)
		case '6':
			require.True(t, isIPv6Addr(cand.addr), "IPv4 candidate %q on %q", cand.raw, network)
		}
	}
}

func FuzzOrderCandidates(f *testing.F) {
	f.Add("192.0.2.1", "2001:db8::1", "notanip", "tcp")
	f.Add("::1", "::1", "127.0.0.1", "tcp4")
	f.Add("2001:DB8::1", "2001:db8::1", "", "tcp6")
	f.Add("::ffff:192.0.2.1", "192.0.2.1", "10.0.0.1", "")
	f.Add("fe80::1%eth0", "fe80::1%eth1", "fe80::1%eth0", "udp")

	f.Fuzz(func(t *testing.T, a, b, c, network string) {
		ips := []string{a, b, c}

		cache := New(nil, 1, 1*time.Minute, WithAddressRotation())

		got := cache.orderCandidates(network, ips)

		// Every returned candidate must originate from the input, and the
		// ordering must never invent or multiply candidates.
		require.LessOrEqual(t, len(got), len(ips))

		for _, cand := range got {
			require.Contains(t, ips, cand.raw)
		}

		assertNoDuplicateDestinations(t, got)
		assertFamilyFilter(t, network, got)
	})
}
