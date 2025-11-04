package prometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
)

func TestWithHandlerOpts(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := promhttp.HandlerOpts{EnableOpenMetrics: true}
	err := WithHandlerOpts(opt)(c)
	require.NoError(t, err)
	require.True(t, c.handlerOpts.EnableOpenMetrics, "expecting EnableOpenMetrics to be true")
}

func TestWithCollector(t *testing.T) {
	t.Parallel()

	c := initClient()
	m := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "optiontest",
			Help: "Option Test collector.",
		},
	)
	err := WithCollector(m)(c)
	require.NoError(t, err)
}

func TestWithInboundRequestSizeBuckets(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := []float64{1, 2, 3}
	err := WithInboundRequestSizeBuckets(opt)(c)
	require.NoError(t, err)
	require.Equal(t, opt, c.inboundRequestSizeBuckets, "expecting %v, got %v", opt, c.inboundRequestSizeBuckets)
}

func TestWithInboundResponseSizeBuckets(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := []float64{4, 5, 6}
	err := WithInboundResponseSizeBuckets(opt)(c)
	require.NoError(t, err)
	require.Equal(t, opt, c.inboundResponseSizeBuckets, "expecting %v, got %v", opt, c.inboundRequestSizeBuckets)
}

func TestWithInboundRequestDurationBuckets(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := []float64{7, 8, 9}
	err := WithInboundRequestDurationBuckets(opt)(c)
	require.NoError(t, err)
	require.Equal(t, opt, c.inboundRequestDurationBuckets, "expecting %v, got %v", opt, c.inboundRequestSizeBuckets)
}

func TestWithOutboundRequestDurationBuckets(t *testing.T) {
	t.Parallel()

	c := initClient()
	opt := []float64{10, 11, 12}
	err := WithOutboundRequestDurationBuckets(opt)(c)
	require.NoError(t, err)
	require.Equal(t, opt, c.outboundRequestDurationBuckets, "expecting %v, got %v", opt, c.inboundRequestSizeBuckets)
}
