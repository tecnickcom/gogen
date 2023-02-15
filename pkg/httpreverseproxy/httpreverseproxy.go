// Package httpreverseproxy provides an HTTP Reverse Proxy that takes an incoming request and sends it to another server, proxying the response back to the client.
// It wraps the net/http/httputil ReverseProxy with common functionalities.
// It uses the internal github.com/Vonage/gosrvlib/pkg/logging package ("go.uber.org/zap") for error logs.
package httpreverseproxy
