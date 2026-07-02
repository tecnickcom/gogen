//go:generate go tool mockgen -write_package_comment=false -package httpretrier -destination ./mock_test.go . HTTPClient
package httpretrier

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/testutil"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "succeeds with defaults",
			wantErr: false,
		},
		{
			name: "succeeds with custom values",
			opts: []Option{
				WithRetryIfFn(func(_ *http.Response, _ error) bool { return true }),
				WithAttempts(5),
				WithDelay(601 * time.Millisecond),
				WithDelayFactor(1.3),
				WithJitter(109 * time.Millisecond),
			},
			wantErr: false,
		},
		{
			name: "succeeds with RetryIfForWriteRequests",
			opts: []Option{
				WithRetryIfFn(RetryIfForWriteRequests),
			},
			wantErr: false,
		},
		{
			name: "succeeds with RetryIfForReadRequests",
			opts: []Option{
				WithRetryIfFn(RetryIfForReadRequests),
			},
			wantErr: false,
		},
		{
			name:    "fails with invalid option",
			opts:    []Option{WithJitter(0)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(http.DefaultClient, tt.opts...)

			if tt.wantErr {
				require.Nil(t, c, "New() returned value should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, c, "New() returned value should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func Test_defaultRetryIf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "true with error",
			err:  errors.New("ERROR"),
			want: true,
		},
		{
			name: "false with no error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := defaultRetryIf(nil, tt.err)

			require.Equal(t, tt.want, got)
		})
	}
}

func TestRetryIfForWriteRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		err    error
		want   bool
	}{
		{
			name:   "true with error",
			status: http.StatusOK,
			err:    errors.New("ERROR"),
			want:   true,
		},
		{
			name:   "true with http.StatusTooManyRequests",
			status: http.StatusTooManyRequests,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusBadGateway",
			status: http.StatusBadGateway,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusServiceUnavailable",
			status: http.StatusServiceUnavailable,
			err:    nil,
			want:   true,
		},
		{
			name:   "false with no matching status code",
			status: http.StatusOK,
			err:    nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := &http.Response{
				Status:     http.StatusText(tt.status),
				StatusCode: tt.status,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}
			got := RetryIfForWriteRequests(response, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRetryIfForReadRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		err    error
		want   bool
	}{
		{
			name:   "true with error",
			status: http.StatusOK,
			err:    errors.New("ERROR"),
			want:   true,
		},
		{
			name:   "true with http.StatusNotFound",
			status: http.StatusNotFound,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusRequestTimeout",
			status: http.StatusRequestTimeout,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusConflict",
			status: http.StatusConflict,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusLocked",
			status: http.StatusLocked,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusTooEarly",
			status: http.StatusTooEarly,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusTooManyRequests",
			status: http.StatusTooManyRequests,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusInternalServerError",
			status: http.StatusInternalServerError,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusBadGateway",
			status: http.StatusBadGateway,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusServiceUnavailable",
			status: http.StatusServiceUnavailable,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusGatewayTimeout",
			status: http.StatusGatewayTimeout,
			err:    nil,
			want:   true,
		},
		{
			name:   "true with http.StatusInsufficientStorage",
			status: http.StatusInsufficientStorage,
			err:    nil,
			want:   true,
		},
		{
			name:   "false with no matching status code",
			status: http.StatusOK,
			err:    nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := &http.Response{
				Status:     http.StatusText(tt.status),
				StatusCode: tt.status,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}
			got := RetryIfForReadRequests(response, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRetryIfFnByHTTPMethod(t *testing.T) {
	t.Parallel()

	testResp := &http.Response{}
	testErr := errors.New("sample error")

	tests := []struct {
		name       string
		httpMethod string
		want       RetryIfFn
	}{
		{
			name:       "GET method",
			httpMethod: http.MethodGet,
			want:       RetryIfForReadRequests,
		},
		{
			name:       "POST method",
			httpMethod: http.MethodPost,
			want:       RetryIfForWriteRequests,
		},
		{
			name:       "PUT method",
			httpMethod: http.MethodPut,
			want:       RetryIfForWriteRequests,
		},
		{
			name:       "PATCH method",
			httpMethod: http.MethodPatch,
			want:       RetryIfForWriteRequests,
		},
		{
			name:       "DELETE method",
			httpMethod: http.MethodDelete,
			want:       RetryIfForWriteRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RetryIfFnByHTTPMethod(tt.httpMethod)

			require.Equal(t, tt.want(testResp, testErr), got(testResp, testErr))
		})
	}
}

//nolint:gocognit
func TestHTTPRetrier_Do(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setupMocks       func(mock *MockHTTPClient)
		ctxTimeout       time.Duration
		body             io.Reader
		wantStatus       int
		wantErr          bool
		requestBodyError bool
	}{
		{
			name: "success at first attempt",
			setupMocks: func(mock *MockHTTPClient) {
				rOK := &http.Response{
					Status:     http.StatusText(http.StatusOK),
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				mock.EXPECT().Do(gomock.Any()).Return(rOK, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success at first attempt with body",
			setupMocks: func(mock *MockHTTPClient) {
				rOK := &http.Response{
					Status:     http.StatusText(http.StatusOK),
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				mock.EXPECT().Do(gomock.Any()).Return(rOK, nil)
			},
			body:       bytes.NewReader([]byte(`some body`)),
			wantStatus: http.StatusOK,
		},
		{
			name: "success at third attempt after multiple retry conditions",
			setupMocks: func(mock *MockHTTPClient) {
				rErr := &http.Response{
					Status:     http.StatusText(http.StatusInternalServerError),
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				rOK := &http.Response{
					Status:     http.StatusText(http.StatusOK),
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}

				mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("network error"))
				mock.EXPECT().Do(gomock.Any()).Return(rErr, nil)
				mock.EXPECT().Do(gomock.Any()).Return(rOK, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "fail all attempts",
			setupMocks: func(mock *MockHTTPClient) {
				rErr := &http.Response{
					Status:     http.StatusText(http.StatusInternalServerError),
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}

				mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("network error"))
				mock.EXPECT().Do(gomock.Any()).Return(rErr, nil).Times(3)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "request context timeout",
			setupMocks: func(mock *MockHTTPClient) {
				rErr := &http.Response{
					Status:     http.StatusText(http.StatusInternalServerError),
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}

				mock.EXPECT().Do(gomock.Any()).DoAndReturn(func(_ *http.Request) (*http.Response, error) {
					time.Sleep(500 * time.Millisecond)
					return rErr, nil
				})
			},
			ctxTimeout: 100 * time.Millisecond,
			wantErr:    true,
		},
		{
			name:             "request body error",
			requestBodyError: true,
			setupMocks: func(mock *MockHTTPClient) {
				rErr := &http.Response{
					Status:     http.StatusText(http.StatusInternalServerError),
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				mock.EXPECT().Do(gomock.Any()).Return(rErr, nil)
			},
			wantErr: true,
		},
		{
			name: "close error",
			setupMocks: func(mock *MockHTTPClient) {
				rErr := &http.Response{
					Status:     http.StatusText(http.StatusInternalServerError),
					StatusCode: http.StatusInternalServerError,
					Body:       testutil.NewErrorCloser("close error"),
				}
				mock.EXPECT().Do(gomock.Any()).Return(rErr, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP := NewMockHTTPClient(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(mockHTTP)
			}

			ctx := t.Context()

			if tt.ctxTimeout > 0 {
				timeoutCtx, cancel := context.WithTimeout(t.Context(), tt.ctxTimeout)

				defer cancel()

				ctx = timeoutCtx
			}

			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", tt.body)
			require.NoError(t, err)

			if tt.requestBodyError {
				r.GetBody = func() (io.ReadCloser, error) {
					return nil, errors.New("ERROR")
				}
			}

			opts := []Option{
				WithRetryIfFn(RetryIfForReadRequests),
				WithAttempts(4),
				WithDelay(100 * time.Millisecond),
				WithDelayFactor(1.2),
				WithJitter(50 * time.Millisecond),
			}

			retrier, err := New(mockHTTP, opts...)
			require.NoError(t, err)

			resp, err := retrier.Do(r)
			if resp != nil {
				_ = resp.Body.Close()
			}

			require.Equal(t, tt.wantErr, err != nil, "Do() error = %v, wantErr %v", err, tt.wantErr)

			if tt.wantStatus != 0 {
				require.NotNil(t, resp, "Do() response should not be nil")
				require.Equal(t, tt.wantStatus, resp.StatusCode, "Do() status = %v, wantStatus %v", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestHTTPRetrier_setTimer(t *testing.T) {
	t.Parallel()

	s := &doState{
		timer: time.NewTimer(1 * time.Millisecond),
	}

	time.Sleep(2 * time.Millisecond)
	s.setTimer(2 * time.Millisecond)

	<-s.timer.C
}

// flakyClient is a thread-safe [HTTPClient] used by the concurrency test. It
// fails the first call of each goroutine with a transient error, then succeeds.
type flakyClient struct {
	mu       sync.Mutex
	failNext map[*http.Request]bool
}

func newFlakyClient() *flakyClient {
	return &flakyClient{failNext: make(map[*http.Request]bool)}
}

func (f *flakyClient) Do(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.failNext[req] {
		f.failNext[req] = true

		return nil, errors.New("transient network error")
	}

	return &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}, nil
}

// TestHTTPRetrier_Do_concurrent verifies that a single retrier instance shared
// across goroutines handles overlapping Do calls without data races.
func TestHTTPRetrier_Do_concurrent(t *testing.T) {
	t.Parallel()

	retrier, err := New(
		newFlakyClient(),
		WithRetryIfFn(RetryIfForReadRequests),
		WithAttempts(3),
		WithDelay(1*time.Millisecond),
		WithDelayFactor(2),
		WithJitter(1*time.Millisecond),
	)
	require.NoError(t, err)

	const goroutines = 16

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", bytes.NewReader([]byte(`body`)))
			assert.NoError(t, rerr)

			resp, derr := retrier.Do(r)
			assert.NoError(t, derr)

			if resp != nil {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.NoError(t, resp.Body.Close())
			}
		}()
	}

	wg.Wait()
}

// TestHTTPRetrier_Do_bodyReplayOnRetry verifies the next-attempt body is opened
// lazily from GetBody and reused across retries.
func TestHTTPRetrier_Do_bodyReplayOnRetry(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	rErr := &http.Response{
		Status:     http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}
	rOK := &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}

	mockHTTP.EXPECT().Do(gomock.Any()).Return(rErr, nil)
	mockHTTP.EXPECT().Do(gomock.Any()).Return(rOK, nil)

	var getBodyCalls int

	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", bytes.NewReader([]byte(`payload`)))
	require.NoError(t, err)

	origGetBody := r.GetBody
	r.GetBody = func() (io.ReadCloser, error) {
		getBodyCalls++

		return origGetBody()
	}

	retrier, err := New(
		mockHTTP,
		WithRetryIfFn(RetryIfForReadRequests),
		WithAttempts(3),
		WithDelay(1*time.Millisecond),
		WithJitter(1*time.Millisecond),
	)
	require.NoError(t, err)

	resp, err := retrier.Do(r)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, 1, getBodyCalls, "GetBody should be opened exactly once, only for the retry")
}

// TestHTTPRetrier_Do_bodyNotReplayable verifies that when a retry is required
// but the request body has already been consumed and GetBody is nil, Do stops
// retrying and returns the documented error instead of resending a consumed
// body.
func TestHTTPRetrier_Do_bodyNotReplayable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	rErr := &http.Response{
		Status:     http.StatusText(http.StatusServiceUnavailable),
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}

	// Only one attempt must be performed: the retry is aborted because the
	// consumed body cannot be recreated.
	mockHTTP.EXPECT().Do(gomock.Any()).Return(rErr, nil)

	// A streaming (non-replayable) body: net/http does not auto-set GetBody
	// for reader types it does not recognize.
	body := struct{ io.Reader }{bytes.NewReader([]byte(`streaming payload`))}

	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/", body)
	require.NoError(t, err)
	require.Nil(t, r.GetBody)

	retrier, err := New(
		mockHTTP,
		WithRetryIfFn(RetryIfForWriteRequests),
		WithAttempts(3),
		WithDelay(1*time.Millisecond),
		WithJitter(1*time.Millisecond),
	)
	require.NoError(t, err)

	resp, err := retrier.Do(r) //nolint:bodyclose // resp is nil on this error path; nothing to close.
	require.Nil(t, resp, "the retriable response body has been closed, so no response is returned")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBodyNotReplayable)
}

// TestHTTPRetrier_Do_contextCancelStopsTimer verifies that when the request
// context is canceled mid-flight, Do returns an error and the per-call timer is
// stopped (no leak, no panic).
func TestHTTPRetrier_Do_contextCancelStopsTimer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	mockHTTP.EXPECT().Do(gomock.Any()).DoAndReturn(func(_ *http.Request) (*http.Response, error) {
		time.Sleep(200 * time.Millisecond)

		return &http.Response{
			Status:     http.StatusText(http.StatusInternalServerError),
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
		}, nil
	}).AnyTimes()

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	require.NoError(t, err)

	retrier, err := New(
		mockHTTP,
		WithRetryIfFn(RetryIfForReadRequests),
		WithAttempts(5),
		WithDelay(10*time.Millisecond),
		WithJitter(1*time.Millisecond),
	)
	require.NoError(t, err)

	resp, err := retrier.Do(r)
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.Error(t, err)
	require.ErrorContains(t, err, "request context has been canceled")
}

// TestHTTPRetrier_nextRetryDelay verifies the exponential backoff progression
// and that the computed delay is clamped to the configured maximum.
func TestHTTPRetrier_nextRetryDelay(t *testing.T) {
	t.Parallel()

	retrier, err := New(
		http.DefaultClient,
		WithDelay(100*time.Millisecond),
		WithDelayFactor(2),
		WithJitter(1), // 1ns ceiling: rand.Int63n(1) is always 0, so no jitter noise
		WithMaxDelay(350*time.Millisecond),
	)
	require.NoError(t, err)

	s := &doState{nextDelay: float64(retrier.delay)}

	// 100ms -> not clamped
	d1 := retrier.nextRetryDelay(s)
	require.Equal(t, 100*time.Millisecond, d1)

	// 200ms -> not clamped
	d2 := retrier.nextRetryDelay(s)
	require.Equal(t, 200*time.Millisecond, d2)

	// 400ms computed, clamped to 350ms max
	d3 := retrier.nextRetryDelay(s)
	require.Equal(t, 350*time.Millisecond, d3)

	// 800ms computed, still clamped to 350ms max
	d4 := retrier.nextRetryDelay(s)
	require.Equal(t, 350*time.Millisecond, d4)
}
