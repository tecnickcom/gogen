package statsd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	statsdTestNetwork = "udp"
	statsdTestAddr    = ":0"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "succeeds with empty options",
			wantErr: false,
		},
		{
			name: "succeeds with custom options",
			opts: []Option{
				WithPrefix("TEST"),
				WithNetwork("udp"),
				WithAddress(":1111"),
				WithFlushPeriod(time.Duration(1) * time.Second),
			},
			wantErr: false,
		},
		{
			name: "unable to dial server",
			opts: []Option{
				WithNetwork("tcp"),
				WithAddress(":65001"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.opts...)
			if tt.wantErr {
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func TestInstrumentHandler(t *testing.T) {
	t.Parallel()

	srv, err := newStatsdServer(t, func(p []byte) {
		exp := "TEST.inbound./test.POST.in:1|c\nTEST.inbound./test.POST.501.count:1|c\nTEST.inbound./test.POST.501.request_size:27|g\nTEST.inbound./test.POST.501.response_size:16|g\nTEST.inbound./test.POST.501.time:0|ms\nTEST.inbound./test.POST.out:1|c"
		require.Equal(t, exp, string(p))
	})
	require.NoError(t, err, "newStatsdServer() unexpected error = %v", err)

	defer srv.Close()

	c, err := New(
		WithPrefix("TEST"),
		WithNetwork(statsdTestNetwork),
		WithAddress(srv.addr),
	)
	require.NoError(t, err, "New() unexpected error = %v", err)

	defer c.Close() // nolint:errcheck

	rr := httptest.NewRecorder()
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/test", strings.NewReader("TEST"))
	require.NoError(t, err, "failed creating http request: %s", err)

	handler := c.InstrumentHandler("/test", c.MetricsHandlerFunc())
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotImplemented {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotImplemented)
	}
}

func TestInstrumentRoundTripper(t *testing.T) {
	t.Parallel()

	srv, err := newStatsdServer(t, func(p []byte) {
		exp := "TEST.outbound.GET.in:1|c\nTEST.outbound.GET.200.count:1|c\nTEST.outbound.GET.200.time:1|ms\nTEST.outbound.GET.out:1|c"
		got := string(p)
		if got != exp {
			t.Errorf("expected: %v , got: %v", exp, got)
		}
	})
	require.NoError(t, err, "newStatsdServer() unexpected error = %v", err)

	defer srv.Close()

	c, err := New(
		WithPrefix("TEST"),
		WithNetwork(statsdTestNetwork),
		WithAddress(srv.addr),
	)
	require.NoError(t, err, "New() unexpected error = %v", err)

	defer c.Close() // nolint:errcheck

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`OK`))
			},
		),
	)
	defer server.Close()

	client := server.Client()
	client.Timeout = 1 * time.Second
	client.Transport = c.InstrumentRoundTripper(client.Transport)

	// nolint:noctx
	resp, err := client.Get(server.URL)
	require.NoError(t, err, "client.Get() unexpected error = %v", err)
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()
}

func TestIncLogLevelCounter(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "unexpected error = %v", err)

	c.IncLogLevelCounter("debug")
}

func TestIncErrorCounter(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "unexpected error = %v", err)

	c.IncErrorCounter("test_task", "test_operation", "3791")
}

type statsdServer struct {
	tb     testing.TB
	addr   string
	closer io.Closer
	closed chan bool
}

func newStatsdServer(tb testing.TB, f func([]byte)) (*statsdServer, error) {
	tb.Helper()

	s := &statsdServer{tb: tb, closed: make(chan bool)}

	laddr, err := net.ResolveUDPAddr(statsdTestNetwork, statsdTestAddr)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP(statsdTestNetwork, laddr)
	if err != nil {
		return nil, fmt.Errorf("unable to open UDP connection: %w", err)
	}

	s.closer = conn
	s.addr = conn.LocalAddr().String()

	go func() {
		buf := make([]byte, 1024)

		for {
			n, err := conn.Read(buf)
			if err != nil {
				s.closed <- true
				return
			}

			if n > 0 {
				f(buf[:n])
			}
		}
	}()

	return s, nil
}

func (s *statsdServer) Close() {
	if err := s.closer.Close(); err != nil {
		s.tb.Error(err)
	}

	<-s.closed
}
