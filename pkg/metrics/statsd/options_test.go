package statsd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithPrefix(t *testing.T) {
	t.Parallel()

	c := &Client{}
	want := "TEST"
	WithPrefix(want)(c)
	require.Equal(t, want, c.prefix, "WithPrefix() expecting %v, got %v", want, c.prefix)
}

func TestWithNetwork(t *testing.T) {
	t.Parallel()

	c := &Client{}
	want := "tcp"
	WithNetwork(want)(c)
	require.Equal(t, want, c.network, "WithNetwork() expecting %v, got %v", want, c.network)
}

func TestWithAddress(t *testing.T) {
	t.Parallel()

	c := &Client{}
	want := ":6053"
	WithAddress(want)(c)
	require.Equal(t, want, c.address, "WithAddress() expecting %v, got %v", want, c.address)
}

func TestWithFlushPeriod(t *testing.T) {
	t.Parallel()

	c := &Client{}
	want := time.Duration(101) * time.Millisecond
	WithFlushPeriod(want)(c)
	require.Equal(t, want, c.flushPeriod, "WithFlushPeriod() expecting %v, got %v", want, c.flushPeriod)
}
