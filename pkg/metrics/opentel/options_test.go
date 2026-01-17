package opentel

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestWithTracerProvider(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := trace.NewTracerProvider()
	err := WithTracerProvider(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.tracerProvider)
}

func TestWithMeterProvider(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := metric.NewMeterProvider()
	err := WithMeterProvider(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.meterProvider)
}
