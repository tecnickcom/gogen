package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestContext(t *testing.T) {
	ctx := Context()

	l1 := logging.FromContext(ctx)
	l2 := logging.FromContext(ctx)
	require.Equal(t, l1, l2)
}

func TestContextWithLogObserver(t *testing.T) {
	ctx, logs := ContextWithLogObserver(zap.DebugLevel)

	l := logging.FromContext(ctx)
	l.Info("test message")

	require.Equal(t, 1, logs.Len())
	require.Equal(t, "test message", logs.All()[0].Message)
}

func TestContextWithHTTPRouterParams(t *testing.T) {
	params := map[string]string{"test_arg_1": "test_val_1", "test_arg_2": "test_val_2"}

	r := httptest.NewRequest(http.MethodGet, "http://test.url.invalid", nil)
	r = r.WithContext(ContextWithHTTPRouterParams(context.Background(), params))

	require.Equal(t, "test_val_1", httputil.PathParam(r, "test_arg_1"))
	require.Equal(t, "test_val_2", httputil.PathParam(r, "test_arg_2"))
}
