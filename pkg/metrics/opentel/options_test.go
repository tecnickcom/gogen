package opentel

import (
	"testing"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestWithTracerProvider(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := sdktrace.NewTracerProvider()
	err := WithTracerProvider(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.tracerProvider)
}

func TestWithMeterProvider(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := sdkmetric.NewMeterProvider()
	err := WithMeterProvider(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.meterProvider)
}
