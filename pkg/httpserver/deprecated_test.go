package httpserver

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithInstrumentHandler(t *testing.T) {
	t.Parallel()

	v := func(path string, handler http.HandlerFunc) http.Handler { return handler }
	cfg := defaultConfig()
	err := WithInstrumentHandler(v)(cfg)
	require.NoError(t, err)
	require.Len(t, cfg.middleware, 1)
}
