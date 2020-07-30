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
		// NOTE: Add initialization and wiring of external components and service here
		//
		// <INIT CODE HERE>
		//

		// NOTE: Uncomment this line if only default HTTP routes exist
		// hh := httpserver.NopBinder()
		hh := httphandler.New(nil)

		// NOTE: Uncomment to create a custom healthcheck handler
		// healthCheckHandler := healthcheck.Handler(healthcheck.HealthCheckerMap{
		// 	"<extCompName>": extCompInstance,
		// }, appInfo)

		defaultServerOpts := []httpserver.Option{
			// NOTE: Uncomment to use the JSendX router for 404, 405 and panic handlers
			// httpserver.WithRouter(jsendx.NewRouter(appInfo)),

			httpserver.WithRoutesIndexHandlerFunc(jsendx.DefaultRoutesIndexHandler(appInfo)),
			httpserver.WithPingHandlerFunc(jsendx.DefaultPingHandler(appInfo)),

			// NOTE: Uncomment this line to enable custom health check reporting
			// httpserver.WithStatusHandlerFunc(healthCheckHandler),
			httpserver.WithStatusHandlerFunc(jsendx.DefaultStatusHandler(appInfo)),
		}

		// Use a separate server for monitoring routes if monitor_address and server_address are different
		if cfg.MonitoringAddress != cfg.ServerAddress {
			httpMonitoringOpts := append(defaultServerOpts, []httpserver.Option{
				httpserver.WithServerAddr(cfg.MonitoringAddress),
			}...)
			if err := httpserver.Start(ctx, httpserver.NopBinder(), httpMonitoringOpts...); err != nil {
				return fmt.Errorf("error starting monitoring HTTP server: %w", err)
			}
		}

		httpServiceOpts := append(defaultServerOpts, []httpserver.Option{
			httpserver.WithServerAddr(cfg.ServerAddress),
		}...)

		// Disable default routes if we are starting the monitoring routes on a separate server instance
		if cfg.MonitoringAddress != cfg.ServerAddress {
			httpServiceOpts = append(httpServiceOpts, []httpserver.Option{
				httpserver.WithDisableDefaultRoutes(),
			}...)
		}

		if err := httpserver.Start(ctx, hh, httpServiceOpts...); err != nil {
			return fmt.Errorf("error starting service HTTP server: %w", err)
		}

		return nil
	}
}
