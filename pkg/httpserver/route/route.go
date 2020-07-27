package route

import (
	"net/http"
)

// Route contains the HTTP route description
type Route struct {
	Method      string           `json:"method"`      // HTTP method
	Path        string           `json:"path"`        // URL path
	Handler     http.HandlerFunc `json:"-"`           // Handler function
	Description string           `json:"description"` // Description
}
