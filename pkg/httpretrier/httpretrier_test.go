//go:generate mockgen -package httpretrier -destination ./mock_test.go . HTTPClient
package httpretrier

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
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
				WithRetryIfFn(func(r *http.Response, err error) bool { return true }),
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
		tt := tt
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
			err:  fmt.Errorf("ERROR"),
			want: true,
		},
		{
			name: "false with no error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
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
			err:    fmt.Errorf("ERROR"),
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
		tt := tt
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
			err:    fmt.Errorf("ERROR"),
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
		tt := tt
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

//nolint:gocognit
func TestHTTPRetrier_Do(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		setupMocks            func(mock *MockHTTPClient)
		ctxTimeout            time.Duration
		body                  io.Reader
		wantRemainingAttempts uint
		wantErr               bool
		requestBodyError      bool
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
			wantRemainingAttempts: 3,
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
			body:                  bytes.NewReader([]byte(`some body`)),
			wantRemainingAttempts: 3,
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
				mock.EXPECT().Do(gomock.Any()).Return(nil, fmt.Errorf("network error"))
				mock.EXPECT().Do(gomock.Any()).Return(rErr, nil)
				mock.EXPECT().Do(gomock.Any()).Return(rOK, nil)
			},
			wantRemainingAttempts: 1,
		},
		{
			name: "fail all attempts",
			setupMocks: func(mock *MockHTTPClient) {
				rErr := &http.Response{
					Status:     http.StatusText(http.StatusInternalServerError),
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				mock.EXPECT().Do(gomock.Any()).Return(nil, fmt.Errorf("network error"))
				mock.EXPECT().Do(gomock.Any()).Return(rErr, nil).Times(3)
			},
			wantRemainingAttempts: 0,
		},
		{
			name: "request context timeout",
			setupMocks: func(mock *MockHTTPClient) {
				rErr := &http.Response{
					Status:     http.StatusText(http.StatusInternalServerError),
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				mock.EXPECT().Do(gomock.Any()).DoAndReturn(func(r *http.Request) (*http.Response, error) {
					time.Sleep(500 * time.Millisecond)
					return rErr, nil
				})
			},
			ctxTimeout:            100 * time.Millisecond,
			wantRemainingAttempts: 3,
			wantErr:               true,
		},
		{
			name:                  "request body error",
			requestBodyError:      true,
			ctxTimeout:            100 * time.Millisecond,
			wantRemainingAttempts: DefaultAttempts,
			wantErr:               true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP := NewMockHTTPClient(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(mockHTTP)
			}

			ctx := testutil.Context()
			if tt.ctxTimeout > 0 {
				timeoutCtx, cancel := context.WithTimeout(testutil.Context(), tt.ctxTimeout)
				defer cancel()

				ctx = timeoutCtx
			}

			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", tt.body)
			require.NoError(t, err)

			if tt.requestBodyError {
				r.GetBody = func() (io.ReadCloser, error) { return nil, fmt.Errorf("ERROR") }
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
			require.Equal(t, tt.wantRemainingAttempts, retrier.remainingAttempts, "Do() remainingAttempts = %v, wantRemainingAttempts %v", err, tt.wantErr)
		})
	}
}

func TestHTTPRetrier_setTimer(t *testing.T) {
	t.Parallel()

	c := &HTTPRetrier{
		timer: time.NewTimer(1 * time.Millisecond),
	}

	time.Sleep(2 * time.Millisecond)
	c.setTimer(2 * time.Millisecond)
	<-c.timer.C
}
