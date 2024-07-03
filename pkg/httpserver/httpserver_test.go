//go:generate mockgen -package httpserver -destination ./mock_test.go . Binder

package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNopBinder(t *testing.T) {
	t.Parallel()
	require.NotNil(t, NopBinder())
}

func Test_nopBinder_BindHTTP(t *testing.T) {
	t.Parallel()
	require.Nil(t, NopBinder().BindHTTP(context.Background()))
}

func Test_defaultIndexHandler(t *testing.T) {
	t.Parallel()

	routes := []Route{
		{
			Method:      http.MethodGet,
			Path:        "/get",
			Handler:     nil,
			Description: "Get endpoint",
		},
		{
			Method:      http.MethodPost,
			Path:        "/post",
			Handler:     nil,
			Description: "Post endpoint",
		},
	}
	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	defaultIndexHandler(routes).ServeHTTP(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	expBody, err := json.Marshal(&Index{Routes: routes})
	require.NoError(t, err)

	require.Equal(t, string(expBody)+"\n", string(body))
}

func Test_defaultIPHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ipFunc  GetPublicIPFunc
		wantIP  string
		wantErr bool
	}{
		{
			name:    "success response",
			ipFunc:  func(_ context.Context) (string, error) { return "0.0.0.0", nil },
			wantIP:  "0.0.0.0",
			wantErr: false,
		},
		{
			name:    "error response",
			ipFunc:  func(_ context.Context) (string, error) { return "ERROR", errors.New("ERROR") },
			wantIP:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
			defaultIPHandler(tt.ipFunc).ServeHTTP(rr, req)

			resp := rr.Result()
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			bodyData, _ := io.ReadAll(resp.Body)
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
	t.Parallel()

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	defaultPingHandler(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "OK\n", string(body))
}

func Test_defaultStatusHandler(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	defaultStatusHandler(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "OK\n", string(body))
}

func Test_notImplementedHandler(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(testutil.Context(), http.MethodGet, "/", nil)
	notImplementedHandler(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	require.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}

type customMiddlewareBinder struct {
	firstMiddleware  chan struct{}
	secondMiddleware chan struct{}
}

func (c *customMiddlewareBinder) handler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *customMiddlewareBinder) slowHandler(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(2 * time.Millisecond)
	w.WriteHeader(http.StatusOK)
}

func (c *customMiddlewareBinder) middleware(ch chan struct{}) MiddlewareFn {
	return func(_ MiddlewareArgs, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ch <- struct{}{}

			next.ServeHTTP(w, r)
		})
	}
}

func (c *customMiddlewareBinder) BindHTTP(_ context.Context) []Route {
	return []Route{
		{
			Method:      http.MethodGet,
			Path:        "/hello",
			Description: "Test endpoint",
			Handler:     c.handler,
			Middleware:  []MiddlewareFn{c.middleware(c.firstMiddleware), c.middleware(c.secondMiddleware)},
			Timeout:     10 * time.Second,
		},
		{
			Method:      http.MethodGet,
			Path:        "/timeout",
			Description: "Timeout endpoint",
			Handler:     c.slowHandler,
			Middleware:  []MiddlewareFn{c.middleware(c.firstMiddleware), c.middleware(c.secondMiddleware)},
			Timeout:     1 * time.Millisecond,
		},
	}
}

func Test_customMiddlewares(t *testing.T) {
	t.Parallel()

	binder := &customMiddlewareBinder{
		firstMiddleware:  make(chan struct{}),
		secondMiddleware: make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	l := zap.NewNop()
	cfg := defaultConfig()
	cfg.setRouter(ctx)
	loadRoutes(ctx, l, binder, cfg)

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-binder.firstMiddleware:
		}

		select {
		case <-ctx.Done():
			return
		case <-binder.secondMiddleware:
		}
	}()

	resp := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:1234/hello", nil)
	require.NoError(t, err, "failed to create request")
	cfg.router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, "unexpected response code")
	require.NoError(t, ctx.Err(), "context should not be canceled")

	resp = httptest.NewRecorder()
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:1234/timeout", nil)
	require.NoError(t, err, "failed to create request")
	cfg.router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusServiceUnavailable, resp.Code, "unexpected response code")
	require.NoError(t, ctx.Err(), "context should not be canceled")
}

//nolint:gocognit
func TestStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		opts           []Option
		failListenPort int
		setupBinder    func(*MockBinder)
		shutdownSig    bool
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
			failListenPort: 12345,
			wantErr:        true,
		},
		{
			name: "succeed",
			opts: []Option{
				WithServerAddr(":11111"),
				WithRequestTimeout(1 * time.Minute),
				WithShutdownTimeout(1 * time.Millisecond),
				WithEnableAllDefaultRoutes(),
				WithInstrumentHandler(func(_ string, handler http.HandlerFunc) http.Handler { return handler }),
				WithShutdownTimeout(1 * time.Second),
			},
			setupBinder: func(b *MockBinder) {
				b.EXPECT().BindHTTP(gomock.Any()).Times(1)
			},
			wantErr: false,
		},
		{
			name: "succeed and shutdown with signal",
			opts: []Option{
				WithServerAddr(":11112"),
				WithShutdownTimeout(1 * time.Second),
			},
			setupBinder: func(b *MockBinder) {
				b.EXPECT().BindHTTP(gomock.Any()).Times(1)
			},
			shutdownSig: true,
			wantErr:     false,
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
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockBinder := NewMockBinder(mockCtrl)
			if tt.setupBinder != nil {
				tt.setupBinder(mockBinder)
			}

			opts := tt.opts

			shutdownWG := &sync.WaitGroup{}
			shutdownSG := make(chan struct{})

			opts = append(opts, WithShutdownWaitGroup(shutdownWG))
			opts = append(opts, WithShutdownSignalChan(shutdownSG))

			ctx, cancelCtx := context.WithCancel(testutil.Context())
			defer func() {
				if tt.shutdownSig {
					close(shutdownSG)
				}

				time.Sleep(100 * time.Millisecond)
				cancelCtx()
			}()

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

type mockListenerErr struct{}

func (ls mockListenerErr) Accept() (net.Conn, error) {
	return nil, errors.New("ERROR")
}

func (ls mockListenerErr) Close() error {
	return errors.New("ERROR")
}

func (ls mockListenerErr) Addr() net.Addr {
	return nil
}

func Test_Serve_error(t *testing.T) {
	t.Parallel()

	h := &HTTPServer{
		cfg: defaultConfig(),
		ctx: context.TODO(),
		httpServer: &http.Server{
			Addr:              ":54321",
			ReadHeaderTimeout: 1 * time.Millisecond,
			ReadTimeout:       1 * time.Millisecond,
			WriteTimeout:      1 * time.Millisecond,
		},
		listener: mockListenerErr{},
		logger:   logging.NopLogger(),
	}

	h.serve()
}
