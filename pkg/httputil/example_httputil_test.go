package httputil_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/tecnickcom/gogen/pkg/httputil"
)

func ExampleLink() {
	// A trusted, compile-time template is used as a fmt.Sprintf format string.
	fmt.Println(httputil.Link("https://api.example.invalid", "users/%s", "42"))

	// Untrusted string segments must be escaped by the caller to avoid injecting
	// path or query delimiters.
	fmt.Println(httputil.Link("https://api.example.invalid", "users/%s", url.PathEscape("a/b?c")))

	// Output:
	// https://api.example.invalid/users/42
	// https://api.example.invalid/users/a%2Fb%3Fc
}

func ExampleQueryIntOrDefault() {
	q := url.Values{"limit": []string{"25"}}

	fmt.Println(httputil.QueryIntOrDefault(q, "limit", 10))
	fmt.Println(httputil.QueryIntOrDefault(q, "missing", 10))

	// Output:
	// 25
	// 10
}

func ExampleNewHTTPResp() {
	// The logger is where structured response entries go; the body is written to w.
	res := httputil.NewHTTPResp(slog.New(slog.DiscardHandler))

	rr := httptest.NewRecorder()
	res.SendJSON(context.Background(), rr, http.StatusOK, struct {
		Message string `json:"message"`
	}{Message: "hello"})

	fmt.Print(rr.Body.String())

	// Output: {"message":"hello"}
}

func ExampleNewResponseWriterWrapper() {
	// Middleware wraps the writer to observe the status and byte size a handler produced.
	rr := httptest.NewRecorder()
	ww := httputil.NewResponseWriterWrapper(rr)

	ww.WriteHeader(http.StatusCreated)
	_, _ = ww.Write([]byte("created"))

	fmt.Printf("status=%d size=%d\n", ww.Status(), ww.Size())

	// Output: status=201 size=7
}
