package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/httputil"
	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/Vonage/gosrvlib/pkg/redact"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/Vonage/gosrvlib/pkg/traceid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestRequestInjectHandler(t *testing.T) {
	t.Parallel()

	nextHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())
		assert.NotNil(t, l, "logger not found")

		l.Info("injected")

		// check if the request_time can be retrieved.
		reqTime, ok := httputil.GetRequestTime(r)
		assert.True(t, ok)
		assert.NotEmpty(t, reqTime)
	})

	ctx, logs := testutil.ContextWithLogObserver(zapcore.DebugLevel)
	handler := RequestInjectHandler(logging.FromContext(ctx), traceid.DefaultHeader, redact.HTTPData, nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(nil, req)

	logEntries := logs.All()
	require.Len(t, logEntries, 1, "expected only 1 log message")

	logEntry := logEntries[0]
	logContextMap := logEntry.ContextMap()

	// check traceid
	tiValue, tiExists := logContextMap[traceid.DefaultLogKey]
	require.True(t, tiExists, "traceid field missing")
	require.NotEmpty(t, tiValue, "expected traceid value not found")

	// check request_time
	rtValue, rtExists := logContextMap["request_time"]
	require.True(t, rtExists, "request_time field missing")
	require.NotEmpty(t, rtValue, "expected request_time value not found")

	// check request_method
	mValue, mExists := logContextMap["request_method"]
	require.True(t, mExists, "request_method field missing")
	require.Equal(t, http.MethodGet, mValue)

	// check request_path
	pValue, pExists := logContextMap["request_path"]
	require.True(t, pExists, "request_path field missing")
	require.Equal(t, "/", pValue)

	// check request_query
	rqValue, rqExists := logContextMap["request_query"]
	require.True(t, rqExists, "request_query field missing")
	require.Equal(t, "", rqValue)

	// check request_user_agent
	_, ipExists := logContextMap["request_remote_address"]
	require.True(t, ipExists, "request_remote_address field missing")

	// check request_uri
	ruValue, ruExists := logContextMap["request_uri"]
	require.True(t, ruExists, "request_uri field missing")
	require.Equal(t, "/", ruValue)

	// check request_user_agent
	uaValue, uaExists := logContextMap["request_user_agent"]
	require.True(t, uaExists, "request_user_agent field missing")
	require.Equal(t, "", uaValue)

	// check request_x_forwarded_for
	rxffValue, rxffExists := logContextMap["request_x_forwarded_for"]
	require.True(t, rxffExists, "request_x_forwarded_for field missing")
	require.Equal(t, "", rxffValue)

	// message
	require.Equal(t, "injected", logEntry.Message)
}
