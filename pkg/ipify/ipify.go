/*
Package ipify provides a small client to resolve the current instance public IP
address using the ipify service (https://www.ipify.org/).

# Problem

Services running behind NAT, cloud load balancers, or dynamic outbound egress
often need to discover their externally visible IP address at runtime (for
allow-list updates, diagnostics, or registration workflows). Implementing this
from scratch requires HTTP setup, timeout handling, status validation, and
error fallback logic.

# Solution

This package wraps those concerns in a focused [Client] API:
  - [New] creates a configurable ipify client.
  - [Client.GetPublicIP] performs the request and returns the resolved IP.
  - [Client.HealthCheck] probes endpoint reachability for parity with the other
    gogen HTTP clients.

The default configuration uses:
  - endpoint: https://api.ipify.org
  - timeout:  4 seconds

Use [WithURL], [WithTimeout], [WithHTTPClient], and [WithErrorIP] to adapt the
client to custom endpoints, transport stacks, and fallback policies. A
non-positive timeout is clamped to the default; an API URL that is missing,
unparseable, or not http/https with a host is rejected by [New].

# Error Fallback Behavior

When request creation, transport, status-code validation, or body reading fails,
[Client.GetPublicIP] returns the configured error-IP value together with the
error. By default the fallback string is empty, but it can be set (for example
to "0.0.0.0") via [WithErrorIP]. The response body is whitespace-trimmed and an
empty body is treated as a failure.

# Errors

Configuration problems are reported at construction time with errors matching
the exported sentinel [ErrInvalidOptions]. A nil or empty response body from the
endpoint surfaces as [ErrInvalidResponse]. Match the sentinels with errors.Is.

# IPv6 Note

The default endpoint resolves standard ipify behavior. To force IPv6-capable
resolution, point the client to `https://api64.ipify.org` via [WithURL].

# Benefits

ipify gives applications a minimal, testable, and timeout-safe way to discover
public IP information without duplicating HTTP boilerplate.
*/
package ipify
