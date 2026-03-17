package opentel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithSDKResourceFn(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := DefaultSDKResource
	err := WithSDKResourceFn(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.resFn)
}

func TestWithTracerProviderFn(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := DefaultTracerProviderOTLP
	err := WithTracerProviderFn(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.tracerProviderFn)
}

func TestWithMeterProviderFn(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := DefaultMeterProviderOTLP
	err := WithMeterProviderFn(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.meterProviderFn)
}

func TestWithPropagator(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := DefaultPropagator()
	err := WithPropagator(opt)(c)
	require.NoError(t, err)
	require.NotNil(t, c.propagator)
}
