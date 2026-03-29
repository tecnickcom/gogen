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

The default configuration uses:
  - endpoint: https://api.ipify.org
  - timeout:  4 seconds

Use [WithURL], [WithTimeout], [WithHTTPClient], and [WithErrorIP] to adapt the
client to custom endpoints, transport stacks, and fallback policies.

# Error Fallback Behavior

When request creation, transport, status-code validation, or body reading fails,
[Client.GetPublicIP] returns the configured error-IP value together with the
error. By default the fallback string is empty, but it can be set (for example
to "0.0.0.0") via [WithErrorIP].

# IPv6 Note

The default endpoint resolves standard ipify behavior. To force IPv6-capable
resolution, point the client to `https://api64.ipify.org` via [WithURL].

# Benefits

ipify gives applications a minimal, testable, and timeout-safe way to discover
public IP information without duplicating HTTP boilerplate.
*/
package ipify
