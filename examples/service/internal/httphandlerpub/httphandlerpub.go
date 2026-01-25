// Package httphandlerpub handles the inbound public service requests.
package httphandlerpub

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

// HTTPHandlerPublic is the struct containing all the http handlers.
type HTTPHandlerPublic struct {
	service Service
	httpres *httputil.HTTPResp
}

// New creates a new instance of the HTTP handler.
func New(s Service, l *slog.Logger) *HTTPHandlerPublic {
	return &HTTPHandlerPublic{
		service: s,
		httpres: httputil.NewHTTPResp(l),
	}
}

// BindHTTP implements the function to bind the handler to a server.
func (h *HTTPHandlerPublic) BindHTTP(_ context.Context) []httpserver.Route {
	return []httpserver.Route{
		{
			Method:      http.MethodGet,
			Path:        "/uid",
			Description: "Generates a random UID",
			Handler:     h.handleGenUID,
		},
	}
}

func (h *HTTPHandlerPublic) handleGenUID(w http.ResponseWriter, r *http.Request) {
	h.httpres.SendJSON(r.Context(), w, http.StatusOK, uidc.NewID128())
}
