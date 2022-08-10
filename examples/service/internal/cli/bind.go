package cli

import (
	"context"
	"fmt"
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

const (
	httpClientTimeout = 1 * time.Minute
)

// bind is the entry point of the service, this is where the wiring of all components happens.
func bind(cfg *appConfig, appInfo *jsendx.AppInfo, mtr instr.Metrics) bootstrap.BindFunc {
	return func(ctx context.Context, l *zap.Logger, m metrics.Client) error {
		// We assume the service is disabled and override the service binder if required
		serviceBinder := httpserver.NopBinder()
		statusHandler := jsendx.DefaultStatusHandler(appInfo)

		// common HTTP client used for all outbound requests
		httpClient := httpclient.New(
			httpclient.WithTimeout(httpClientTimeout),
			httpclient.WithRoundTripper(m.InstrumentRoundTripper),
			httpclient.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpclient.WithComponent(appInfo.ProgramName),
		)

		if cfg.Enabled {
			// wire the binder
			serviceBinder = httphandler.New(nil)

			// override the default healthcheck handler
			healthCheckHandler := healthcheck.NewHandler(
				[]healthcheck.HealthCheck{
					// healthcheck.New("<ID>", < HANDLER >),
					// healthcheck.NewWithTimeout("<ID>", <HANDLER>, <TIMEOUT>),
				},
				healthcheck.WithResultWriter(jsendx.HealthCheckResultWriter(appInfo)),
			)
			statusHandler = healthCheckHandler.ServeHTTP
		}

		// ipify client
		ipifyClient, err := ipify.New(
			ipify.WithHTTPClient(httpClient),
			ipify.WithTimeout(time.Duration(cfg.Ipify.Timeout)*time.Second),
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
			httpserver.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpserver.WithRouter(jsendx.NewRouter(appInfo, m.InstrumentHandler)), // set default 404, 405 and panic handlers
			httpserver.WithEnableAllDefaultRoutes(),
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

		// start public server
		httpPublicOpts := []httpserver.Option{
			httpserver.WithServerAddr(cfg.PublicAddress),
			httpserver.WithInstrumentHandler(m.InstrumentHandler),
			httpserver.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpserver.WithEnableDefaultRoutes(httpserver.PingRoute),
		}

		if err := httpserver.Start(ctx, serviceBinder, httpPublicOpts...); err != nil {
			return fmt.Errorf("error starting public HTTP server: %w", err)
		}

		return nil
	}
}
