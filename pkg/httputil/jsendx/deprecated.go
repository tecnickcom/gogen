package jsendx

import (
	"net/http"
	"runtime/debug"

	"github.com/Vonage/gosrvlib/pkg/httpserver"
	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// NewRouter is deprecated.
// Deprecated: Set instead the router error handlers with the following options:
//
//	httpserver.WithNotFoundHandlerFunc(jsendx.DefaultNotFoundHandlerFunc(appInfo))
//	httpserver.WithMethodNotAllowedHandlerFunc(jsendx.DefaultMethodNotAllowedHandlerFunc(appInfo))
//	httpserver.WithPanicHandlerFunc(jsendx.DefaultPanicHandlerFunc(appInfo))
func NewRouter(info *AppInfo, instrumentHandler httpserver.InstrumentHandler) *httprouter.Router {
	r := httprouter.New()

	r.NotFound = instrumentHandler("404", func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusNotFound, info, "invalid endpoint")
	})

	r.MethodNotAllowed = instrumentHandler("405", func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusMethodNotAllowed, info, "the request cannot be routed")
	})

	r.PanicHandler = func(w http.ResponseWriter, r *http.Request, p any) {
		logging.FromContext(r.Context()).Error("panic",
			zap.Any("err", p),
			zap.String("stacktrace", string(debug.Stack())),
		)
		Send(r.Context(), w, http.StatusInternalServerError, info, "internal error")
	}

	return r
}
