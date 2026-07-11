package httpreverseproxy_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/julienschmidt/httprouter"
	"github.com/tecnickcom/nurago/pkg/httpreverseproxy"
)

// ExampleClient_ForwardRequest wires the reverse-proxy client to a catch-all route
// and forwards an incoming request to an upstream service. The catch-all parameter
// must be named "path" (the default) so the segment after "/proxy/" becomes the
// upstream path; use httpreverseproxy.WithPathParam to change the name.
func ExampleClient_ForwardRequest() {
	// Stand-in upstream service that echoes the path it received.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The example intentionally reflects the request path to show how it is
		// rewritten; a real upstream would not echo untrusted input.
		fmt.Fprintf(w, "upstream received %s", r.URL.Path) //nolint:gosec // G705: illustrative echo
	}))
	defer upstream.Close()

	client, err := httpreverseproxy.New(upstream.URL + "/v2")
	if err != nil {
		panic(err)
	}

	// Register the proxy under a catch-all route; httprouter injects the matched
	// parameters into the request context that the default rewrite reads.
	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/proxy/*path", client.ForwardRequest)

	edge := httptest.NewServer(router)
	defer edge.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, edge.URL+"/proxy/users", nil)
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	// Output: upstream received /v2/users
}
