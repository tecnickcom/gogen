package jirasrv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/tecnickcom/nurago/pkg/httpretrier"
	"github.com/tecnickcom/nurago/pkg/httputil"
	"github.com/tecnickcom/nurago/pkg/validator"
)

// Default configuration values.
const (
	defaultTimeout     = 1 * time.Minute
	defaultPingTimeout = 15 * time.Second
	apiBasePath        = "/rest/api/2" // https://docs.atlassian.com/software/jira/docs/api/REST/9.17.0/

	// defaultMaxRetryAfter caps how long a server-provided Retry-After header can
	// stall a retry, so a rate-limited or misconfigured endpoint cannot force a
	// very long wait on a caller that did not set a context deadline.
	defaultMaxRetryAfter = 60 * time.Second

	// maxBodyBytes caps how much of the health-check response body is drained for
	// keep-alive connection reuse, bounding time if the endpoint returns a very
	// large body.
	maxBodyBytes = 4 << 10 // 4 KiB
)

// newValidator is a package-level indirection so tests can force a validator
// setup failure; in production it always delegates to [validator.New].
var newValidator = validator.New //nolint:gochecknoglobals

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// Client wraps Jira Server REST API operations with validation and retries.
type Client struct {
	httpClient    HTTPClient
	readRetrier   *httpretrier.HTTPRetrier
	writeRetrier  *httpretrier.HTTPRetrier
	apiURL        *url.URL
	valid         *validator.Validator
	timeout       time.Duration
	pingTimeout   time.Duration
	retryDelay    time.Duration
	retryAttempts uint
	token         string
	pingAddr      string
}

// New constructs a Jira Server REST client.
//
// It validates base URL and token, initializes validators and endpoint paths,
// applies options, and provisions a default HTTP client when none is provided.
func New(addr, token string, opts ...Option) (*Client, error) {
	baseURL, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrInvalidAddress, addr, err)
	}

	// Reject relative or schemeless addresses at construction time; otherwise
	// every later request fails with an obscure "missing scheme" transport error.
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("%w %q: missing scheme or host", ErrInvalidAddress, addr)
	}

	if token == "" {
		return nil, ErrEmptyToken
	}

	valid, err := newValidator(
		validator.WithFieldNameTag("json"),
		validator.WithCustomValidationTags(validator.CustomValidationTags()),
		validator.WithErrorTemplates(validator.ErrorTemplates()),
	)
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	apiURL := baseURL.JoinPath(apiBasePath)

	c := &Client{
		apiURL:        apiURL,
		valid:         valid,
		timeout:       defaultTimeout,
		pingTimeout:   defaultPingTimeout,
		retryDelay:    httpretrier.DefaultDelay,
		retryAttempts: httpretrier.DefaultAttempts,
		token:         token,
		pingAddr:      apiURL.JoinPath("serverInfo").String() + "?doHealthCheck=true",
	}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	// The retriers hold only immutable configuration, so a single instance per
	// policy is shared by all requests; building them here surfaces invalid retry
	// settings at construction time instead of on every send. Both use identical
	// attempts/delay, so errors.Join reports any invalid setting through one path.
	readRetrier, rerr := c.buildRetrier(httpretrier.RetryIfForReadRequests)
	writeRetrier, werr := c.buildRetrier(httpretrier.RetryIfForWriteRequests)

	jerr := errors.Join(rerr, werr)
	if jerr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRetryConfig, jerr)
	}

	c.readRetrier = readRetrier
	c.writeRetrier = writeRetrier

	return c, nil
}

// HealthCheck verifies Jira server reachability and health endpoint status.
//
// It runs with pingTimeout and returns an error for transport failures or
// unexpected HTTP status codes.
func (c *Client) HealthCheck(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, rerr := c.httpRequest(ctx, http.MethodGet, c.pingAddr, nil)
	if rerr != nil {
		return rerr
	}

	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return fmt.Errorf("healthcheck request: %w", derr)
	}

	defer func() {
		// Drain (capped) so the connection can be reused by keep-alive.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	return nil
}

// SendRequest validates and sends a Jira REST API request.
//
// It builds the target URL from endpoint and query, JSON-encodes request when
// provided, applies auth/content headers, and executes through the configured
// method-aware retrier.
//
// NOTE: The caller must close the response body.
//
// NOTE: GET requests use the idempotent read retry policy
// ([httpretrier.RetryIfForReadRequests]), which retries some codes that are
// often terminal (404/408/409/423/425/429/5xx) to tolerate read-after-write
// eventual consistency. A legitimate 404 on a GET is therefore retried; supply
// a custom client/transport if that is undesirable.
func (c *Client) SendRequest(
	ctx context.Context,
	httpMethod string,
	endpoint string,
	query *url.Values,
	request any,
) (*http.Response, error) {
	var (
		buffer io.Reader
		err    error
	)

	if request != nil {
		buffer, err = c.requestBuffer(ctx, request)
		if err != nil {
			return nil, err
		}
	}

	targetURL := c.apiURL.JoinPath(endpoint)
	if query != nil {
		targetURL.RawQuery = query.Encode()
	}

	r, err := c.httpRequest(ctx, httpMethod, targetURL.String(), buffer)
	if err != nil {
		return nil, err
	}

	return c.retrierForMethod(httpMethod).Do(r) //nolint:wrapcheck
}

// requestBuffer validates request and JSON-encodes it into a buffer.
//
// Only struct payloads (or non-nil pointers to structs) carry validation tags,
// so validation is skipped for other JSON-encodable payloads (maps, slices,
// strings), which the validator would otherwise reject as invalid input.
func (c *Client) requestBuffer(
	ctx context.Context,
	request any,
) (*bytes.Buffer, error) {
	if isValidatable(request) {
		err := c.valid.ValidateStructCtx(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("invalid request: %w", err)
		}
	}

	buf := &bytes.Buffer{}

	err := json.NewEncoder(buf).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	return buf, nil
}

// isValidatable reports whether v is a struct or a non-nil pointer chain
// leading to a struct, i.e. an input the struct validator can process.
func isValidatable(v any) bool {
	rv := reflect.ValueOf(v)

	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return false
		}

		rv = rv.Elem()
	}

	return rv.Kind() == reflect.Struct
}

// httpRequest builds an HTTP request with Jira default headers attached.
func (c *Client) httpRequest(
	ctx context.Context,
	httpMethod string,
	urlStr string,
	request io.Reader,
) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, httpMethod, urlStr, request)
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	c.setRequestHeaders(r)

	return r, nil
}

// setRequestHeaders attaches JSON and bearer-token headers to r.
func (c *Client) setRequestHeaders(r *http.Request) {
	httputil.AddJSONHeaders(r)
	httputil.AddBearerToken(c.token, r)
}

// buildRetrier constructs a shared retrier for the given retry policy, applying
// the client's configured attempts/delay and Retry-After handling. The result
// is immutable, so a single instance per policy is reused across all requests.
func (c *Client) buildRetrier(retryIf httpretrier.RetryIfFn) (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(retryIf),
		httpretrier.WithAttempts(c.retryAttempts),
		httpretrier.WithDelay(c.retryDelay),
		httpretrier.WithRespectRetryAfter(),
		httpretrier.WithMaxRetryAfter(defaultMaxRetryAfter),
	)
}

// retrierForMethod returns the prebuilt retrier matching the HTTP method's
// idempotency: the read policy for GET, the write policy otherwise (mirroring
// [httpretrier.RetryIfFnByHTTPMethod]).
func (c *Client) retrierForMethod(httpMethod string) *httpretrier.HTTPRetrier {
	if httpMethod == http.MethodGet {
		return c.readRetrier
	}

	return c.writeRetrier
}
