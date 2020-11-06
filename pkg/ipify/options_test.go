package ipify

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	want := 17 * time.Second
	c := &Client{}
	WithTimeout(want)(c)
	require.Equal(t, want, c.timeout, "WithTimeout() = %want, want %want", c.timeout, want)
}

func TestWithURL(t *testing.T) {
	t.Parallel()

	want := "https://test.ipify.invalid"
	c := &Client{}
	WithURL(want)(c)
	require.Equal(t, want, c.apiURL, "WithURL() = %want, want %want", c.apiURL, want)
}

func TestWithErrorIP(t *testing.T) {
	t.Parallel()

	want := "0.0.0.0"
	c := &Client{}
	WithErrorIP(want)(c)
	require.Equal(t, want, c.errorIP, "WithErrorIP() = %want, want %want", c.errorIP, want)
}
