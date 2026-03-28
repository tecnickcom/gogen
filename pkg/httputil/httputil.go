/*
Package httputil contains reusable HTTP request and response utility functions
for Go services.

It solves the problem of repeated request/response boilerplate by providing
helpers that simplify query parsing, body handling, header management, and
standardized HTTP response construction.

The package is intended for use alongside the standard net/http package and
provides common helpers that make HTTP handler code cleaner and more consistent.

Top features:
- request query parsing helpers
- request body reading utilities
- response writing and error response helpers
- support for standardized HTTP payload handling

Benefits:
- reduce repetitive HTTP handler code
- improve clarity and consistency in request/response flows
- make it easier to build maintainable web services
*/
package httputil
