package passwordpwned

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/undefinedlabs/go-mpatch"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      []Option
		wantErrIs error
		wantErr   bool
	}{
		{
			name:      "fails with invalid character in URL",
			opts:      []Option{WithURL("http://invalid-url.domain.invalid\u007F")},
			wantErrIs: ErrInvalidURL,
			wantErr:   true,
		},
		{
			name:      "fails with missing scheme",
			opts:      []Option{WithURL("//missing-scheme.invalid")},
			wantErrIs: ErrInvalidURL,
			wantErr:   true,
		},
		{
			name:      "fails with missing host",
			opts:      []Option{WithURL("https://")},
			wantErrIs: ErrInvalidURL,
			wantErr:   true,
		},
		{
			name:      "fails with URL query",
			opts:      []Option{WithURL("https://test.invalid?token=x")},
			wantErrIs: ErrInvalidURL,
			wantErr:   true,
		},
		{
			name:      "fails with URL fragment",
			opts:      []Option{WithURL("https://test.invalid#frag")},
			wantErrIs: ErrInvalidURL,
			wantErr:   true,
		},
		{
			name:      "fails with empty user agent",
			opts:      []Option{WithUserAgent("")},
			wantErrIs: ErrInvalidUserAgent,
			wantErr:   true,
		},
		{
			name:      "fails with control characters in user agent",
			opts:      []Option{WithUserAgent("evil/1\r\nX-Injected: true")},
			wantErrIs: ErrInvalidUserAgent,
			wantErr:   true,
		},
		{
			name:    "fails with zero retry attempts",
			opts:    []Option{WithRetryAttempts(0)},
			wantErr: true,
		},
		{
			name:    "fails with zero retry delay",
			opts:    []Option{WithRetryDelay(0)},
			wantErr: true,
		},
		{
			name:    "fails with negative retry delay",
			opts:    []Option{WithRetryDelay(-1 * time.Second)},
			wantErr: true,
		},
		{
			name:    "succeeds with defaults",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(tt.opts...)

			if tt.wantErr {
				require.Nil(t, c, "New() returned client should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}

				return
			}

			require.NotNil(t, c, "New() returned client should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

//nolint:paralleltest,gocognit // TestClient_IsPwnedPassword mpatch-es process-global functions in the parallel phase; this test must stay in the sequential phase.
func TestClient_HealthCheck(t *testing.T) {
	tests := []struct {
		name              string
		createMockHandler func(t *testing.T) http.HandlerFunc
		setupMocks        func(m *MockHTTPClient)
		setupPatches      func() (*mpatch.Patch, error)
		wantErr           bool
		wantErrIs         error
	}{
		{
			name: "healthy endpoint",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()

				return func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/"+rangePath+"/"+healthCheckPrefix, r.URL.Path)
					assert.Equal(t, defaultUserAgent, r.Header.Get("User-Agent"))
					assert.Empty(t, r.Header.Get("Add-Padding")) // the probe skips padding
					w.WriteHeader(http.StatusOK)
				}
			},
			wantErr: false,
		},
		{
			name: "unexpected status code",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()

				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
			wantErr:   true,
			wantErrIs: ErrUnexpectedStatus,
		},
		{
			name: "transport error",
			setupMocks: func(m *MockHTTPClient) {
				m.EXPECT().Do(gomock.Any()).Return(nil, errors.New("transport error")).Times(1)
			},
			wantErr: true,
		},
		{
			name: "nil response body from non-conforming client is still a valid status signal",
			setupMocks: func(m *MockHTTPClient) {
				m.EXPECT().Do(gomock.Any()).Return(&http.Response{StatusCode: http.StatusOK}, nil).Times(1)
			},
			wantErr: false,
		},
		{
			name: "create request error",
			setupPatches: func() (*mpatch.Patch, error) {
				patch, err := mpatch.PatchMethod(http.NewRequestWithContext, newRequestWithContextPatch)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				_ = patch.Patch()

				return patch, nil
			},
			wantErr: true,
		},
	}

	//nolint:paralleltest // parent stays in the sequential phase (see above), so subtests must not be parallel.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip mpatch-based testcases on Apple Silicon macOS
			if tt.setupPatches != nil && runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
				t.Skip("mpatch not supported on Mac silicon - skipping test")
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mux := http.NewServeMux()
			if tt.createMockHandler != nil {
				mux.HandleFunc("/"+rangePath+"/", tt.createMockHandler(t))
			}

			ts := httptest.NewServer(mux)
			defer ts.Close()

			clientOpts := []Option{
				WithURL(ts.URL),
				WithPingTimeout(time.Second),
			}

			if tt.setupMocks != nil {
				mc := NewMockHTTPClient(ctrl)
				tt.setupMocks(mc)
				clientOpts = append(clientOpts, WithHTTPClient(mc))
			}

			c, err := New(clientOpts...)
			require.NoError(t, err)

			if tt.setupPatches != nil {
				patch, perr := tt.setupPatches()
				require.NoError(t, perr)

				defer func() {
					_ = patch.Unpatch()
				}()
			}

			err = c.HealthCheck(t.Context())
			require.Equal(t, tt.wantErr, err != nil, err)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			}
		})
	}
}
