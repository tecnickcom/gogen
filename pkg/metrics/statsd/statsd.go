/*
Package statsd implements the metrics interface for StatsD.

The metrics (counters and timers) are actively sent over UDP or TCP to a central
StatsD server. StatsD is a network daemon that listens for statistics and
aggregates them to one or more pluggable backend services (e.g., Graphite).

This package is based on github.com/tecnickcom/statsd.
*/
package statsd
