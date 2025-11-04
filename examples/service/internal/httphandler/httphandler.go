// Package httphandler handles the inbound service requests.
package httphandler

import (
	"context"
	"net/http"

	"github.com/tecnickcom/gogen/pkg/httpserver"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/uidc"
)

// Service is the interface representing the business logic of the service.
type Service any

// New creates a new instance of the HTTP handler.
func New(s Service) *HTTPHandler {
	return &HTTPHandler{
		service: s,
	}
}

// HTTPHandler is the struct containing all the http handlers.
type HTTPHandler struct {
	service Service
}

// BindHTTP implements the function to bind the handler to a server.
func (h *HTTPHandler) BindHTTP(_ context.Context) []httpserver.Route {
	return []httpserver.Route{
		{
			Method:      http.MethodGet,
			Path:        "/uid",
			Description: "Generates a random UID",
			Handler:     h.handleGenUID,
		},
	}
}

func (h *HTTPHandler) handleGenUID(w http.ResponseWriter, r *http.Request) {
	httputil.SendJSON(r.Context(), w, http.StatusOK, uidc.NewID128())
}
