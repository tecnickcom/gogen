package httpclient_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/tecnickcom/gogen/pkg/httpclient"
	"github.com/tecnickcom/gogen/pkg/traceid"
)

func ExampleNew() {
	// Build a reusable client from composable options. Create it once and reuse
	// it: each client owns a private transport and connection pool.
	client := httpclient.New(
		httpclient.WithTimeout(5*time.Second),
		httpclient.WithComponent("my-service"),
		httpclient.WithTraceIDHeaderName(traceid.DefaultHeader),
		httpclient.WithLogPrefix("http_"),
	)

	// Release pooled connections when the client is no longer needed.
	defer client.CloseIdleConnections()

	fmt.Println(client != nil)

	// Output:
	// true
}

func ExampleClient_Do() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	}))

	client := httpclient.New()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Always close the body: it releases the per-request timeout timer and the
	// underlying connection (use defer resp.Body.Close() in real code).
	_ = resp.Body.Close()
	server.Close()

	fmt.Println(string(body))

	// Output:
	// pong
}
