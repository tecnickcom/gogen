package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
)

func TestWithHandlerOpts(t *testing.T) {
	c := initClient()
	opt := promhttp.HandlerOpts{EnableOpenMetrics: true}
	err := WithHandlerOpts(opt)(c)
	require.NoError(t, err)
	require.True(t, c.handlerOpts.EnableOpenMetrics, "expecting EnableOpenMetrics to be true")
}

func TestWithCollector(t *testing.T) {
	c := initClient()
	m := prometheus.NewGoCollector()
	err := WithCollector(m)(c)
	require.NoError(t, err)
}
