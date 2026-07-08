package passwordpwned

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1" //nolint:gosec // SHA-1 is required by the HIBP API and mirrored here for test hashes.
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// hashParts returns the uppercase k-anonymity prefix and suffix of password's
// SHA-1 hash, matching the client's own hashing.
func hashParts(t *testing.T, password string) (string, string) {
	t.Helper()

	sum := sha1.Sum([]byte(password)) //nolint:gosec // required by the HIBP API.
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))

	return hash[:prefixLen], hash[prefixLen:]
}

// gzipBytes returns s compressed with gzip, for Content-Encoding test fixtures.
func gzipBytes(t *testing.T, s string) []byte {
	t.Helper()

	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	return buf.Bytes()
}

// TestClient_PwnedCount_PlainText exercises the Content-Encoding handling, size
// limit, response validation, and count parsing using uncompressed (identity)
// responses, which are far easier to author than brotli fixtures and represent a
// legitimate encoding a mirror or proxy may return.
//
//nolint:paralleltest,gocognit // TestClient_IsPwnedPassword mpatch-es process-global functions in the parallel phase; this test must stay in the sequential phase.
func TestClient_PwnedCount_PlainText(t *testing.T) {
	const password = "plaintext.password"

	prefix, suffix := hashParts(t, password)
	otherSuffix := strings.Repeat("F", suffixLen) // a 35-char suffix that is not password's.

	tests := []struct {
		name            string
		contentEncoding string
		body            string
		gzipBody        bool
		status          int
		sizeLimit       int64
		assertRequest   func(t *testing.T, r *http.Request)
		wantCount       int
		wantErr         bool
		wantErrIs       error
	}{
		{
			name: "pwned with multi-digit count and k-anonymity request contract",
			body: suffix + ":42\r\n" + otherSuffix + ":0\r\n",
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Only the 5-char prefix must ever be sent (privacy guarantee).
				assert.Equal(t, "/"+rangePath+"/"+prefix, r.URL.Path)
				assert.Len(t, prefix, prefixLen)
				assert.Equal(t, "true", r.Header.Get("Add-Padding"))
				assert.Equal(t, "br", r.Header.Get("Accept-Encoding"))
				assert.Equal(t, defaultUserAgent, r.Header.Get("User-Agent"))
			},
			wantCount: 42,
		},
		{
			name:            "pwned with explicit identity encoding",
			contentEncoding: "identity",
			body:            suffix + ":7\r\n",
			wantCount:       7,
		},
		{
			name:            "pwned with gzip encoding",
			contentEncoding: "gzip",
			gzipBody:        true,
			body:            suffix + ":11\r\n",
			wantCount:       11,
		},
		{
			name:      "lowercase response suffix still matches",
			body:      strings.ToLower(suffix) + ":6\r\n",
			wantCount: 6,
		},
		{
			name:      "padding entry with zero count is not pwned",
			body:      suffix + ":0\r\n",
			wantCount: 0,
		},
		{
			name:      "suffix not found is not pwned",
			body:      otherSuffix + ":5\r\n",
			wantCount: 0,
		},
		{
			name:            "unsupported content encoding is rejected",
			contentEncoding: "zstd",
			body:            suffix + ":1\r\n",
			wantErr:         true,
			wantErrIs:       ErrUnsupportedEncoding,
		},
		{
			name:            "invalid gzip body is rejected",
			contentEncoding: "gzip",
			body:            suffix + ":1\r\n", // plain text despite the gzip header
			wantErr:         true,
		},
		{
			name:      "empty body is treated as malformed",
			body:      "",
			wantErr:   true,
			wantErrIs: ErrMalformedResponse,
		},
		{
			name:      "malformed count with non-digit separator",
			body:      suffix + ":X\r\n",
			wantErr:   true,
			wantErrIs: ErrMalformedResponse,
		},
		{
			name:      "response exceeding size limit is rejected",
			body:      suffix + ":1\r\n",
			sizeLimit: 4,
			wantErr:   true,
			wantErrIs: ErrResponseTooLarge,
		},
		{
			name:      "non-200 status is rejected with sentinel",
			status:    http.StatusInternalServerError,
			wantErr:   true,
			wantErrIs: ErrUnexpectedStatus,
		},
		{
			name:      "garbage html body is treated as malformed",
			body:      "<html><body>login required</body></html>",
			wantErr:   true,
			wantErrIs: ErrMalformedResponse,
		},
		{
			name:      "body shorter than one range line is malformed",
			body:      "ABC:1",
			wantErr:   true,
			wantErrIs: ErrMalformedResponse,
		},
		{
			name:      "matched line truncated at end of body",
			body:      otherSuffix + ":0\r\n" + suffix,
			wantErr:   true,
			wantErrIs: ErrMalformedResponse,
		},
		{
			name:      "matched line with wrong separator",
			body:      otherSuffix + ":0\r\n" + suffix + ";2\r\n",
			wantErr:   true,
			wantErrIs: ErrMalformedResponse,
		},
		{
			name:      "matched line with non-digit count",
			body:      otherSuffix + ":0\r\n" + suffix + ":X\r\n",
			wantErr:   true,
			wantErrIs: ErrMalformedResponse,
		},
	}

	//nolint:paralleltest // parent stays in the sequential phase (see above), so subtests must not be parallel.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respBody := []byte(tt.body)
			if tt.gzipBody {
				respBody = gzipBytes(t, tt.body)
			}

			mux := http.NewServeMux()
			mux.HandleFunc("/"+rangePath+"/", func(w http.ResponseWriter, r *http.Request) {
				if tt.assertRequest != nil {
					tt.assertRequest(t, r)
				}

				if tt.status != 0 {
					w.WriteHeader(tt.status)

					return
				}

				if tt.contentEncoding != "" {
					w.Header().Set("Content-Encoding", tt.contentEncoding)
				}

				_, werr := w.Write(respBody)
				assert.NoError(t, werr)
			})

			ts := httptest.NewServer(mux)
			t.Cleanup(ts.Close)

			opts := []Option{WithURL(ts.URL), WithRetryAttempts(1)}
			if tt.sizeLimit > 0 {
				opts = append(opts, WithResponseSizeLimit(tt.sizeLimit))
			}

			c, err := New(opts...)
			require.NoError(t, err)

			count, err := c.PwnedCount(t.Context(), password)
			require.Equal(t, tt.wantErr, err != nil, err)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			}

			require.Equal(t, tt.wantCount, count)

			// IsPwnedPassword must agree with PwnedCount.
			pwned, perr := c.IsPwnedPassword(t.Context(), password)
			require.Equal(t, tt.wantErr, perr != nil, perr)
			require.Equal(t, tt.wantCount > 0, pwned)
		})
	}
}

// TestClient_trailingSlashURL verifies that a WithURL value with a trailing
// slash still produces a clean "/range/<prefix>" request path.
//
//nolint:paralleltest // shares the process-global mpatch constraint described on TestClient_PwnedCount_PlainText.
func TestClient_trailingSlashURL(t *testing.T) {
	const password = "trailing.slash.password"

	prefix, suffix := hashParts(t, password)

	mux := http.NewServeMux()
	mux.HandleFunc("/"+rangePath+"/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/"+rangePath+"/"+prefix, r.URL.Path) // no doubled slash

		_, err := w.Write([]byte(suffix + ":3\r\n"))
		assert.NoError(t, err)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	c, err := New(WithURL(ts.URL+"/"), WithRetryAttempts(1))
	require.NoError(t, err)

	count, err := c.PwnedCount(t.Context(), password)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

// TestClient_PwnedCount_NilResponseBody verifies that a non-conforming injected
// HTTPClient returning a response without a body yields an error, not a panic.
//
//nolint:paralleltest // shares the process-global mpatch constraint described on TestClient_PwnedCount_PlainText.
func TestClient_PwnedCount_NilResponseBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := NewMockHTTPClient(ctrl)
	mc.EXPECT().Do(gomock.Any()).Return(&http.Response{StatusCode: http.StatusOK}, nil).Times(1)

	c, err := New(WithHTTPClient(mc), WithRetryAttempts(1))
	require.NoError(t, err)

	count, err := c.PwnedCount(t.Context(), "any.password")
	require.ErrorIs(t, err, ErrMalformedResponse)
	require.Equal(t, 0, count)
}

// TestClient_PwnedCount_RetryOn429 pins the retry wiring: a 429 throttling
// response must be retried (read-request policy) and the follow-up success
// must be returned. The Retry-After honoring mechanics are tested in the
// httpretrier package.
//
//nolint:paralleltest // shares the process-global mpatch constraint described on TestClient_PwnedCount_PlainText.
func TestClient_PwnedCount_RetryOn429(t *testing.T) {
	const password = "retry.password"

	_, suffix := hashParts(t, password)

	var reqCount atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/"+rangePath+"/", func(w http.ResponseWriter, _ *http.Request) {
		if reqCount.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)

			return
		}

		_, werr := w.Write([]byte(suffix + ":9\r\n"))
		assert.NoError(t, werr)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	c, err := New(WithURL(ts.URL), WithRetryAttempts(2), WithRetryDelay(10*time.Millisecond))
	require.NoError(t, err)

	count, err := c.PwnedCount(t.Context(), password)
	require.NoError(t, err)
	require.Equal(t, 9, count)
	require.Equal(t, int32(2), reqCount.Load(), "expected the 429 to be retried exactly once")
}
