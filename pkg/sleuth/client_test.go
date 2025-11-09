//go:generate go tool mockgen -write_package_comment=false -package sleuth -destination ./mock_test.go . HTTPClient
package sleuth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/httpretrier"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/undefinedlabs/go-mpatch"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		addr        string
		org         string
		apikey      string
		opts        []Option
		wantTimeout time.Duration
		wantErr     bool
	}{
		{
			name:    "fails with invalid character in URL",
			addr:    "http://invalid-url.domain.invalid\u007F",
			org:     "testorg",
			apikey:  "0123456789abcdef",
			wantErr: true,
		},
		{
			name:    "fails with empty org",
			addr:    "http://service.domain.invalid:1234",
			org:     "",
			apikey:  "0123456789abcdef",
			wantErr: true,
		},
		{
			name:    "fails with empty api key",
			addr:    "http://service.domain.invalid:1234",
			org:     "testorg",
			apikey:  "",
			wantErr: true,
		},
		{
			name:        "succeeds with defaults",
			addr:        "http://service.domain.invalid:1234",
			org:         "testorg",
			apikey:      "0123456789abcdef",
			wantTimeout: defaultPingTimeout,
			wantErr:     false,
		},
		{
			name:        "succeeds with options",
			addr:        "http://service.domain.invalid:1234",
			org:         "testorg",
			apikey:      "0123456789abcdef",
			opts:        []Option{WithPingTimeout(2 * time.Second)},
			wantTimeout: 2 * time.Second,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.opts = append(tt.opts, WithRetryAttempts(1))

			c, err := New(
				tt.addr,
				tt.org,
				tt.apikey,
				tt.opts...,
			)

			if tt.wantErr {
				require.Nil(t, c, "New() returned client should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, c, "New() returned client should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
			require.Equal(t, tt.wantTimeout, c.pingTimeout, "New() unexpected pingTimeout = %d got %d", tt.wantTimeout, c.pingTimeout)
		})
	}
}

//nolint:gocognit
func TestClient_HealthCheck(t *testing.T) {
	t.Parallel()

	timeout := 100 * time.Millisecond

	tests := []struct {
		name                  string
		pingHandlerDelay      time.Duration
		pingHandlerStatusCode int
		pingURL               string
		pingBody              string
		bodyErr               bool
		wantErr               bool
	}{
		{
			name:                  "fails because ping url error",
			pingHandlerStatusCode: http.StatusOK,
			pingURL:               "%^*&-ERROR",
			pingBody:              regexPatternHealthcheck,
			wantErr:               true,
		},
		{
			name:                  "fails because bad response body",
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              regexPatternHealthcheck,
			bodyErr:               true,
			wantErr:               true,
		},
		{
			name:                  "returns error because of timeout",
			pingHandlerDelay:      timeout + 1,
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              regexPatternHealthcheck,
			wantErr:               true,
		},
		{
			name:                  "returns error from endpoint",
			pingHandlerStatusCode: http.StatusInternalServerError,
			pingBody:              regexPatternHealthcheck,
			wantErr:               true,
		},
		{
			name:                  "fails because bad response body",
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              "error response",
			wantErr:               true,
		},
		{
			name:                  "returns success from endpoint",
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              regexPatternHealthcheck,
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if tt.pingHandlerDelay != 0 {
					time.Sleep(tt.pingHandlerDelay)
				}

				if tt.bodyErr {
					w.Header().Set("Content-Length", "1")
				}

				httputil.SendText(r.Context(), w, tt.pingHandlerStatusCode, tt.pingBody)
			})

			ts := httptest.NewServer(mux)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"testorg",
				"0123456789abcdef",
				WithRetryAttempts(1),
				WithTimeout(timeout),
				WithPingTimeout(timeout),
			)
			require.NoError(t, err, "Client.HealthCheck() create client unexpected error = %v", err)

			if tt.pingURL != "" {
				c.pingURL = tt.pingURL
			}

			err = c.HealthCheck(t.Context())
			if tt.wantErr {
				require.Error(t, err, "Client.HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.NoError(t, err, "Client.HealthCheck() unexpected error = %v", err)
			}
		})
	}
}

func TestClient_newWriteHTTPRetrier(t *testing.T) {
	t.Parallel()

	c, err := New(
		"https://test.invalid",
		"testorg",
		"0123456789abcdef",
		WithRetryAttempts(1),
	)
	require.NoError(t, err)

	hr, err := c.newWriteHTTPRetrier()

	require.NoError(t, err)
	require.NotNil(t, hr)
}

func Test_httpRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		urlStr  string
		req     any
		wantErr bool
	}{
		{
			name:    "fail input validation",
			urlStr:  "https://test.invalid",
			req:     make(chan int), // this payload can't be encoded in JSON
			wantErr: true,
		},
		{
			name:    "fail invalid URL",
			urlStr:  "%^*&-ERROR",
			req:     make(chan int), // this payload can't be encoded in JSON
			wantErr: true,
		},
		{
			name:    "success",
			urlStr:  "https://test.invalid",
			req:     "test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := httpPostRequest(t.Context(), tt.urlStr, "0123456789abcdef", tt.req)

			if !tt.wantErr {
				require.NoError(t, err)
				require.NotNil(t, r)
			} else {
				require.Error(t, err)
				require.Nil(t, r)
			}
		})
	}
}

//go:noinline
func newRequestWithContextPatch(_ context.Context, _, _ string, _ io.Reader) (*http.Request, error) {
	return nil, errors.New("ERROR: newRequestWithContextPatch")
}

//go:noinline
func newHTTPRetrierPatch(httpretrier.HTTPClient, ...httpretrier.Option) (*httpretrier.HTTPRetrier, error) {
	return nil, errors.New("ERROR: newHTTPRetrierPatch")
}

//nolint:gocognit,paralleltest
func Test_sendRequest(t *testing.T) {
	tests := []struct {
		name              string
		req               *DeployRegistrationRequest
		createMockHandler func(t *testing.T) http.HandlerFunc
		setupMocks        func(client *MockHTTPClient)
		setupPatches      func() (*mpatch.Patch, error)
		wantErr           bool
	}{
		{
			name: "failed to execute request - transport error",
			setupMocks: func(m *MockHTTPClient) {
				m.EXPECT().Do(gomock.Any()).Return(nil, errors.New("transport error")).Times(1)
			},
			wantErr: true,
		},
		{
			name: "failed to execute request - NewRequest error",
			setupPatches: func() (*mpatch.Patch, error) {
				patch, err := mpatch.PatchMethod(http.NewRequestWithContext, newRequestWithContextPatch)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				_ = patch.Patch()

				return patch, nil
			},
			wantErr: true,
		},
		{
			name: "failed to execute request - HTTPRetrier error",
			setupPatches: func() (*mpatch.Patch, error) {
				patch, err := mpatch.PatchMethod(httpretrier.New, newHTTPRetrierPatch)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				_ = patch.Patch()

				return patch, nil
			},
			wantErr: true,
		},
		{
			name: "unexpected http error status code",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()

				return func(w http.ResponseWriter, r *http.Request) {
					httputil.SendStatus(r.Context(), w, http.StatusInternalServerError)
				}
			},
			wantErr: true,
		},
		{
			name: "invalid response status < 200",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()

				return func(w http.ResponseWriter, r *http.Request) {
					httputil.SendText(r.Context(), w, http.StatusSwitchingProtocols, "")
				}
			},
			wantErr: true,
		},
		{
			name: "fail input validation",
			req: &DeployRegistrationRequest{
				Deployment: "test_deployment_error",
			},
			wantErr: true,
		},
		{
			name: "success valid response",
			createMockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()

				return func(w http.ResponseWriter, r *http.Request) {
					httputil.SendText(r.Context(), w, http.StatusOK, "Success")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			urlTestPath := "/test"

			mux := http.NewServeMux()
			if tt.createMockHandler != nil {
				mux.HandleFunc(urlTestPath, tt.createMockHandler(t))
			}

			ts := httptest.NewServer(mux)
			defer ts.Close()

			clientOpts := []Option{}

			if tt.setupMocks != nil {
				mc := NewMockHTTPClient(ctrl)
				tt.setupMocks(mc)
				clientOpts = append(clientOpts, WithHTTPClient(mc), WithRetryAttempts(1))
			}

			c, err := New(
				ts.URL,
				"testorg",
				"0123456789abcdef",
				clientOpts...,
			)
			require.NoError(t, err)

			if tt.setupPatches != nil {
				patch, err := tt.setupPatches()
				require.NoError(t, err)

				defer func() {
					_ = patch.Unpatch()
				}()
			}

			if tt.req == nil {
				tt.req = &DeployRegistrationRequest{
					Deployment: "test_deployment",
					Sha:        "96086c3354a0475073837a24a7fa95a5eb42aab9",
				}
			}

			err = sendRequest(t.Context(), c, ts.URL+urlTestPath, tt.req)
			require.Equal(t, tt.wantErr, err != nil, "error: %v", err)
		})
	}
}

func getTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	createMockHandler := func(t *testing.T) http.HandlerFunc {
		t.Helper()

		return func(w http.ResponseWriter, r *http.Request) {
			httputil.SendText(r.Context(), w, http.StatusOK, "Success")
		}
	}

	mux.HandleFunc("/", createMockHandler(t))

	return httptest.NewServer(mux)
}

func TestClient_SendDeployRegistration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *DeployRegistrationRequest
		wantErr bool
	}{
		{
			name:    "fail with empty request",
			req:     &DeployRegistrationRequest{},
			wantErr: true,
		},
		{
			name: "fail with invalid tags",
			req: &DeployRegistrationRequest{
				Deployment: "test_deployment",
				Sha:        "96086c3354a0475073837a24a7fa95a5eb42aab9",
				Tags: []string{
					"alpha",
					"beta",
				},
			},
			wantErr: true,
		},
		{
			name: "success with required fields",
			req: &DeployRegistrationRequest{
				Deployment: "test_deployment",
				Sha:        "96086c3354a0475073837a24a7fa95a5eb42aab9",
			},
			wantErr: false,
		},
		{
			name: "success with all fields set",
			req: &DeployRegistrationRequest{
				Deployment:  "test_deployment",
				Sha:         "96086c3354a0475073837a24a7fa95a5eb42aab9",
				Environment: "test",
				Date:        "2023-04-24 12:20:00",
				Tags: []string{
					"#alpha",
					"#beta",
				},
				IgnoreIfDuplicate: true,
				Email:             "test@example.invalid",
				Links: map[string]string{
					"one": "https://test.one.invalid",
					"two": "https://test.two.invalid",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := getTestServer(t)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"testorg",
				"0123456789abcdef",
				WithRetryAttempts(1),
			)
			require.NoError(t, err)

			err = c.SendDeployRegistration(t.Context(), tt.req)
			require.Equal(t, tt.wantErr, err != nil, "error: %v", err)
		})
	}
}

func TestClient_SendManualChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *ManualChangeRequest
		wantErr bool
	}{
		{
			name:    "fail with empty request",
			req:     &ManualChangeRequest{},
			wantErr: true,
		},
		{
			name: "fail with invalid tags",
			req: &ManualChangeRequest{
				Project: "test_project",
				Name:    "test_name",
				Tags: []string{
					"alpha",
					"beta",
				},
			},
			wantErr: true,
		},
		{
			name: "success with required fields",
			req: &ManualChangeRequest{
				Project: "test_project",
				Name:    "test_name",
			},
			wantErr: false,
		},
		{
			name: "success with all fields set",
			req: &ManualChangeRequest{
				Project:     "test_project",
				Name:        "test_name",
				Description: "test_description",
				Environment: "test",
				Tags: []string{
					"#alpha",
					"#beta",
				},
				Author: "author@example.invalid",
				Email:  "test@example.invalid",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := getTestServer(t)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"testorg",
				"0123456789abcdef",
				WithRetryAttempts(1),
			)
			require.NoError(t, err)

			err = c.SendManualChange(t.Context(), tt.req)
			require.Equal(t, tt.wantErr, err != nil, "error: %v", err)
		})
	}
}

func TestClient_SendCustomIncidentImpactRegistration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *CustomIncidentImpactRegistrationRequest
		wantErr bool
	}{
		{
			name:    "fail with empty request",
			req:     &CustomIncidentImpactRegistrationRequest{},
			wantErr: true,
		},
		{
			name: "fail with invalid type",
			req: &CustomIncidentImpactRegistrationRequest{
				Project:      "test_project",
				Environment:  "test",
				ImpactSource: "test_impact_source",
				Type:         "invalid",
			},
			wantErr: true,
		},
		{
			name: "success with required fields",
			req: &CustomIncidentImpactRegistrationRequest{
				Project:      "test_project",
				Environment:  "test",
				ImpactSource: "test_impact_source",
				Type:         Triggered,
			},
			wantErr: false,
		},
		{
			name: "success with all fields set",
			req: &CustomIncidentImpactRegistrationRequest{
				Project:      "test_project",
				Environment:  "test",
				ImpactSource: "test_impact_source",
				Type:         Triggered,
				ID:           "abcdef0123456789",
				Date:         "2023-04-24 13:00:00",
				EndedDate:    "2023-04-24 14:10:00",
				Title:        "test_incident_title",
				URL:          "http://test.external.url.invalid",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := getTestServer(t)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"testorg",
				"0123456789abcdef",
				WithRetryAttempts(1),
			)
			require.NoError(t, err)

			err = c.SendCustomIncidentImpactRegistration(t.Context(), tt.req)
			require.Equal(t, tt.wantErr, err != nil, "error: %v", err)
		})
	}
}

func TestClient_SendCustomMetricImpactRegistration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *CustomMetricImpactRegistrationRequest
		wantErr bool
	}{
		{
			name:    "fail with empty request",
			req:     &CustomMetricImpactRegistrationRequest{},
			wantErr: true,
		},
		{
			name: "fail with invalid date",
			req: &CustomMetricImpactRegistrationRequest{
				ImpactID: 3451,
				Value:    123.4561,
				Date:     "error_date",
			},
			wantErr: true,
		},
		{
			name: "success with required fields",
			req: &CustomMetricImpactRegistrationRequest{
				ImpactID: 3452,
				Value:    123.4562,
			},
			wantErr: false,
		},
		{
			name: "success with all fields set",
			req: &CustomMetricImpactRegistrationRequest{
				ImpactID: 3453,
				Value:    123.4563,
				Date:     "2023-04-24 13:14:15",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := getTestServer(t)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"testorg",
				"0123456789abcdef",
				WithRetryAttempts(1),
			)
			require.NoError(t, err)

			err = c.SendCustomMetricImpactRegistration(t.Context(), tt.req)
			require.Equal(t, tt.wantErr, err != nil, "error: %v", err)
		})
	}
}
