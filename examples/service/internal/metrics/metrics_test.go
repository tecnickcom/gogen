package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	m := New()
	require.NotNil(t, m, "Metrics should not be nil")
	require.NotNil(t, m.collectorExample, "collectorExample not be nil")
}

func TestCreateMetricsClientFunc(t *testing.T) {
	t.Parallel()

	m := New()
	c, err := m.CreateMetricsClientFunc()
	require.NoError(t, err, "CreateMetricsClientFunc() unexpected error = %v", err)
	require.NotNil(t, c, "metrics.Client should not be nil")
}

func TestIncExampleCounter(t *testing.T) {
	t.Parallel()

	m := New()
	i := testutil.CollectAndCount(m.collectorExample, NameExample)
	require.Equal(t, 0, i, "failed to assert right metrics: got %v want %v", i, 0)
	m.IncExampleCounter("test")
	i = testutil.CollectAndCount(m.collectorExample, NameExample)
	require.Equal(t, 1, i, "failed to assert right metrics: got %v want %v", i, 1)
}
