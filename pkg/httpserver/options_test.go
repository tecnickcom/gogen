package httpserver

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
	"github.com/stretchr/testify/require"
)

func TestWithRouter(t *testing.T) {
	t.Parallel()

	v := httprouter.New()
	cfg := &config{}
	err := WithRouter(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.router).Pointer())
}

func TestWithServerAddr(t *testing.T) {
	t.Parallel()

	v := ":1234"
	cfg := &config{}
	err := WithServerAddr(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, v, cfg.serverAddr)
}

func TestWithServerReadTimeout(t *testing.T) {
	t.Parallel()

	v := 13 * time.Second
	cfg := &config{}
	err := WithServerReadTimeout(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, v, cfg.serverReadTimeout)
}

func TestWithServerWriteTimeout(t *testing.T) {
	t.Parallel()

	v := 17 * time.Second
	cfg := &config{}
	err := WithServerWriteTimeout(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, v, cfg.serverWriteTimeout)
}

func TestWithShutdownTimeout(t *testing.T) {
	t.Parallel()

	v := 19 * time.Second
	cfg := &config{}
	err := WithShutdownTimeout(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, v, cfg.shutdownTimeout)
}

func TestWithTLSCertData(t *testing.T) {
	tests := []struct {
		desc     string
		certData []byte
		keyData  []byte
		wantErr  bool
	}{
		{
			desc:     "should fail with invalid certificate data",
			certData: []byte(""),
			keyData:  []byte(""),
			wantErr:  true,
		},
		{
			desc: "should succeed with valid certificate data",
			certData: []byte(`-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`),
			keyData: []byte(`-----BEGIN PRIVATE KEY-----
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
-----END PRIVATE KEY-----`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			cfg := &config{}
			err := WithTLSCertData(tt.certData, tt.keyData)(cfg)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, cfg.tlsConfig)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg.tlsConfig)
			}
		})
	}
}

func TestWithEnableDefaultRoutes(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	err := WithEnableDefaultRoutes(IndexRoute, MetricsRoute)(cfg)
	require.NoError(t, err)
	require.Equal(t, []defaultRoute{IndexRoute, MetricsRoute}, cfg.defaultEnabledRoutes)
}

func TestWithEnableAllDefaultRoutes(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	err := WithEnableAllDefaultRoutes()(cfg)
	require.NoError(t, err)
	require.Equal(t, allDefaultRoutes, cfg.defaultEnabledRoutes)
}

func TestWithIndexHandlerFunc(t *testing.T) {
	t.Parallel()

	v := func(routes []route.Route) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// mock function
		}
	}
	cfg := &config{}
	err := WithIndexHandlerFunc(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.indexHandlerFunc).Pointer())
}

func TestWithIPHandlerFunc(t *testing.T) {
	t.Parallel()

	v := func(_ http.ResponseWriter, _ *http.Request) {
		// mock function
	}
	cfg := &config{}
	err := WithIPHandlerFunc(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.ipHandlerFunc).Pointer())
}

func TestWithMetricsHandlerFunc(t *testing.T) {
	t.Parallel()

	v := func(_ http.ResponseWriter, _ *http.Request) {
		// mock function
	}
	cfg := &config{}
	err := WithMetricsHandlerFunc(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.metricsHandlerFunc).Pointer())
}

func TestWithPingHandlerFunc(t *testing.T) {
	t.Parallel()

	v := func(_ http.ResponseWriter, _ *http.Request) {
		// mock function
	}
	cfg := &config{}
	err := WithPingHandlerFunc(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.pingHandlerFunc).Pointer())
}

func TestWithPProfHandlerFunc(t *testing.T) {
	t.Parallel()

	v := func(_ http.ResponseWriter, _ *http.Request) {
		// mock function
	}
	cfg := &config{}
	err := WithPProfHandlerFunc(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.pprofHandlerFunc).Pointer())
}

func TestWithStatusHandlerFunc(t *testing.T) {
	t.Parallel()

	v := func(_ http.ResponseWriter, _ *http.Request) {
		// mock function
	}
	cfg := &config{}
	err := WithStatusHandlerFunc(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.statusHandlerFunc).Pointer())
}

func TestWithTraceIDHeaderName(t *testing.T) {
	t.Parallel()

	v := "X-Test-Header"
	cfg := &config{}
	err := WithTraceIDHeaderName(v)(cfg)
	require.NoError(t, err)
	require.Equal(t, v, cfg.traceIDHeaderName)
}
