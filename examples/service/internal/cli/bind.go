package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gogenexampleowner/gogenexample/internal/db"
	"github.com/gogenexampleowner/gogenexample/internal/httphandlerpriv"
	"github.com/gogenexampleowner/gogenexample/internal/httphandlerpub"
	instr "github.com/gogenexampleowner/gogenexample/internal/metrics"
	"github.com/tecnickcom/gogen/pkg/bootstrap"
	"github.com/tecnickcom/gogen/pkg/healthcheck"
	"github.com/tecnickcom/gogen/pkg/httpclient"
	"github.com/tecnickcom/gogen/pkg/httpserver"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/httputil/jsendx"
	"github.com/tecnickcom/gogen/pkg/ipify"
	"github.com/tecnickcom/gogen/pkg/metrics"
	"github.com/tecnickcom/gogen/pkg/sqlconn"
	"github.com/tecnickcom/gogen/pkg/traceid"
)

// bind is the entry point of the service, this is where the wiring of all components happens.
//
//nolint:gocognit,funlen
func bind(cfg *appConfig, appInfo *jsendx.AppInfo, mtr instr.Metrics, wg *sync.WaitGroup, sc chan struct{}) bootstrap.BindFunc {
	return func(ctx context.Context, l *slog.Logger, m metrics.Client) error {
		jsx := jsendx.NewJSXResp(httputil.NewHTTPResp(l))

		// We assume the service is disabled and override the service binder if required
		serviceBinderPrivate := httpserver.NopBinder()
		serviceBinderPublic := httpserver.NopBinder()
		statusHandler := jsx.DefaultStatusHandler(appInfo)
		healthchecks := []healthcheck.HealthCheck{}

		// common HTTP client options used for all outbound requests
		httpClientOpts := []httpclient.Option{
			httpclient.WithLogger(l),
			httpclient.WithRoundTripper(m.InstrumentRoundTripper),
			httpclient.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpclient.WithComponent(appInfo.ProgramName),
		}

		ipifyTimeout := time.Duration(cfg.Clients.Ipify.Timeout) * time.Second
		ipifyHTTPClient := httpclient.New(
			append(httpClientOpts, httpclient.WithTimeout(ipifyTimeout))...,
		)

		ipifyClient, err := ipify.New(
			ipify.WithHTTPClient(ipifyHTTPClient),
			ipify.WithTimeout(ipifyTimeout),
			ipify.WithURL(cfg.Clients.Ipify.Address),
		)
		if err != nil {
			return fmt.Errorf("failed to build ipify client: %w", err)
		}

		//nolint:nestif
		if cfg.Enabled {
			serviceBinderPrivate = httphandlerpriv.New(nil, l)
			serviceBinderPublic = httphandlerpub.New(nil, l)

			reldb := db.Databases{
				Enabled: cfg.DB.Enabled,
			}

			if cfg.DB.Enabled {
				reldb.Main, healthchecks, err = newDatabase(ctx, "main", cfg.DB.Main, healthchecks, mtr, wg, sc)
				if err != nil {
					return err
				}

				reldb.Read, healthchecks, err = newDatabase(ctx, "read", cfg.DB.Read, healthchecks, mtr, wg, sc)
				if err != nil {
					return err
				}
			}

			// override the default healthcheck handler
			healthCheckHandler := healthcheck.NewHandler(
				healthchecks,
				healthcheck.WithResultWriter(jsx.HealthCheckResultWriter(appInfo)),
			)
			statusHandler = healthCheckHandler.ServeHTTP
		}

		middleware := func(args httpserver.MiddlewareArgs, next http.Handler) http.Handler {
			return m.InstrumentHandler(args.Path, next.ServeHTTP)
		}

		// MONITORING SERVER

		httpMonitoringOpts := []httpserver.Option{
			httpserver.WithLogger(l),
			httpserver.WithServerAddr(cfg.Servers.Monitoring.Address),
			httpserver.WithRequestTimeout(time.Duration(cfg.Servers.Monitoring.Timeout) * time.Second),
			httpserver.WithMetricsHandlerFunc(m.MetricsHandlerFunc()),
			httpserver.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpserver.WithMiddlewareFn(middleware),
			httpserver.WithNotFoundHandlerFunc(jsx.DefaultNotFoundHandlerFunc(appInfo)),
			httpserver.WithMethodNotAllowedHandlerFunc(jsx.DefaultMethodNotAllowedHandlerFunc(appInfo)),
			httpserver.WithPanicHandlerFunc(jsx.DefaultPanicHandlerFunc(appInfo)),
			httpserver.WithEnableAllDefaultRoutes(),
			httpserver.WithIndexHandlerFunc(jsx.DefaultIndexHandler(appInfo)),
			httpserver.WithIPHandlerFunc(jsx.DefaultIPHandler(appInfo, ipifyClient.GetPublicIP)),
			httpserver.WithPingHandlerFunc(jsx.DefaultPingHandler(appInfo)),
			httpserver.WithStatusHandlerFunc(statusHandler),
			httpserver.WithShutdownWaitGroup(wg),
			httpserver.WithShutdownSignalChan(sc),
		}

		httpMonitoringServer, err := httpserver.New(ctx, httpserver.NopBinder(), httpMonitoringOpts...)
		if err != nil {
			return fmt.Errorf("error creating monitoring HTTP server: %w", err)
		}

		httpMonitoringServer.StartServer()

		// PRIVATE SERVER

		httpPrivateOpts := []httpserver.Option{
			httpserver.WithLogger(l),
			httpserver.WithServerAddr(cfg.Servers.Private.Address),
			httpserver.WithRequestTimeout(time.Duration(cfg.Servers.Private.Timeout) * time.Second),
			httpserver.WithMiddlewareFn(middleware),
			httpserver.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpserver.WithEnableDefaultRoutes(httpserver.PingRoute),
			httpserver.WithShutdownWaitGroup(wg),
			httpserver.WithShutdownSignalChan(sc),
		}

		httpPrivateServer, err := httpserver.New(ctx, serviceBinderPrivate, httpPrivateOpts...)
		if err != nil {
			return fmt.Errorf("error creating private HTTP server: %w", err)
		}

		httpPrivateServer.StartServer()

		// PUBLIC SERVER

		httpPublicOpts := []httpserver.Option{
			httpserver.WithLogger(l),
			httpserver.WithServerAddr(cfg.Servers.Public.Address),
			httpserver.WithRequestTimeout(time.Duration(cfg.Servers.Public.Timeout) * time.Second),
			httpserver.WithMiddlewareFn(middleware),
			httpserver.WithTraceIDHeaderName(traceid.DefaultHeader),
			httpserver.WithEnableDefaultRoutes(httpserver.PingRoute),
			httpserver.WithShutdownWaitGroup(wg),
			httpserver.WithShutdownSignalChan(sc),
		}

		httpPublicServer, err := httpserver.New(ctx, serviceBinderPublic, httpPublicOpts...)
		if err != nil {
			return fmt.Errorf("error creating public HTTP server: %w", err)
		}

		httpPublicServer.StartServer()

		// example of custom metric
		mtr.IncExampleCounter("START")

		return nil
	}
}

func newDatabase(
	ctx context.Context,
	name string,
	dbcfg cfgDB,
	healthchecks []healthcheck.HealthCheck,
	mtr instr.Metrics,
	wg *sync.WaitGroup,
	sc chan struct{},
) (*sqlconn.SQLConn, []healthcheck.HealthCheck, error) {
	// Extra options are required to correctly parse time.Time and for SQLX to work properly with projections that use joins.
	// Ref.: https://pkg.go.dev/github.com/go-sql-driver/mysql#readme-usage
	dbDSN := dbcfg.DSN // + "?parseTime=true&columnsWithAlias=true"

	sqlConnOpts := []sqlconn.Option{
		sqlconn.WithDefaultDriver(dbcfg.Driver),
		sqlconn.WithPingTimeout(time.Duration(dbcfg.TimeoutPing) * time.Second),
		sqlconn.WithConnMaxOpen(dbcfg.ConnMaxOpen),
		sqlconn.WithConnMaxIdleCount(dbcfg.ConnMaxIdleCount),
		sqlconn.WithConnMaxIdleTime(time.Duration(dbcfg.ConnMaxIdleTime) * time.Second),
		sqlconn.WithConnMaxLifetime(time.Duration(dbcfg.ConnMaxLifetime) * time.Second),
		sqlconn.WithShutdownWaitGroup(wg),
		sqlconn.WithShutdownSignalChan(sc),
	}

	sqlConn, err := sqlconn.New(ctx, dbcfg.Driver, dbDSN, sqlConnOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to %s DB: %w", name, err)
	}

	err = mtr.InstrumentDB("db_"+name, sqlConn.DB())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to instrument %s DB: %w", name, err)
	}

	return sqlConn, append(healthchecks, healthcheck.New("db", sqlConn)), nil
}
