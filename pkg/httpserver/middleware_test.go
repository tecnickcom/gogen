package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/redact"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/nexmoinc/gosrvlib/pkg/traceid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestRequestInjectHandler(t *testing.T) {
	t.Parallel()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())
		require.NotNil(t, l, "logger not found")

		l.Info("injected")
	})

	ctx, logs := testutil.ContextWithLogObserver(zapcore.DebugLevel)
	handler := RequestInjectHandler(logging.FromContext(ctx), traceid.DefaultHeader, redact.HTTPData, nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(nil, req)

	logEntries := logs.All()
	require.Len(t, logEntries, 1, "expected only 1 log message")

	logEntry := logEntries[0]
	logContextMap := logEntry.ContextMap()

	// check request id
	idValue, idExists := logContextMap["traceid"]
	require.True(t, idExists, "traceid field missing")
	require.NotEmpty(t, idValue, "expected requestId value not found")

	// check method
	mValue, mExists := logContextMap["request_method"]
	require.True(t, mExists, "request_method field missing")
	require.Equal(t, http.MethodGet, mValue)

	// check path
	pValue, pExists := logContextMap["request_path"]
	require.True(t, pExists, "request_path field missing")
	require.Equal(t, "/", pValue)

	// request_query
	rqValue, rqExists := logContextMap["request_query"]
	require.True(t, rqExists, "request_query field missing")
	require.Equal(t, "", rqValue)

	// request_uri
	ruValue, ruExists := logContextMap["request_uri"]
	require.True(t, ruExists, "request_uri field missing")
	require.Equal(t, "/", ruValue)

	// request_useragent
	uaValue, uaExists := logContextMap["request_useragent"]
	require.True(t, uaExists, "request_useragent field missing")
	require.Equal(t, "", uaValue)

	// request_useragent
	_, ipExists := logContextMap["remote_ip"]
	require.True(t, ipExists, "remote_ip field missing")

	// message
	require.Equal(t, "injected", logEntry.Message)
}
