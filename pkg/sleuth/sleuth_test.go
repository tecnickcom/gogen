//go:generate mockgen -package sleuth -destination ./mock_test.go . HTTPClient
package sleuth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/httpretrier"
	"github.com/Vonage/gosrvlib/pkg/httputil"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/undefinedlabs/go-mpatch"
)

//go:noinline
func newRequestWithContextPatch(_ context.Context, _, _ string, _ io.Reader) (*http.Request, error) {
	return nil, fmt.Errorf("error")
}

//go:noinline
func newHTTPRetrierPatch(httpretrier.HTTPClient, ...httpretrier.Option) (*httpretrier.HTTPRetrier, error) {
	return nil, fmt.Errorf("error")
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := httpRequest(testutil.Context(), tt.urlStr, "0123456789abcdef", tt.req)

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

//nolint:gocognit,tparallel,paralleltest
func Test_sendRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		req               *DeployRegistrationRequest
		createMockHandler func(t *testing.T) http.HandlerFunc
		setupMocks        func(client *MockHTTPClient)
		setupPatches      func() (*mpatch.Patch, error)
		wantErr           bool
	}{
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
			name: "failed to execute request - transport error",
			setupMocks: func(m *MockHTTPClient) {
				m.EXPECT().Do(gomock.Any()).Return(nil, fmt.Errorf("transport error")).Times(1)
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
		tt := tt

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

			err = sendRequest(testutil.Context(), c, ts.URL+urlTestPath, tt.req)
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
		tt := tt

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

			err = c.SendDeployRegistration(testutil.Context(), tt.req)
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
		tt := tt

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

			err = c.SendManualChange(testutil.Context(), tt.req)
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
		tt := tt

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

			err = c.SendCustomIncidentImpactRegistration(testutil.Context(), tt.req)
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
		tt := tt

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

			err = c.SendCustomMetricImpactRegistration(testutil.Context(), tt.req)
			require.Equal(t, tt.wantErr, err != nil, "error: %v", err)
		})
	}
}
