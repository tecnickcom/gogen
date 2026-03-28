/*
Package httpreverseproxy provides a reusable HTTP reverse proxy wrapper.

It solves the problem of forwarding incoming requests to upstream servers while
standardizing logging, error handling, and proxy behavior across services.

The package wraps the standard net/http/httputil ReverseProxy (or equivalent)
and adds the common plumbing needed for production proxies.

Top features:
- request forwarding with upstream response proxying
- centralized logging and error handling
- middleware-friendly proxy configuration

Benefits:
- reduce boilerplate when building reverse proxy endpoints
- keep proxy behavior consistent across services
- improve observability of proxied traffic
*/
package httpreverseproxy
