package opentel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
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
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceBatchTimeoutSec = 5
	metricIntervalSec    = 60
)

const (
	labelCode      = "code"
	labelDBname    = "db.name"
	labelLevel     = "level"
	labelOperation = "operation"
	labelTask      = "task"
)

const (
	// NameErrorMeter is the name of the meter that counts errors statistics.
	NameErrorMeter = "errors"

	// NameErrorLevel is the name of the collector that counts the number of errors for each log severity level.
	NameErrorLevel = "level"

	// NameErrorCode is the name of the collector that counts the number of errors by task, operation and error code.
	NameErrorCode = "code"
)

const (
	otelExporterOtlpTracesEndpoint  = "localhost:4318"
	otelExporterOtlpMetricsEndpoint = "localhost:4318"
)

// TShutdownFuncs is a type alias for the OpenTelemetry shutdown functions.
type TShutdownFuncs = func(ctx context.Context) error

// SDKResourceFunc is a function that returns an SDK Resource.
type SDKResourceFunc = func(ctx context.Context, name, version string) *sdkresource.Resource

// TraceProviderFunc is a function that returns an SDK Trace Provider.
type TraceProviderFunc = func(ctx context.Context, res *sdkresource.Resource) *sdktrace.TracerProvider

// MetricProviderFunc is a function that returns an SDK Metr Provider.
type MetricProviderFunc = func(ctx context.Context, res *sdkresource.Resource) *sdkmetric.MeterProvider

// Client represents the state type of this client.
type Client struct {
	propagator          propagation.TextMapPropagator
	resFn               SDKResourceFunc
	res                 *sdkresource.Resource
	tracerProviderFn    TraceProviderFunc
	tracerProvider      *sdktrace.TracerProvider
	meterProviderFn     MetricProviderFunc
	meterProvider       *sdkmetric.MeterProvider
	shutdownFuncs       []TShutdownFuncs
	collectorErrorLevel metric.Int64Counter
	collectorErrorCode  metric.Int64Counter
}

// New creates a new metrics instance.
func New(ctx context.Context, name, version string, opts ...Option) (*Client, error) {
	c := initClient()

	for _, applyOpt := range opts {
		err := applyOpt(c)
		if err != nil {
			return nil, err
		}
	}

	err := c.set(ctx, name, version)

	return c, err
}

// initClient returns a Client instance with default values.
func initClient() *Client {
	return &Client{}
}

// InstrumentDB wraps a sql.DB to collect metrics.
func (c *Client) InstrumentDB(dbName string, db *sql.DB) error {
	reg, err := defRegisterDBStatsMetrics(
		db,
		otelsql.WithAttributes(
			attribute.String(labelDBname, dbName),
		),
		otelsql.WithMeterProvider(c.meterProvider),
		otelsql.WithTracerProvider(c.tracerProvider),
	)
	if err != nil {
		return fmt.Errorf("failed instrumenting the database: %w", err)
	}

	c.shutdownFuncs = append(
		c.shutdownFuncs,
		func(_ context.Context) error {
			return reg.Unregister()
		},
	)

	return nil
}

func defRegisterDBStatsMetrics(db *sql.DB, opts ...otelsql.Option) (metric.Registration, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	return otelsql.RegisterDBStatsMetrics(db, opts...) //nolint:wrapcheck
}

// InstrumentHandler returns the input handler.
func (c *Client) InstrumentHandler(path string, handler http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(handler, path, otelhttp.WithMeterProvider(c.meterProvider))
}

// InstrumentRoundTripper returns the input Roundtripper.
func (c *Client) InstrumentRoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return otelhttp.NewTransport(next)
}

// MetricsHandlerFunc returns an http handler function.
// NOTE: OpenTelemetry metrics are typically exported via exporters.
// The handler will return just "OK".
func (c *Client) MetricsHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`OK`)) }
}

// IncLogLevelCounter counts the number of errors for each log severity level.
func (c *Client) IncLogLevelCounter(level string) {
	c.collectorErrorLevel.Add(
		context.TODO(),
		1,
		metric.WithAttributes(
			attribute.String(labelLevel, level),
		))
}

// IncErrorCounter increments the number of errors by task, operation and error code.
func (c *Client) IncErrorCounter(task, operation, code string) {
	c.collectorErrorCode.Add(
		context.TODO(),
		1,
		metric.WithAttributes(
			attribute.String(labelTask, task),
			attribute.String(labelOperation, operation),
			attribute.String(labelCode, code),
		),
	)
}

// Close invokes shutdown cleanup functions registered during setup.
func (c *Client) Close() error {
	return c.CloseCtx(context.TODO())
}

// CloseCtx invokes context-aware shutdown cleanup functions registered during setup.
func (c *Client) CloseCtx(ctx context.Context) error {
	var err error

	for _, fn := range c.shutdownFuncs {
		err = errors.Join(err, fn(ctx))
	}

	c.shutdownFuncs = nil

	return err
}

func (c *Client) set(ctx context.Context, name, version string) error {
	// propagator
	if c.propagator == nil {
		c.propagator = DefaultPropagator()
	}

	otel.SetTextMapPropagator(c.propagator)

	// SDK resource
	if c.resFn == nil {
		c.resFn = DefaultSDKResource
	}

	c.res = c.resFn(ctx, name, version)

	// trace provider
	if c.tracerProviderFn == nil {
		c.tracerProviderFn = DefaultTracerProviderStdout
	}

	c.tracerProvider = c.tracerProviderFn(ctx, c.res)

	c.shutdownFuncs = append(c.shutdownFuncs, c.tracerProvider.Shutdown)
	otel.SetTracerProvider(c.tracerProvider)

	// meter provider
	if c.meterProviderFn == nil {
		c.meterProviderFn = DefaultMeterProviderStdout
	}

	c.meterProvider = c.meterProviderFn(ctx, c.res)

	c.shutdownFuncs = append(c.shutdownFuncs, c.meterProvider.Shutdown)
	otel.SetMeterProvider(c.meterProvider)

	errMeter := c.meterProvider.Meter(NameErrorMeter)
	cel, erra := c.setInt64Counter(ctx, errMeter, NameErrorLevel)
	cec, errb := c.setInt64Counter(ctx, errMeter, NameErrorCode)

	c.collectorErrorLevel = cel
	c.collectorErrorCode = cec

	return errors.Join(erra, errb)
}

func (c *Client) setInt64Counter(ctx context.Context, errMeter metric.Meter, name string) (metric.Int64Counter, error) {
	counter, err := errMeter.Int64Counter(name)
	if err != nil {
		return nil, errors.Join(err, c.CloseCtx(ctx))
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
func DefaultSDKResource(ctx context.Context, name, version string) *sdkresource.Resource {
	res, _ := sdkresource.New(
		ctx,
		sdkresource.WithAttributes(
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
		),
	)

	return res
}

// DefaultTracerProviderWithExporter provides a default TracerProvider for the given the trace exporter.
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
func DefaultTracerProviderStdout(_ context.Context, res *sdkresource.Resource) *sdktrace.TracerProvider {
	exp, _ := stdouttrace.New() // no error

	return DefaultTracerProviderWithExporter(res, exp)
}

// DefaultTracerProviderOTLP provides a default OTLP OpenTelemetry Tracer Provider.
func DefaultTracerProviderOTLP(ctx context.Context, res *sdkresource.Resource) *sdktrace.TracerProvider {
	exp, _ := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(otelExporterOtlpTracesEndpoint), // OTEL_EXPORTER_OTLP_TRACES_ENDPOINT env var will take precedence.
		otlptracehttp.WithInsecure(),
	) // no error

	return DefaultTracerProviderWithExporter(res, exp)
}

// DefaultMeterProviderWithExporter provides a MeterProvider for the given the metric exporter.
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
func DefaultMeterProviderStdout(_ context.Context, res *sdkresource.Resource) *sdkmetric.MeterProvider {
	exp, _ := stdoutmetric.New() // returned error is always nil

	return DefaultMeterProviderWithExporter(res, exp)
}

// DefaultMeterProviderOTLP provides a default OTLP OpenTelemetry Meter Provider.
func DefaultMeterProviderOTLP(ctx context.Context, res *sdkresource.Resource) *sdkmetric.MeterProvider {
	exp, _ := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithEndpoint(otelExporterOtlpMetricsEndpoint), // OTEL_EXPORTER_OTLP_METRICS_ENDPOINT env var will take precedence.
		otlpmetrichttp.WithInsecure(),
	) // no error

	return DefaultMeterProviderWithExporter(res, exp)
}

// TraceID returns the trace ID associate with the context.
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
