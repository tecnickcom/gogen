// Package httphandlerpub handles the inbound public service requests.
package httphandlerpub

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/tecnickcom/gogen/pkg/httpserver"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/random"
)

// Service is the interface representing the business logic of the service.
type Service any

// HTTPHandlerPublic is the struct containing all the http handlers.
type HTTPHandlerPublic struct {
	service Service
	httpres *httputil.HTTPResp
	rnd     *random.Rnd
}

// New creates a public API handler with shared response and UID utilities.
//
// It solves the same routing and response consistency problem as the private
// handler, but for endpoints intended for external consumers.
func New(s Service, l *slog.Logger) *HTTPHandlerPublic {
	return &HTTPHandlerPublic{
		service: s,
		httpres: httputil.NewHTTPResp(l),
		rnd:     random.New(nil),
	}
}

// BindHTTP returns the public routes exposed by this handler.
//
// Defining public routes as data makes endpoint exposure explicit and easy to
// evolve without changing server bootstrap logic.
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

// handleGenUID responds with a UUIDv7 string in JSON format.
//
// It provides a simple, externally reachable route useful for integration
// checks and example client wiring.
func (h *HTTPHandlerPublic) handleGenUID(w http.ResponseWriter, r *http.Request) {
	h.httpres.SendJSON(r.Context(), w, http.StatusOK, h.rnd.UUIDv7().String())
}
