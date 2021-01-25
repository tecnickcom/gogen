//go:generate mockgen -package httpserver -destination ./mock_test.go . Router,Binder

package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/nexmoinc/gosrvlib/pkg/traceid"
	"github.com/stretchr/testify/require"
)

func TestNopBinder(t *testing.T) {
	require.NotNil(t, NopBinder())
}

func Test_nopBinder_BindHTTP(t *testing.T) {
	require.Nil(t, NopBinder().BindHTTP(context.Background()))
}

func Test_defaultRouter(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		setupRouter func(Router)
		wantStatus  int
	}{
		{
			name:       "should handle 404",
			method:     http.MethodGet,
			path:       "/not/found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "should handle 405",
			method: http.MethodPost,
			setupRouter: func(r Router) {
				fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
				})
				r.Handler(http.MethodGet, "/not/allowed", fn)
			},
			path:       "/not/allowed",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "should handle panic in handler",
			method: http.MethodGet,
			setupRouter: func(r Router) {
				fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					panic("panicking!")
				})
				r.Handler(http.MethodGet, "/panic", fn)
			},
			path:       "/panic",
			wantStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defaultInstrumentHandler := func(path string, handler http.HandlerFunc) http.Handler { return handler }
			r := defaultRouter(testutil.Context(), traceid.DefaultHeader, defaultInstrumentHandler)

			if tt.setupRouter != nil {
				tt.setupRouter(r)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, httptest.NewRequest(tt.method, tt.path, nil))

			resp := rr.Result()
			require.Equal(t, tt.wantStatus, resp.StatusCode, "status code got = %d, want = %d", resp.StatusCode, tt.wantStatus)
		})
	}
}

func Test_defaultIndexHandler(t *testing.T) {
	routes := []route.Route{
		{
			Method:      "GET",
			Path:        "/get",
			Handler:     nil,
			Description: "Get endpoint",
		},
		{
			Method:      "POST",
			Path:        "/post",
			Handler:     nil,
			Description: "Post endpoint",
		},
	}
	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	defaultIndexHandler(routes).ServeHTTP(rr, req)

	resp := rr.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	expBody, _ := json.Marshal(&route.Index{Routes: routes})

	require.Equal(t, string(expBody)+"\n", string(body))
}

func Test_defaultIPHandler(t *testing.T) {
	tests := []struct {
		name    string
		ipFunc  GetPublicIPFunc
		wantIP  string
		wantErr bool
	}{
		{
			name:    "success response",
			ipFunc:  func(ctx context.Context) (string, error) { return "0.0.0.0", nil },
			wantIP:  "0.0.0.0",
			wantErr: false,
		},
		{
			name:    "error response",
			ipFunc:  func(ctx context.Context) (string, error) { return "ERROR", fmt.Errorf("ERROR") },
			wantIP:  "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
			defaultIPHandler(tt.ipFunc).ServeHTTP(rr, req)

			resp := rr.Result()
			bodyData, _ := ioutil.ReadAll(resp.Body)
			body := string(bodyData)

			require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))

			if tt.wantErr {
				require.Equal(t, http.StatusFailedDependency, resp.StatusCode)
				require.Equal(t, "ERROR", body)
			} else {
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, "0.0.0.0", body)
			}
		})
	}
}

func Test_defaultPingHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	defaultPingHandler(rr, req)

	resp := rr.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "OK\n", string(body))
}

func Test_defaultStatusHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	defaultStatusHandler(rr, req)

	resp := rr.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "OK\n", string(body))
}

// nolint:gocognit
func TestStart(t *testing.T) {
	tests := []struct {
		name           string
		opts           []Option
		failListenPort int
		setupBinder    func(*MockBinder)
		setupRouter    func(*MockRouter)
		wantErr        bool
	}{
		{
			name: "fail with invalid config",
			opts: []Option{
				WithTraceIDHeaderName(""),
			},
			wantErr: true,
		},
		{
			name: "fail with option error",
			opts: []Option{
				WithTLSCertData([]byte(``), []byte(``)),
			},
			wantErr: true,
		},
		{
			name: "fail listen port already bound",
			opts: []Option{
				WithServerAddr(":12345"),
				WithShutdownTimeout(1 * time.Millisecond),
			},
			setupBinder: func(b *MockBinder) {
				b.EXPECT().BindHTTP(gomock.Any()).Times(1)
			},
			setupRouter: func(r *MockRouter) {
				r.EXPECT().Handler(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			failListenPort: 12345,
			wantErr:        true,
		},
		{
			name: "succeed",
			opts: []Option{
				WithServerAddr(":11111"),
				WithShutdownTimeout(1 * time.Millisecond),
				WithEnableAllDefaultRoutes(),
			},
			setupBinder: func(b *MockBinder) {
				b.EXPECT().BindHTTP(gomock.Any()).Times(1)
			},
			setupRouter: func(r *MockRouter) {
				r.EXPECT().Handler(gomock.Any(), gomock.Any(), gomock.Any()).Times(6)
			},
			wantErr: false,
		},
		{
			name: "succeed w/ TLS",
			opts: []Option{
				WithTLSCertData([]byte(`-----BEGIN CERTIFICATE-----
MIICBjCCAW8CFB9PJprToZgFfDJpt3Qk6JIEaMEEMA0GCSqGSIb3DQEBCwUAMEIx
CzAJBgNVBAYTAlhYMRUwEwYDVQQHDAxEZWZhdWx0IENpdHkxHDAaBgNVBAoME0Rl
ZmF1bHQgQ29tcGFueSBMdGQwHhcNMjAwNzIyMTMyMTExWhcNMzAwNzIwMTMyMTEx
WjBCMQswCQYDVQQGEwJYWDEVMBMGA1UEBwwMRGVmYXVsdCBDaXR5MRwwGgYDVQQK
DBNEZWZhdWx0IENvbXBhbnkgTHRkMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQDTHo34VDfPXuDR4mDPpfh8hvja8loIB60b/qvv81TnJEyjLRzaI4dXclFZwUWC
zWi6LxgVcpILMG4n2KieK4h22EsaQZ7ncZ6pLTHlNJfQXWcHzUmwbA1CNyxJN72Q
LLLE3yw8Xm5AM4QegPJQ3+I27GTnAocygqVKX+aU8rUdgQIDAQABMA0GCSqGSIb3
DQEBCwUAA4GBAE3CSgcBH2P2Y0vvjyijavSCIyvau3ex1cmmybZBDen9aGhw34X5
iotTHm8vUEMtinenht11ypQhxefAreTg0RjsZuCzHlgOQrUIpY5qNSTBNTChbU/b
V6QQpxzrYshYcFuiGxSAdZMa8AFVB4Wan7Ji+vvDTJOyXbDqxA3kLFLi
-----END CERTIFICATE-----`), []byte(`-----BEGIN PRIVATE KEY-----
MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBANMejfhUN89e4NHi
YM+l+HyG+NryWggHrRv+q+/zVOckTKMtHNojh1dyUVnBRYLNaLovGBVykgswbifY
qJ4riHbYSxpBnudxnqktMeU0l9BdZwfNSbBsDUI3LEk3vZAsssTfLDxebkAzhB6A
8lDf4jbsZOcChzKCpUpf5pTytR2BAgMBAAECgYBPSNZAQECFXDhKGh4JXWcoPPgQ
IZu2EEvui4G+pz9nXrZ5QWPoeBdHu+LZNkAIk2OVKEJ/K3u1QAbeZ/tLC0Y/zGmS
Nv0wgCQ+A4FfQH6l5Hh3jrxFDgbjv+Lrb3Np52AC/NIU0DamNK0VffM/kZpj6Gl0
6uUtqwZwh57rJXjMkQJBAPub3EyG1p3/2CEMm2B7jmn5S+qXKgNdA681mvHY2Q6u
hhtIVtKgEV/yTvx4U6JqD1EAm8MpjfqcGHKqXIqJLn8CQQDWzct+hh5AXrirSz7o
j4WxtWuYRDr+2BWFRee0s5CaWy0y7L3fOv+RwbfFSmBgsGPSq+zXKcvOGU0S5Oca
87P/AkAxinbN+p63bXC40SqmzK014Ig6IJl9IAthrERd6jySz3pIVO4DetDw+1zi
CS8ug4OQh3Yj70KtXZ7StQiTnn8xAkBgE4I+YDytq/BLZYeIu5Ef8DZkz7fXfsz5
ZFAD6gD2mWt5CJzQePIQvqW0z9SVyq+Lbiyr/FzVHUn09n9L9c7/AkA1VDTPiY/H
DSk+QcX0L58Fc7RiaBnykcJRfHnd15MlyqtUJ02iitNJOoSVBNQzr59Iyt7nGBzm
YlAqGKDZ+A+l
-----END PRIVATE KEY-----`)),
				WithServerAddr(":22222"),
				WithShutdownTimeout(1 * time.Millisecond),
				WithEnableAllDefaultRoutes(),
			},
			setupBinder: func(b *MockBinder) {
				b.EXPECT().BindHTTP(gomock.Any()).Times(1)
			},
			setupRouter: func(r *MockRouter) {
				r.EXPECT().Handler(gomock.Any(), gomock.Any(), gomock.Any()).Times(6)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockBinder := NewMockBinder(mockCtrl)
			if tt.setupBinder != nil {
				tt.setupBinder(mockBinder)
			}

			ctx, cancelCtx := context.WithCancel(testutil.Context())
			defer func() {
				cancelCtx()
				time.Sleep(100 * time.Millisecond)
			}()
			opts := tt.opts

			mockRouter := NewMockRouter(mockCtrl)
			if tt.setupRouter != nil {
				tt.setupRouter(mockRouter)
				opts = append(opts, WithRouter(mockRouter))
			}

			if tt.failListenPort != 0 {
				l, err := net.Listen("tcp", fmt.Sprintf(":%d", tt.failListenPort))
				require.NoError(t, err, "failed starting pre-listener")
				defer func() { _ = l.Close() }()
			}
			err := Start(ctx, mockBinder, opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
