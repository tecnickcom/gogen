/*
Package statsd implements [github.com/tecnickcom/nurago/pkg/metrics.Client]
using the StatsD protocol.

It emits counters, gauges, and timers over UDP or TCP to a StatsD daemon, which
then forwards aggregated metrics to downstream systems (such as
Graphite-compatible storage). Application code depends on the shared metrics
interface rather than the backend-specific client.

Use [New] to create a client and configure it with options like [WithPrefix],
[WithNetwork], [WithAddress], and [WithFlushPeriod].

# Behavior Notes

StatsD is push-only in this implementation, so [Client.MetricsHandlerFunc]
returns HTTP 501 (Not Implemented). Database instrumentation via
[Client.InstrumentDB] is currently a no-op.

This package is based on github.com/tecnickcom/statsd.
*/
package statsd
