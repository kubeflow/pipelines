// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func TestCreateVisualization_MissingVisualizationReturnsInvalidInput(t *testing.T) {
	t.Setenv("ENABLE_PYTHON_VISUALIZATIONS", "true")
	wire, err := proto.Marshal(&apiv1beta1.CreateVisualizationRequest{Namespace: "ns1"})
	require.NoError(t, err)
	var callErr error
	if assert.NotPanics(t, func() {
		_, callErr = apiv1beta1.VisualizationService_ServiceDesc.Methods[0].Handler(
			&VisualizationServer{}, context.Background(),
			func(dst interface{}) error { return proto.Unmarshal(wire, dst.(proto.Message)) }, nil,
		)
	}) {
		assert.Equal(t, codes.InvalidArgument, status.Code(callErr))
	}
}

func TestCreateVisualization_NilRequestReturnsInvalidInput(t *testing.T) {
	t.Setenv("ENABLE_PYTHON_VISUALIZATIONS", "true")
	var err error
	if assert.NotPanics(t, func() {
		_, err = (&VisualizationServer{}).CreateVisualizationV1(context.Background(), nil)
	}) {
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}

func TestValidateCreateVisualizationRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager: manager,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_ArgumentsAreEmpty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager: manager,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_SourceIsEmpty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager: manager,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "",
		Arguments: "{}",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires a Source to be provided. Received")
}

func TestValidateCreateVisualizationRequest_SourceIsEmptyAndTypeIsCustom(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager: manager,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_CUSTOM,
		Arguments: "{}",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_ArgumentsNotValidJSON(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager: manager,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires valid JSON to be provided as Arguments. Received {")
}

func TestGenerateVisualization(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		rw.Write([]byte("roc_curve"))
	}))
	defer httpServer.Close()
	server := &VisualizationServer{
		resourceManager: manager,
		serviceURL:      httpServer.URL,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, err)
	assert.Equal(t, []byte("roc_curve"), body)
}

func TestGenerateVisualization_ServiceNotAvailableError(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		if req.Method == http.MethodGet {
			rw.WriteHeader(500)
		} else {
			rw.WriteHeader(200)
		}
	}))
	server := &VisualizationServer{
		resourceManager: manager,
		serviceURL:      httpServer.URL,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}

func TestGenerateVisualization_ServiceHostNotExistError(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	nonExistingServerURL := "http://127.0.0.2:53484"
	server := &VisualizationServer{
		resourceManager: manager,
		serviceURL:      nonExistingServerURL,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, body)
	errMsg := err.Error()
	assert.Contains(t, errMsg, "Unable to verify visualization service aliveness")
	assert.Contains(t, err.Error(), fmt.Sprintf("dial tcp %s", nonExistingServerURL[7:]))
}

func TestGenerateVisualization_ServerError(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		// The get requests 200s to indicate the service is alive, but the
		// visualization request fails with a 500.
		if req.Method == http.MethodGet {
			rw.WriteHeader(200)
		} else {
			rw.WriteHeader(500)
		}
	}))
	defer httpServer.Close()
	server := &VisualizationServer{
		resourceManager: manager,
		serviceURL:      httpServer.URL,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, body)
	assert.Equal(t, "500 Internal Server Error", err.Error())
}

func TestGetVisualizationServiceURL(t *testing.T) {
	server := &VisualizationServer{
		resourceManager: nil,
		serviceURL:      "http://host:port",
	}
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: nil,
	}
	url := server.getVisualizationServiceURL(request)
	assert.Equal(t, "http://host:port", url)
}

func TestGetVisualizationServiceURL_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set("VisualizationService.Name", "ml-pipeline-visualizationserver")
	viper.Set("VisualizationService.Port", "8888")

	server := &VisualizationServer{
		resourceManager: nil,
		serviceURL:      "http://host:port",
	}

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: nil,
		Namespace:     "ns1",
	}
	url := server.getVisualizationServiceURL(request)
	assert.Equal(t, "http://ml-pipeline-visualizationserver.ns1:8888", url)

	// when namespace is not provided, we fall back to the default visuliaztion service
	request = &apiv1beta1.CreateVisualizationRequest{
		Visualization: nil,
	}
	url = server.getVisualizationServiceURL(request)
	assert.Equal(t, "http://host:port", url)
}

func TestCreateVisualization_Unauthorized(t *testing.T) {
	t.Setenv("ENABLE_PYTHON_VISUALIZATIONS", "true")
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	server := &VisualizationServer{
		resourceManager: resourceManager,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
		Namespace:     "ns1",
	}
	_, err := server.CreateVisualizationV1(ctx, request)
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbCreate,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeVisualizations,
	}
	assert.EqualError(
		t,
		err,
		util.Wrap(getPermissionDeniedError(userIdentity, resourceAttributes), "Failed to authorize on namespace").Error(),
	)
}

func TestCreateVisualization_Unauthenticated(t *testing.T) {
	t.Setenv("ENABLE_PYTHON_VISUALIZATIONS", "true")
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	server := &VisualizationServer{
		resourceManager: resourceManager,
	}
	visualization := &apiv1beta1.Visualization{
		Type:      apiv1beta1.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: visualization,
		Namespace:     "ns1",
	}
	_, err := server.CreateVisualizationV1(ctx, request)
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"there is no user identity header",
	)
}

// visualizationRoundTripper records requests without resolving Kubernetes service names.
type visualizationRoundTripper func(*http.Request) (*http.Response, error)

func (f visualizationRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestCreateVisualization_NamespaceAuthorization(t *testing.T) {
	t.Setenv("ENABLE_PYTHON_VISUALIZATIONS", "true")
	tests := []struct {
		name          string
		multiUser     bool
		namespace     string
		authenticated bool
		allowed       bool
		wantCode      codes.Code
		wantReview    bool
		wantHost      string
	}{
		{
			name: "multi-user empty namespace", multiUser: true, authenticated: true,
			allowed: true, wantCode: codes.InvalidArgument,
		},
		{
			name: "multi-user empty namespace without identity", multiUser: true,
			allowed: true, wantCode: codes.InvalidArgument,
		},
		{
			name: "multi-user unauthorized namespace", multiUser: true, namespace: "ns1",
			authenticated: true, wantCode: codes.PermissionDenied, wantReview: true,
		},
		{
			name: "multi-user namespace without identity", multiUser: true, namespace: "ns1",
			allowed: true, wantCode: codes.Unauthenticated,
		},
		{
			name: "multi-user authorized namespace", multiUser: true, namespace: "ns1",
			authenticated: true, allowed: true, wantReview: true,
			wantHost: "ml-pipeline-visualizationserver.ns1:8888",
		},
		{
			name: "single-user empty namespace", wantHost: "system-visualization:8888",
		},
	}
	for _, test := range tests {
		for _, visualizationType := range []apiv1beta1.Visualization_Type{
			apiv1beta1.Visualization_ROC_CURVE, apiv1beta1.Visualization_CUSTOM,
		} {
			t.Run(test.name+"/"+visualizationType.String(), func(t *testing.T) {
				oldMultiUser := viper.Get(common.MultiUserMode)
				oldServiceName := viper.Get(visualizationServiceName)
				oldServicePort := viper.Get(visualizationServicePort)
				t.Cleanup(func() {
					viper.Set(common.MultiUserMode, oldMultiUser)
					viper.Set(visualizationServiceName, oldServiceName)
					viper.Set(visualizationServicePort, oldServicePort)
				})
				viper.Set(common.MultiUserMode, test.multiUser)
				viper.Set(visualizationServiceName, "ml-pipeline-visualizationserver")
				viper.Set(visualizationServicePort, "8888")

				var methods []string
				oldHTTPClient := http.DefaultClient
				t.Cleanup(func() { http.DefaultClient = oldHTTPClient })
				http.DefaultClient = &http.Client{Transport: visualizationRoundTripper(func(request *http.Request) (*http.Response, error) {
					methods = append(methods, request.Method)
					assert.Equal(t, test.wantHost, request.URL.Host)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("visualization HTML")),
						Header:     make(http.Header),
					}, nil
				})}

				clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
				t.Cleanup(func() { clientManager.Close() })
				reviews := &recordingSubjectAccessReviewClient{allowed: test.allowed}
				clientManager.SubjectAccessReviewClientFake = reviews
				server := &VisualizationServer{
					resourceManager: resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false}),
					serviceURL:      "http://system-visualization:8888",
				}
				ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
				if test.authenticated {
					ctx = userContext()
				}
				request := &apiv1beta1.CreateVisualizationRequest{
					Namespace: test.namespace,
					Visualization: &apiv1beta1.Visualization{
						Type: visualizationType, Source: "gs://ml-pipeline/roc/data.csv", Arguments: "{}",
					},
				}

				result, err := server.CreateVisualizationV1(ctx, request)
				if test.wantCode != codes.OK {
					require.Error(t, err)
					assert.True(t, util.IsUserErrorCodeMatch(err, test.wantCode), "unexpected error: %v", err)
					assert.Nil(t, result)
					assert.Empty(t, methods, "rejected requests must not contact any visualization service")
				} else {
					require.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, "visualization HTML", result.Html)
					assert.Equal(t, []string{http.MethodGet, http.MethodPost}, methods)
				}
				if test.wantReview {
					assert.Equal(t, []authorizationv1.ResourceAttributes{{
						Namespace: test.namespace,
						Verb:      common.RbacResourceVerbCreate,
						Group:     common.RbacPipelinesGroup,
						Version:   common.RbacPipelinesVersion,
						Resource:  common.RbacResourceTypeVisualizations,
					}}, reviews.requests)
				} else {
					assert.Empty(t, reviews.requests)
				}
			})
		}
	}
}

func TestCreateVisualization_Disabled(t *testing.T) {
	for _, value := range []string{"", "false", "TRUE", "1", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ENABLE_PYTHON_VISUALIZATIONS", value)
			server := &VisualizationServer{}
			result, err := server.CreateVisualizationV1(context.Background(), nil)
			assert.Nil(t, result)
			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, status.Code(err))
			assert.Contains(t, err.Error(), "Python visualizations are disabled")
		})
	}
}
