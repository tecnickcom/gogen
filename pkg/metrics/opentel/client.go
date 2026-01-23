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

// TShutdownFuncs is a type alias for the OpenTelemetry shutdown functions.
type TShutdownFuncs = func(ctx context.Context) error

// Client represents the state type of this client.
type Client struct {
	tracerProvider      *sdktrace.TracerProvider
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
	otel.SetTextMapPropagator(defaultPropagator())

	res, _ := sdkresource.New(
		ctx,
		sdkresource.WithAttributes(
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
		),
	)

	// trace provider
	if c.tracerProvider == nil {
		tp := defaultTracerProvider(res)
		c.tracerProvider = tp
	}

	c.shutdownFuncs = append(c.shutdownFuncs, c.tracerProvider.Shutdown)
	otel.SetTracerProvider(c.tracerProvider)

	// meter provider
	if c.meterProvider == nil {
		mp := defaultMeterProvider(res)
		c.meterProvider = mp
	}

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

// defaultPropagator provides a default OpenTelemetry TextMapPropagator.
func defaultPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// defaultTracerProvider provides a default OpenTelemetry TracerProvider.
func defaultTracerProvider(res *sdkresource.Resource) *sdktrace.TracerProvider {
	traceExporter, _ := stdouttrace.New() // no error

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			traceExporter,
			sdktrace.WithBatchTimeout(traceBatchTimeoutSec*time.Second),
		),
		sdktrace.WithResource(res),
	)
}

// defaultMeterProvider provides a default OpenTelemetry MeterProvider.
func defaultMeterProvider(res *sdkresource.Resource) *sdkmetric.MeterProvider {
	metricExporter, _ := stdoutmetric.New() // returned error is always nil

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExporter,
				sdkmetric.WithInterval(metricIntervalSec*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)
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
