package healthcheck

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithConfigureRequest(t *testing.T) {
	t.Parallel()

	v := func(_ *http.Request) {}
	cfg := &checkConfig{}
	WithConfigureRequest(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.configureRequest).Pointer())
}
