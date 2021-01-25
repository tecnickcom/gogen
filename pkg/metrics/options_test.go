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
	name := "TestWithCollector"
	m := prometheus.NewGoCollector()
	err := WithCollector(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.Collector, 1, "Expecting only 1 collector")
}

func TestWithCollectorGauge(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorGauge"
	m := prometheus.NewGauge(prometheus.GaugeOpts{Name: name})
	err := WithCollectorGauge(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorGauge, 1, "Expecting only 1 collector")
}

func TestWithCollectorCounter(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorCounter"
	m := prometheus.NewCounter(prometheus.CounterOpts{Name: name})
	err := WithCollectorCounter(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorCounter, 1, "Expecting only 1 collector")
}

func TestWithCollectorSummary(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorSummary"
	m := prometheus.NewSummary(prometheus.SummaryOpts{Name: name})
	err := WithCollectorSummary(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorSummary, 1, "Expecting only 1 collector")
}

func TestWithCollectorHistogram(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorHistogram"
	m := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name})
	err := WithCollectorHistogram(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorHistogram, 1, "Expecting only 1 collector")
}

func TestWithCollectorGaugeVec(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorGaugeVec"
	m := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, []string{})
	err := WithCollectorGaugeVec(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorGaugeVec, 1, "Expecting only 1 collector")
}

func TestWithCollectorCounterVec(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorCounterVec"
	m := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name}, []string{})
	err := WithCollectorCounterVec(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorCounterVec, 1, "Expecting only 1 collector")
}

func TestWithCollectorSummaryVec(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorSummaryVec"
	m := prometheus.NewSummaryVec(prometheus.SummaryOpts{Name: name}, []string{})
	err := WithCollectorSummaryVec(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorSummaryVec, 1, "Expecting only 1 collector")
}

func TestWithCollectorHistogramVec(t *testing.T) {
	c := initClient()
	name := "TestWithCollectorHistogramVec"
	m := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, []string{})
	err := WithCollectorHistogramVec(name, m)(c)
	require.NoError(t, err)
	require.Len(t, c.CollectorHistogramVec, 1, "Expecting only 1 collector")
}
