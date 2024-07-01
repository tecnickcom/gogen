//go:generate mockgen -package passwordpwned -destination ./mock_test.go . HTTPClient
package passwordpwned

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/httpretrier"
	"github.com/Vonage/gosrvlib/pkg/httputil"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/undefinedlabs/go-mpatch"
)

//go:noinline
func newRequestWithContextPatch(_ context.Context, _, _ string, _ io.Reader) (*http.Request, error) {
	return nil, errors.New("error")
}

//go:noinline
func newHTTPRetrierPatch(httpretrier.HTTPClient, ...httpretrier.Option) (*httpretrier.HTTPRetrier, error) {
	return nil, errors.New("error")
}

//nolint:gocognit,tparallel
func TestClient_IsPwnedPassword(t *testing.T) {
	t.Parallel()

	// pwned.password.1 : AC8A89B5F24DE5F1D9AE8499A204B5098B08DF1B
	// pwned.password.2 : 274AC46FA9F7FDDB8AB4A5BB8295A47E3929171E
	// pwned.password.3 : C1C39EBC8981022DC3220FF6C17D1933BA5E5061
	// pwned.password.4 : 05955AE4E6ADFB93265CA2BCF0560529CF0BFDC9
	// pwned.password.5 : 34D03CE275F04C48AF10A4E23AB85D27AF3239B0
	// pwned.password.6 : ACE9846F1DC7F76EB2E5D064BDEFE65B712F85D3

	// body:
	// 9B5F24DE5F1D9AE8499A204B5098B08DF1B:1
	// 46FA9F7FDDB8AB4A5BB8295A47E3929171E:2
	// EBC8981022DC3220FF6C17D1933BA5E5061:3
	// AE4E6ADFB93265CA2BCF0560529CF0BFDC9:4
	// CE275F04C48AF10A4E23AB85D27AF3239B0:0
	// 46F1DC7F76EB2E5D064BDEFE65B712F85D3:0

	// base64 brotli encoded body
	retBody, _ := base64.StdEncoding.DecodeString("G+IA+I2ULm8UPTY2L7T4yFAoTZILH26i9Ehm9XAi90lEEkgpxCt4c1gfxS7j/GbqZUlq1aPQFF8OCnTcT1v94iEQTTMR3FmjDwZzpa6C4edcWcu5CibTTqo+UAOl6IO66fjSS64H0vLyEFKWOOvpkOcxcRR8EDuc9nPbotUBk5q9NS1HOvwB")

	mockHandleFn := func(t *testing.T, body []byte) http.HandlerFunc {
		t.Helper()

		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", httputil.MimeTextPlain)
			w.Header().Set("Content-Encoding", "br")
			_, err := w.Write(body)
			assert.NoError(t, err)
		}
	}

	tests := []struct {
		name              string
		password          string
		createMockHandler func(t *testing.T) http.HandlerFunc
		setupMocks        func(client *MockHTTPClient)
		setupPatches      func() (*mpatch.Patch, error)
		hashError         bool
		pwned             bool
		wantErr           bool
	}{
		{
			name:      "hash write error",
			password:  "some.password",
			hashError: true,
			pwned:     false,
			wantErr:   true,
		},
		{
			name: "failed to execute request - NewRequest error",
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
		{
			name: "failed to execute request - HTTPRetrier error",
			setupPatches: func() (*mpatch.Patch, error) {
				patch, err := mpatch.PatchMethod(httpretrier.New, newHTTPRetrierPatch)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				_ = patch.Patch()
				return patch, nil
			},
			wantErr: true,
		},
		{
			name: "failed to execute request - transport error",
			setupMocks: func(m *MockHTTPClient) {
				m.EXPECT().Do(gomock.Any()).Return(nil, errors.New("transport error")).Times(1)
			},
			wantErr: true,
		},
		{
			name: "unexpected http error status code",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return func(w http.ResponseWriter, r *http.Request) {
					httputil.SendStatus(r.Context(), w, http.StatusInternalServerError)
				}
			},
			wantErr: true,
		},
		{
			name: "invalid response status < 200",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return func(w http.ResponseWriter, r *http.Request) {
					httputil.SendStatus(r.Context(), w, http.StatusSwitchingProtocols)
				}
			},
			wantErr: true,
		},
		{
			name:     "invalid brotli encoding",
			password: "pwned.password.1", // AC8A89B5F24DE5F1D9AE8499A204B5098B08DF1B
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return mockHandleFn(t, []byte("invalid"))
			},
			pwned:   false,
			wantErr: true,
		},
		{
			name:     "pwned password",
			password: "pwned.password.2", // 274AC46FA9F7FDDB8AB4A5BB8295A47E3929171E
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return mockHandleFn(t, retBody)
			},
			pwned:   true,
			wantErr: false,
		},
		{
			name:     "false pwned password because of padding",
			password: "pwned.password.6", // ACE9846F1DC7F76EB2E5D064BDEFE65B712F85D3
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return mockHandleFn(t, retBody)
			},
			pwned:   false,
			wantErr: false,
		},
		{
			name:     "ok password",
			password: "not.pwned.password",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return mockHandleFn(t, retBody)
			},
			pwned:   false,
			wantErr: false,
		},
	}

	//nolint:paralleltest
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
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
				WithRetryAttempts(1),
			}

			if tt.setupMocks != nil {
				mc := NewMockHTTPClient(ctrl)
				tt.setupMocks(mc)
				clientOpts = append(clientOpts, WithHTTPClient(mc), WithRetryAttempts(1))
			}

			c, err := New(clientOpts...)
			require.NoError(t, err)

			if tt.hashError {
				c.hashObj = &mockHashErr{}
			}

			if tt.setupPatches != nil {
				patch, err := tt.setupPatches()
				require.NoError(t, err)

				defer func() {
					_ = patch.Unpatch()
				}()
			}

			got, err := c.IsPwnedPassword(testutil.Context(), tt.password)

			require.Equal(t, tt.wantErr, err != nil, err)
			require.Equal(t, tt.pwned, got)
		})
	}
}

type mockWriterErr struct{}

func (w *mockWriterErr) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

type mockHashErr struct {
	mockWriterErr
}

func (m *mockHashErr) Sum(b []byte) []byte {
	return b
}

func (m *mockHashErr) Reset() {}

func (m *mockHashErr) Size() int {
	return 0
}

func (m *mockHashErr) BlockSize() int {
	return 0
}
