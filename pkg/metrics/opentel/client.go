package opentel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceBatchTimeoutSec = 5
	metricIntervalSec    = 60

	// shutdownTimeoutSec bounds the context-free [Client.Close] so a hung
	// exporter cannot block process shutdown forever.
	shutdownTimeoutSec = 10
)

const (
	labelCode      = "code"
	labelDBname    = "db.name"
	labelLevel     = "level"
	labelOperation = "operation"
	labelTask      = "task"
)

const (
	// instrumentationScope is the OpenTelemetry instrumentation scope (meter and
	// tracer name). Per OTel conventions it is the instrumenting library's
	// import path rather than a metric category.
	instrumentationScope = "github.com/tecnickcom/nurago/pkg/metrics/opentel"

	// NameLogLevel is the name of the counter that records the number of log
	// lines emitted for each log severity level.
	NameLogLevel = "log_level_total"

	// NameErrorCode is the name of the counter that records the number of errors
	// by task, operation and error code.
	NameErrorCode = "error_code_total"

	// descLogLevel documents the log-level counter.
	descLogLevel = "Number of log lines emitted for each severity level."

	// descErrorCode documents the error-code counter.
	descErrorCode = "Number of errors by task, operation and error code."

	// unitLogRecord is the UCUM unit annotation for a count of log records.
	unitLogRecord = "{log_record}"

	// unitError is the UCUM unit annotation for a count of errors.
	unitError = "{error}"
)

// TShutdownFuncs aliases a shutdown callback used to flush/close OTel resources.
type TShutdownFuncs = func(ctx context.Context) error

// SDKResourceFunc is a function that returns an SDK Resource.
type SDKResourceFunc = func(ctx context.Context, name, version string) *sdkresource.Resource

// TraceProviderFunc is a function that returns an SDK Trace Provider.
//
// It returns an error so custom implementations can surface exporter or
// provider construction failures instead of panicking or silently swallowing
// them.
type TraceProviderFunc = func(ctx context.Context, res *sdkresource.Resource) (*sdktrace.TracerProvider, error)

// MetricProviderFunc is a function that returns an SDK meter provider.
//
// It returns an error so custom implementations can surface exporter or
// provider construction failures instead of panicking or silently swallowing
// them.
type MetricProviderFunc = func(ctx context.Context, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error)

// Client is an OpenTelemetry-backed implementation of the shared metrics
// interface.
//
// Create it with [New].
type Client struct {
	propagator          propagation.TextMapPropagator
	resFn               SDKResourceFunc
	res                 *sdkresource.Resource
	tracerProviderFn    TraceProviderFunc
	tracerProvider      *sdktrace.TracerProvider
	meterProviderFn     MetricProviderFunc
	meterProvider       *sdkmetric.MeterProvider
	mu                  sync.Mutex // guards shutdownFuncs
	shutdownFuncs       []TShutdownFuncs
	collectorErrorLevel metric.Int64Counter
	collectorErrorCode  metric.Int64Counter
}

// New creates an OpenTelemetry metrics/tracing client and installs global OTel
// providers.
//
// name and version can also be provided via environment variables:
// OTEL_SERVICE_NAME and OTEL_SERVICE_VERSION.
// deployment.environment.name can be provided via
// OTEL_DEPLOYMENT_ENVIRONMENT_NAME.
//
// The same attributes can also be supplied through OTEL_RESOURCE_ATTRIBUTES
// using keys: service.name, service.version, deployment.environment.name.
//
// New installs process-global OpenTelemetry providers (tracer, meter,
// propagator) on success, so at most one opentel [Client] should be created per
// process; constructing a second one overwrites the global providers.
func New(ctx context.Context, name, version string, opts ...Option) (*Client, error) {
	c := initClient()

	for _, applyOpt := range opts {
		err := applyOpt(c)
		if err != nil {
			return nil, err
		}
	}

	err := c.set(ctx, name, version)
	if err != nil {
		// Counter/provider setup failed: tear down anything already installed
		// and return no client so callers can't use a half-built instance.
		return nil, errors.Join(err, c.CloseCtx(ctx))
	}

	return c, nil
}

// initClient returns a Client instance with default values.
func initClient() *Client {
	return &Client{}
}

// SqlOpen wraps sql.Open with OTel instrumentation.
func (c *Client) SqlOpen(driverName, dsn string) (*sql.DB, error) {
	options := c.otelsqlOpts(
		otelsql.WithAttributes(otelsql.AttributesFromDSN(dsn)...),
	)

	return otelsql.Open(driverName, dsn, options...) //nolint:wrapcheck
}

// InstrumentDB wraps a sql.DB to collect connection pool statistics.
// For full query tracing, use SqlOpen() before creating the sql.DB.
func (c *Client) InstrumentDB(dbName string, db *sql.DB) error {
	reg, err := defRegisterDBStatsMetrics(
		db,
		c.otelsqlOpts(
			otelsql.WithAttributes(
				attribute.String(labelDBname, dbName),
			),
		)...,
	)
	if err != nil {
		return fmt.Errorf("failed instrumenting the database: %w", err)
	}

	c.appendShutdown(func(_ context.Context) error {
		return reg.Unregister()
	})

	return nil
}

func defRegisterDBStatsMetrics(db *sql.DB, opts ...otelsql.Option) (metric.Registration, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	return otelsql.RegisterDBStatsMetrics(db, opts...) //nolint:wrapcheck
}

// InstrumentHandler wraps handler with OpenTelemetry HTTP server
// instrumentation bound to this client's providers and propagator.
//
// path is used as the span/operation name and MUST be a low-cardinality route
// template, never a raw request URI containing identifiers.
func (c *Client) InstrumentHandler(path string, handler http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(
		handler,
		path,
		otelhttp.WithMeterProvider(c.meterProvider),
		otelhttp.WithTracerProvider(c.tracerProvider),
		otelhttp.WithPropagators(c.propagator),
	)
}

// InstrumentRoundTripper wraps next with OpenTelemetry HTTP client
// instrumentation bound to this client's providers and propagator.
// If next is nil, http.DefaultTransport is used.
func (c *Client) InstrumentRoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return otelhttp.NewTransport(
		next,
		otelhttp.WithMeterProvider(c.meterProvider),
		otelhttp.WithTracerProvider(c.tracerProvider),
		otelhttp.WithPropagators(c.propagator),
	)
}

// MetricsHandlerFunc returns a minimal health-style handler.
//
// OpenTelemetry metrics are exported by configured exporters, so this endpoint
// does not expose a scrape payload and returns "OK".
func (c *Client) MetricsHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`OK`)) }
}

// IncLogLevelCounter counts the number of log lines emitted for each log severity level.
func (c *Client) IncLogLevelCounter(level string) {
	// context.Background is intentional: this is a context-free instrumentation
	// point (see metrics.Client) typically driven by a logging hook with no
	// request context to propagate.
	c.collectorErrorLevel.Add(
		context.Background(),
		1,
		metric.WithAttributes(
			attribute.String(labelLevel, level),
		))
}

// IncErrorCounter increments the number of errors by task, operation and error code.
func (c *Client) IncErrorCounter(task, operation, code string) {
	// context.Background is intentional: see [Client.IncLogLevelCounter].
	c.collectorErrorCode.Add(
		context.Background(),
		1,
		metric.WithAttributes(
			attribute.String(labelTask, task),
			attribute.String(labelOperation, operation),
			attribute.String(labelCode, code),
		),
	)
}

// Close runs all registered shutdown callbacks using a bounded context.
//
// It applies a fixed timeout so a stalled exporter cannot block shutdown
// indefinitely. Use [Client.CloseCtx] when an explicit shutdown deadline is
// required.
func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeoutSec*time.Second)
	defer cancel()

	return c.CloseCtx(ctx)
}

// CloseCtx runs all registered shutdown callbacks with ctx, joining any errors.
//
// Callbacks run in reverse registration order (last registered first), so
// resources are torn down in the opposite order they were set up: DB-stats
// registrations are unregistered before the meter/tracer providers they depend
// on are shut down. It is safe to call concurrently and is idempotent.
func (c *Client) CloseCtx(ctx context.Context) error {
	c.mu.Lock()
	funcs := c.shutdownFuncs
	c.shutdownFuncs = nil
	c.mu.Unlock()

	var err error

	for _, fn := range slices.Backward(funcs) {
		err = errors.Join(err, fn(ctx))
	}

	return err
}

// appendShutdown registers a shutdown callback under the client mutex so it is
// safe to call concurrently with [Client.CloseCtx].
func (c *Client) appendShutdown(fn TShutdownFuncs) {
	c.mu.Lock()
	c.shutdownFuncs = append(c.shutdownFuncs, fn)
	c.mu.Unlock()
}

// otelsqlOpts merges default otelsql options with additional opts.
func (c *Client) otelsqlOpts(opts ...otelsql.Option) []otelsql.Option {
	return append([]otelsql.Option{
		otelsql.WithMeterProvider(c.meterProvider),
		otelsql.WithTracerProvider(c.tracerProvider),
	}, opts...)
}

func (c *Client) set(ctx context.Context, name, version string) error {
	// propagator
	if c.propagator == nil {
		c.propagator = DefaultPropagator()
	}

	// SDK resource
	if c.resFn == nil {
		c.resFn = DefaultSDKResource
	}

	c.res = c.resFn(ctx, name, version)

	// trace provider
	if c.tracerProviderFn == nil {
		c.tracerProviderFn = DefaultTracerProvider
	}

	tracerProvider, err := c.tracerProviderFn(ctx, c.res)
	if err != nil {
		return fmt.Errorf("failed to create the tracer provider: %w", err)
	}

	c.tracerProvider = tracerProvider
	c.appendShutdown(tracerProvider.Shutdown)

	// meter provider
	if c.meterProviderFn == nil {
		c.meterProviderFn = DefaultMeterProvider
	}

	meterProvider, err := c.meterProviderFn(ctx, c.res)
	if err != nil {
		return fmt.Errorf("failed to create the meter provider: %w", err)
	}

	c.meterProvider = meterProvider
	c.appendShutdown(meterProvider.Shutdown)

	meter := newMeter(meterProvider)
	cel, erra := setInt64Counter(meter, NameLogLevel, descLogLevel, unitLogRecord)
	cec, errb := setInt64Counter(meter, NameErrorCode, descErrorCode, unitError)

	err = errors.Join(erra, errb)
	if err != nil {
		return err
	}

	c.collectorErrorLevel = cel
	c.collectorErrorCode = cec

	// Install the global OTel providers only after the whole setup succeeded,
	// so a partial-setup failure (followed by the CloseCtx teardown in New)
	// never leaves shut-down providers installed as process globals.
	otel.SetTextMapPropagator(c.propagator)
	otel.SetTracerProvider(c.tracerProvider)
	otel.SetMeterProvider(c.meterProvider)

	return nil
}

// newMeter returns the meter used to register the internal counters.
// It is a package-level indirection so tests can force a counter-setup failure;
// in production it always delegates to the configured meter provider.
var newMeter = func(mp *sdkmetric.MeterProvider) metric.Meter { //nolint:gochecknoglobals
	return mp.Meter(instrumentationScope)
}

// setInt64Counter creates a named Int64 counter with a description and unit on
// the given meter. On failure it returns the wrapped error without tearing down
// the client; the caller (set/New) decides how to clean up.
func setInt64Counter(meter metric.Meter, name, description, unit string) (metric.Int64Counter, error) {
	counter, err := meter.Int64Counter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %q counter: %w", name, err)
	}

	return counter, nil
}

// DefaultPropagator provides a default OpenTelemetry TextMapPropagator.
func DefaultPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// DefaultSDKResource returns a default OTLP SDK resource.
// The name can be also set via the OTEL_SERVICE_NAME environment variable.
// The service can be also set via the OTEL_SERVICE_VERSION environment variable.
// The deployment environment can be set via the OTEL_DEPLOYMENT_ENVIRONMENT_NAME environment variable.
// The parameters can be also set as OTEL_RESOURCE_ATTRIBUTES environment variables attributes:
// service.name, service.version, deployment.environment.name.
func DefaultSDKResource(ctx context.Context, name, version string) *sdkresource.Resource {
	attrs := []attribute.KeyValue{}
	ora := parseOTELResourceAttributes(os.Getenv("OTEL_RESOURCE_ATTRIBUTES"))

	name = getOTELAttr(name, "OTEL_SERVICE_NAME", "service.name", ora)
	if name != "" {
		attrs = append(attrs, semconv.ServiceName(name))
	}

	version = getOTELAttr(version, "OTEL_SERVICE_VERSION", "service.version", ora)
	if version != "" {
		attrs = append(attrs, semconv.ServiceVersion(version))
	}

	env := getOTELAttr("", "OTEL_DEPLOYMENT_ENVIRONMENT_NAME", "deployment.environment.name", ora)
	if env != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentName(env))
	}

	res, err := sdkresource.New(
		ctx,
		sdkresource.WithAttributes(attrs...),
	)

	return resolveResource(res, err, attrs)
}

// resolveResource falls back to a schemaless resource when construction fails.
// sdkresource.New can report detection or schema-URL conflict errors and, in the
// worst case, return no resource at all. Rather than propagating a nil resource
// (which would drop all service metadata), it returns a schemaless resource
// carrying the explicit attributes.
func resolveResource(res *sdkresource.Resource, err error, attrs []attribute.KeyValue) *sdkresource.Resource {
	if err != nil && res == nil {
		return sdkresource.NewSchemaless(attrs...)
	}

	return res
}

// getOTELAttr returns the value of the OTEL attribute.
// If empty it searches for the envname environment variable and then for
// attrname key inside the ora key/value map.
func getOTELAttr(val, envname, attrname string, ora map[string]string) string {
	if val != "" {
		return val
	}

	val = os.Getenv(envname)
	if val != "" {
		return val
	}

	val, ok := ora[attrname]
	if ok {
		return val
	}

	return ""
}

// parseOTELResourceAttributes extracts key/value pairs.
func parseOTELResourceAttributes(val string) map[string]string {
	attr := make(map[string]string)

	if val == "" {
		return attr
	}

	for item := range strings.SplitSeq(val, ",") {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) == 2 {
			attr[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return attr
}

// DefaultTracerProviderWithExporter provides a default tracer provider for exp.
func DefaultTracerProviderWithExporter(res *sdkresource.Resource, exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exp,
			sdktrace.WithBatchTimeout(traceBatchTimeoutSec*time.Second),
		),
		sdktrace.WithResource(res),
	)
}

// DefaultTracerProviderStdout provides a default STDOUT OpenTelemetry Tracer Provider.
func DefaultTracerProviderStdout(_ context.Context, res *sdkresource.Resource) (*sdktrace.TracerProvider, error) {
	exp, err := stdouttrace.New()

	return DefaultTracerProviderWithExporter(res, exp), err
}

// DefaultTracerProviderOTLP provides a default OTLP OpenTelemetry Tracer Provider.
// The endpoint is defined by (in order of priority):
//   - OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
//   - OTEL_EXPORTER_OTLP_ENDPOINT
//   - "localhost:4318"
func DefaultTracerProviderOTLP(ctx context.Context, res *sdkresource.Resource) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracehttp.New(ctx)

	return DefaultTracerProviderWithExporter(res, exp), err
}

// DefaultTracerProvider provides a default OpenTelemetry Tracer Provider.
// If neither OTEL_EXPORTER_OTLP_TRACES_ENDPOINT or OTEL_EXPORTER_OTLP_ENDPOINT are defined,
// the default STDOUT provider is returned.
func DefaultTracerProvider(ctx context.Context, res *sdkresource.Resource) (*sdktrace.TracerProvider, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		return DefaultTracerProviderOTLP(ctx, res)
	}

	return DefaultTracerProviderStdout(ctx, res)
}

// DefaultMeterProviderWithExporter provides a meter provider for exp.
func DefaultMeterProviderWithExporter(
	res *sdkresource.Resource,
	exp sdkmetric.Exporter,
) *sdkmetric.MeterProvider {
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exp,
				sdkmetric.WithInterval(metricIntervalSec*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)
}

// DefaultMeterProviderStdout provides a default STDOUT OpenTelemetry Meter Provider.
func DefaultMeterProviderStdout(_ context.Context, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error) {
	exp, err := stdoutmetric.New()

	return DefaultMeterProviderWithExporter(res, exp), err
}

// DefaultMeterProviderOTLP provides a default OTLP OpenTelemetry Meter Provider.
// The endpoint is defined by (in order of priority):
//   - OTEL_EXPORTER_OTLP_METRICS_ENDPOINT
//   - OTEL_EXPORTER_OTLP_ENDPOINT
//   - "localhost:4318"
func DefaultMeterProviderOTLP(ctx context.Context, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error) {
	exp, err := otlpmetrichttp.New(ctx)

	return DefaultMeterProviderWithExporter(res, exp), err
}

// DefaultMeterProvider provides a default OTLP OpenTelemetry Meter Provider.
// If neither OTEL_EXPORTER_OTLP_METRICS_ENDPOINT or OTEL_EXPORTER_OTLP_ENDPOINT are defined,
// the default STDOUT provider is returned.
func DefaultMeterProvider(ctx context.Context, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		return DefaultMeterProviderOTLP(ctx, res)
	}

	return DefaultMeterProviderStdout(ctx, res)
}

// TraceID returns the trace ID associated with ctx.
func TraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)

	if spanCtx.HasTraceID() {
		traceID := spanCtx.TraceID()

		return traceID.String()
	}

	return ""
}

// ContextWithSpanContext injects a span context (including trace ID) into the context.
func ContextWithSpanContext(ctx context.Context, traceID trace.TraceID, spanID trace.SpanID) context.Context {
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	return trace.ContextWithSpanContext(ctx, spanCtx)
}
