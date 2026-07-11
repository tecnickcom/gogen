package httpserver_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tecnickcom/nurago/pkg/httpserver"
)

// exampleBinder supplies the custom service routes.
type exampleBinder struct{}

func (b *exampleBinder) BindHTTP(_ context.Context) []httpserver.Route {
	return []httpserver.Route{
		{
			Method:      http.MethodGet,
			Path:        "/hello",
			Description: "Says hello.",
			Handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("hello"))
			},
		},
	}
}

func ExampleNew() {
	ctx := context.Background()

	srv, err := httpserver.New(
		ctx,
		&exampleBinder{},
		httpserver.WithServerAddr(":0"), // ephemeral port; see srv.Addr() for the actual address
		httpserver.WithEnableDefaultRoutes(httpserver.PingRoute, httpserver.StatusRoute),
		httpserver.WithRequestTimeout(30*time.Second),
		httpserver.WithShutdownTimeout(5*time.Second),
		httpserver.WithLogger(slog.New(slog.DiscardHandler)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// The server runs in the background; canceling ctx or calling Shutdown
	// stops it gracefully.
	srv.StartServer()

	// ... serve traffic ...

	err = srv.Shutdown(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("server stopped")

	// Output:
	// server stopped
}
