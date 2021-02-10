package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gosrvlibexample/gosrvlibexample/internal/httphandler"
	instr "github.com/gosrvlibexample/gosrvlibexample/internal/metrics"
	"github.com/nexmoinc/gosrvlib/pkg/bootstrap"
	"github.com/nexmoinc/gosrvlib/pkg/healthcheck"
	"github.com/nexmoinc/gosrvlib/pkg/httpclient"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver"
	"github.com/nexmoinc/gosrvlib/pkg/httputil/jsendx"
	"github.com/nexmoinc/gosrvlib/pkg/ipify"
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	"github.com/nexmoinc/gosrvlib/pkg/traceid"
	"go.uber.org/zap"
)

// bind is the entry point of the service, this is where the wiring of all components happens.
func bind(cfg *appConfig, appInfo *jsendx.AppInfo, mtr instr.Metrics) bootstrap.BindFunc {
	return func(ctx context.Context, l *zap.Logger, m metrics.Client) error {
		var statusHandler http.HandlerFunc

		// We assume the service is disabled and override the service binder if required
		serviceBinder := httpserver.NopBinder()
		if cfg.Enabled {
			// NOTE: Add initialization and wiring of external components and service here
			//
			// <INIT CODE HERE>
			//
			serviceBinder = httphandler.New(nil)

			// Custom healthcheck handler with JSendX response
			healthCheckHandler := healthcheck.NewHandler(
				[]healthcheck.HealthCheck{
					// healthcheck.New("<ID>", < HANDLER >),
					// healthcheck.NewWithTimeout("<ID>", <HANDLER>, <TIMEOUT>),
				},
				healthcheck.WithResultWriter(jsendx.HealthCheckResultWriter(appInfo)),
			)
			statusHandler = healthCheckHandler.ServeHTTP
		} else {
			statusHandler = jsendx.DefaultStatusHandler(appInfo)
		}

		// ipify client
		ipcTimeout := time.Duration(cfg.Ipify.Timeout) * time.Second
		ipc := httpclient.New(
			httpclient.WithTimeout(ipcTimeout),
			httpclient.WithRoundTripper(m.InstrumentRoundTripper),
			httpclient.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpclient.WithComponent("ipify"),
		)

		ipifyClient, err := ipify.NewClient(
			ipify.WithHTTPClient(ipc),
			ipify.WithTimeout(ipcTimeout),
			ipify.WithURL(cfg.Ipify.Address),
		)
		if err != nil {
			return fmt.Errorf("failed to build ipify client: %w", err)
		}

		// start monitoring server
		httpMonitoringOpts := []httpserver.Option{
			httpserver.WithServerAddr(cfg.MonitoringAddress),
			httpserver.WithMetricsHandlerFunc(m.MetricsHandlerFunc()),
			httpserver.WithInstrumentHandler(m.InstrumentHandler),
			httpserver.WithEnableAllDefaultRoutes(),
			httpserver.WithRouter(jsendx.NewRouter(appInfo, m.InstrumentHandler)), // set default 404, 405 and panic handlers
			httpserver.WithIndexHandlerFunc(jsendx.DefaultIndexHandler(appInfo)),
			httpserver.WithIPHandlerFunc(jsendx.DefaultIPHandler(appInfo, ipifyClient.GetPublicIP)),
			httpserver.WithPingHandlerFunc(jsendx.DefaultPingHandler(appInfo)),
			httpserver.WithStatusHandlerFunc(statusHandler),
		}

		if err := httpserver.Start(ctx, httpserver.NopBinder(), httpMonitoringOpts...); err != nil {
			return fmt.Errorf("error starting monitoring HTTP server: %w", err)
		}

		// example of custom metric
		mtr.IncExampleCounter("START")

		// start service server
		httpServiceOpts := []httpserver.Option{
			httpserver.WithServerAddr(cfg.PublicAddress),
			httpserver.WithEnableDefaultRoutes(httpserver.PingRoute),
		}

		if err := httpserver.Start(ctx, serviceBinder, httpServiceOpts...); err != nil {
			return fmt.Errorf("error starting service HTTP server: %w", err)
		}

		return nil
	}
}
