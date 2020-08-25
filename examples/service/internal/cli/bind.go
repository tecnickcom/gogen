package cli

import (
	"context"
	"fmt"

	"github.com/nexmoinc/gosrvlib-sample-service/internal/httphandler"
	"github.com/nexmoinc/gosrvlib/pkg/bootstrap"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver"
	"github.com/nexmoinc/gosrvlib/pkg/httputil/jsendx"
	"go.uber.org/zap"
)

// bind is the entry point of the service, this is where the wiring of all components happens
func bind(cfg *appConfig, appInfo *jsendx.AppInfo) bootstrap.BindFunc {
	return func(ctx context.Context, l *zap.Logger) error {
		defaultServerOpts := []httpserver.Option{
			// NOTE: Uncomment to use the JSendX router for 404, 405 and panic handlers
			// httpserver.WithRouter(jsendx.NewRouter(appInfo)),

			httpserver.WithRoutesIndexHandlerFunc(jsendx.DefaultRoutesIndexHandler(appInfo)),
			httpserver.WithPingHandlerFunc(jsendx.DefaultPingHandler(appInfo)),

			// NOTE: Uncomment this line to enable custom health check reporting
			// httpserver.WithStatusHandlerFunc(healthCheckHandler),
			httpserver.WithStatusHandlerFunc(jsendx.DefaultStatusHandler(appInfo)),
		}

		// We assume the service is disabled and override the service binder if required
		serviceBinder := httpserver.NopBinder()
		if cfg.Enabled {
			// NOTE: Add initialization and wiring of external components and service here
			//
			// <INIT CODE HERE>
			//
			serviceBinder = httphandler.New(nil)

			// NOTE: Uncomment the following block use a custom healthcheck handler
			//
			// // NOTE: Uncomment to create a custom healthcheck handler with default response
			// // healthCheckHandler := healthcheck.NewHandler(
			// // 	[]healthcheck.HealthCheck{
			// // 		healthcheck.New("<ID>", < HANDLER >),
			// // 		healthcheck.NewWithTimeout("<ID>", <HANDLER>, <TIMEOUT>),
			// // 	},
			// // )
			//
			// // NOTE: Uncomment to create a custom healthcheck handler with JSendX response
			// // healthCheckHandler := healthcheck.NewHandler(
			// // 	[]healthcheck.HealthCheck{
			// // 		healthcheck.New("<ID>", < HANDLER >),
			// // 		healthcheck.NewWithTimeout("<ID>", <HANDLER>, <TIMEOUT>),
			// // 	},
			// // 	healthcheck.WithResultWriter(jsendx.HealthCheckResultWriter(appInfo)),
			// // )
			//
			// // override the default healthcheck handler
			// defaultServerOpts = append(defaultServerOpts, httpserver.WithStatusHandlerFunc(healthCheckHandler.ServeHTTP))
		}

		httpServiceOpts := append(defaultServerOpts, httpserver.WithServerAddr(cfg.ServerAddress))

		// Use a separate server for monitoring routes if monitor_address and server_address are different
		if cfg.MonitoringAddress != cfg.ServerAddress {
			// Disable default routes as the monitoring routes on a separate server instance
			httpServiceOpts = append(httpServiceOpts, httpserver.WithDisableDefaultRoutes())

			// Prepare monitoring options
			httpMonitoringOpts := append(defaultServerOpts, httpserver.WithServerAddr(cfg.MonitoringAddress))

			if err := httpserver.Start(ctx, httpserver.NopBinder(), httpMonitoringOpts...); err != nil {
				return fmt.Errorf("error starting monitoring HTTP server: %w", err)
			}
		}

		if err := httpserver.Start(ctx, serviceBinder, httpServiceOpts...); err != nil {
			return fmt.Errorf("error starting service HTTP server: %w", err)
		}

		return nil
	}
}
