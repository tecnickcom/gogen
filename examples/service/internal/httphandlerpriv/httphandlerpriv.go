// Package httphandlerpriv handles the inbound internal service requests.
package httphandlerpriv

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/tecnickcom/nurago/pkg/httpserver"
	"github.com/tecnickcom/nurago/pkg/httputil"
	"github.com/tecnickcom/nurago/pkg/random"
)

// Service is the interface representing the business logic of the service.
type Service any

// HTTPHandlerPrivate is the struct containing all the http handlers.
type HTTPHandlerPrivate struct {
	service Service
	httpres *httputil.HTTPResp
	rnd     *random.Rnd
}

// New creates a private API handler with shared response and UID utilities for
// endpoints on the service's private exposure boundary.
func New(s Service, l *slog.Logger) *HTTPHandlerPrivate {
	return &HTTPHandlerPrivate{
		service: s,
		httpres: httputil.NewHTTPResp(l),
		rnd:     random.New(nil),
	}
}

// BindHTTP returns the private routes exposed by this handler. The route list
// is consumed by nurago's HTTP server binder.
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

// handleGenUID responds with a UUIDv7 string in JSON format.
func (h *HTTPHandlerPrivate) handleGenUID(w http.ResponseWriter, r *http.Request) {
	h.httpres.SendJSON(r.Context(), w, http.StatusOK, h.rnd.UUIDv7().String())
}
