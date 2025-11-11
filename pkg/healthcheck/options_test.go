package healthcheck

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithResultWriter(t *testing.T) {
	t.Parallel()

	v := func(_ context.Context, _ http.ResponseWriter, _ int, _ any) {}
	h := &Handler{}
	WithResultWriter(v)(h)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(h.writeResult).Pointer())
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	h := &Handler{}

	v := slog.Default()
	WithLogger(v)(h)
	require.Equal(t, v, h.logger)
}
