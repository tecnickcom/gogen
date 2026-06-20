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

// bind returns the bootstrap bind function that wires the full runtime graph.
//
// It solves the composition problem in service startup by centralizing how
// handlers, clients, middleware, metrics, health checks, and servers are
// connected.
//
// Top features for developers:
//   - Builds shared outbound HTTP client options (logging, tracing,
//     instrumentation) once and reuses them consistently.
//   - Enables or disables optional components (service routes and databases)
//     from configuration.
//   - Starts monitoring, private, and public servers with consistent timeout,
//     middleware, panic, and shutdown behavior.
//   - Upgrades status reporting from default status to dependency-aware health
//     checks when enabled.
//
// This gives a clear, extensible startup blueprint that keeps operational
// behavior consistent as the service grows.
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
		//
		//nolint:prealloc
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
			// This example has no business logic, so the service is nil. In a real
			// service, replace nil with your service implementation: this is the
			// injection point for the private and public handlers.
			serviceBinderPrivate = httphandlerpriv.New(nil, l)
			serviceBinderPublic = httphandlerpub.New(nil, l)

			// reldb holds the database connections. In this example they are wired
			// only into health checks and graceful shutdown (below and inside
			// newDatabase); a real service would also pass reldb to the handlers
			// created above.
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

		// PRIVATE AND PUBLIC SERVERS
		//
		// Both expose only the default ping route plus their binder's routes, so
		// they are created through the shared startServiceServer helper.

		err = startServiceServer(ctx, "private", serviceBinderPrivate, cfgServer(cfg.Servers.Private), l, middleware, wg, sc)
		if err != nil {
			return err
		}

		err = startServiceServer(ctx, "public", serviceBinderPublic, cfgServer(cfg.Servers.Public), l, middleware, wg, sc)
		if err != nil {
			return err
		}

		// example of custom metric
		mtr.IncExampleCounter("START")

		return nil
	}
}

// startServiceServer builds, starts, and registers a service HTTP server that
// exposes the default ping route plus the routes provided by binder.
//
// The private and public servers share this path because they differ only by
// name, address, timeout, and binder.
func startServiceServer(
	ctx context.Context,
	name string,
	binder httpserver.Binder,
	srv cfgServer,
	l *slog.Logger,
	middleware httpserver.MiddlewareFn,
	wg *sync.WaitGroup,
	sc chan struct{},
) error {
	opts := []httpserver.Option{
		httpserver.WithLogger(l),
		httpserver.WithServerAddr(srv.Address),
		httpserver.WithRequestTimeout(time.Duration(srv.Timeout) * time.Second),
		httpserver.WithMiddlewareFn(middleware),
		httpserver.WithTraceIDHeaderName(traceid.DefaultHeader),
		httpserver.WithEnableDefaultRoutes(httpserver.PingRoute),
		httpserver.WithShutdownWaitGroup(wg),
		httpserver.WithShutdownSignalChan(sc),
	}

	server, err := httpserver.New(ctx, binder, opts...)
	if err != nil {
		return fmt.Errorf("error creating %s HTTP server: %w", name, err)
	}

	server.StartServer()

	return nil
}

// newDatabase creates, instruments, and health-checks a named SQL connection
// so DB dependencies can be added with one reusable path.
//
// It standardizes connection pool options, startup ping behavior, metrics
// instrumentation, and health integration, reducing the risk of divergent DB
// setup between main/read or future database roles.
func newDatabase(
	ctx context.Context,
	name string,
	dbcfg cfgDB,
	healthchecks []healthcheck.HealthCheck,
	mtr instr.Metrics,
	wg *sync.WaitGroup,
	sc chan struct{},
) (*sqlconn.SQLConn, []healthcheck.HealthCheck, error) {
	// The commented-out DSN suffix below is MySQL-specific: it is required to
	// correctly parse time.Time and for SQLX to work properly with projections
	// that use joins. It is left disabled because this example ships without a
	// live database; enable it (or adapt it to your driver) for a real connection.
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
		sqlconn.WithSQLOpenFunc(mtr.SqlOpen),
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
