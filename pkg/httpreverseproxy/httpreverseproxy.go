/*
Package httpreverseproxy provides an HTTP Reverse Proxy that takes an incoming
request and sends it to another server, proxying the response back to the
client. It wraps the standard net/http/httputil ReverseProxy (or equivalent)
with common functionalities, including logging and error handling.
*/
package httpreverseproxy
