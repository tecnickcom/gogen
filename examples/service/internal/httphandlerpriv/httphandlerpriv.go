// Package httphandlerpriv handles the inbound internal service requests.
package httphandlerpriv

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/tecnickcom/gogen/pkg/httpserver"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/uidc"
)

// Service is the interface representing the business logic of the service.
type Service any

// HTTPHandlerPrivate is the struct containing all the http handlers.
type HTTPHandlerPrivate struct {
	service Service
	httpres *httputil.HTTPResp
}

// New creates a new instance of the HTTP handler.
func New(s Service, l *slog.Logger) *HTTPHandlerPrivate {
	return &HTTPHandlerPrivate{
		service: s,
		httpres: httputil.NewHTTPResp(l),
	}
}

// BindHTTP implements the function to bind the handler to a server.
func (h *HTTPHandlerPrivate) BindHTTP(_ context.Context) []httpserver.Route {
	return []httpserver.Route{
		{
			Method:      http.MethodGet,
			Path:        "/uid",
			Description: "Generates a random UID",
			Handler:     h.handleGenUID,
		},
	}
}

func (h *HTTPHandlerPrivate) handleGenUID(w http.ResponseWriter, r *http.Request) {
	h.httpres.SendJSON(r.Context(), w, http.StatusOK, uidc.NewID128())
}
