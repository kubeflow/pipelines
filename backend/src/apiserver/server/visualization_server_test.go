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
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// startFakeVisualizationService spins up an httptest server that returns 200
// for GET (liveness probe) and writes responseBody for POST (generate).
// It sets viper so that buildVisualizationServiceURL resolves to it.
// Callers must defer the returned closer.
func startFakeVisualizationService(t *testing.T, responseBody string, postStatusCode int) (close func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		if req.Method == http.MethodGet {
			rw.WriteHeader(http.StatusOK)
			return
		}
		rw.WriteHeader(postStatusCode)
		if responseBody != "" {
			rw.Write([]byte(responseBody))
		}
	}))

	addr := srv.Listener.Addr().String()
	host, port, _ := splitHostPort(addr)
	viper.Set(visualizationServiceName, host)
	viper.Set(visualizationServicePort, port)

	return func() {
		srv.Close()
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}
}

func splitHostPort(hostport string) (host, port string, err error) {
	return net.SplitHostPort(hostport)
}

func newV1Server(t *testing.T) (*VisualizationServerV1, func() error) {
	t.Helper()
	clients, manager, _ := initWithExperiment(t)
	return NewVisualizationServerV1(manager), clients.Close
}

func newV2Server(t *testing.T) (*VisualizationServer, func() error) {
	t.Helper()
	clients, manager, _ := initWithExperiment(t)
	return NewVisualizationServer(manager), clients.Close
}

func TestBuildVisualizationServiceURL_SingleUser(t *testing.T) {
	viper.Set(visualizationServiceName, "ml-pipeline-visualizationserver")
	viper.Set(visualizationServicePort, "8888")
	defer func() {
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}()

	url := buildVisualizationServiceURL("")
	assert.Equal(t, "http://ml-pipeline-visualizationserver:8888", url)
}

func TestBuildVisualizationServiceURL_MultiuserWithNamespace(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(visualizationServiceName, "ml-pipeline-visualizationserver")
	viper.Set(visualizationServicePort, "8888")
	defer func() {
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}()

	url := buildVisualizationServiceURL("ns1")
	assert.Equal(t, "http://ml-pipeline-visualizationserver.ns1:8888", url)
}

func TestBuildVisualizationServiceURL_MultiuserEmptyNamespaceFallsBackToServiceName(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(visualizationServiceName, "ml-pipeline-visualizationserver")
	viper.Set(visualizationServicePort, "8888")
	defer func() {
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}()

	url := buildVisualizationServiceURL("")
	assert.Equal(t, "http://ml-pipeline-visualizationserver:8888", url)
}

func TestV1_ValidateCreateVisualizationRequest(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	assert.Nil(t, server.validateCreateVisualizationRequest(request))
}

func TestV1_ValidateCreateVisualizationRequest_ArgumentsAreEmpty(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "",
		},
	}
	assert.Nil(t, server.validateCreateVisualizationRequest(request))
	assert.Equal(t, "{}", request.Visualization.Arguments)
}

func TestV1_ValidateCreateVisualizationRequest_SourceIsEmpty(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "",
			Arguments: "{}",
		},
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires a Source to be provided. Received")
}

func TestV1_ValidateCreateVisualizationRequest_SourceIsEmptyAndTypeIsCustom(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_CUSTOM,
			Arguments: "{}",
		},
	}
	assert.Nil(t, server.validateCreateVisualizationRequest(request))
}

func TestV1_ValidateCreateVisualizationRequest_ArgumentsNotValidJSON(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{",
		},
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires valid JSON to be provided as Arguments. Received {")
}

func TestV1_GenerateVisualization(t *testing.T) {
	server, close := newV1Server(t)
	defer close()
	stopService := startFakeVisualizationService(t, "roc_curve", http.StatusOK)
	defer stopService()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	body, err := server.generateVisualization(request)
	assert.Nil(t, err)
	assert.Equal(t, []byte("roc_curve"), body)
}

func TestV1_GenerateVisualization_ServiceNotAvailableError(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	host, port, _ := splitHostPort(srv.Listener.Addr().String())
	viper.Set(visualizationServiceName, host)
	viper.Set(visualizationServicePort, port)
	defer func() {
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	body, err := server.generateVisualization(request)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}

func TestV1_GenerateVisualization_ServiceHostNotExistError(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	viper.Set(visualizationServiceName, "127.0.0.2")
	viper.Set(visualizationServicePort, "53484")
	defer func() {
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	body, err := server.generateVisualization(request)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "Unable to verify visualization service aliveness")
	assert.Contains(t, err.Error(), "dial tcp 127.0.0.2:53484")
}

func TestV1_GenerateVisualization_ServerError(t *testing.T) {
	server, close := newV1Server(t)
	defer close()

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			rw.WriteHeader(http.StatusOK)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	host, port, _ := splitHostPort(srv.Listener.Addr().String())
	viper.Set(visualizationServiceName, host)
	viper.Set(visualizationServicePort, port)
	defer func() {
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}()

	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	body, err := server.generateVisualization(request)
	assert.Nil(t, body)
	assert.Equal(t, "visualization service returned non-OK status: 500 Internal Server Error", err.Error())
}

func TestV1_CreateVisualization_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	server := NewVisualizationServerV1(resourceManager)
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
		Namespace: "ns1",
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
	assert.EqualError(t, err, util.Wrap(getPermissionDeniedError(userIdentity, resourceAttributes), "Failed to authorize on namespace").Error())
}

func TestV1_CreateVisualization_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	server := NewVisualizationServerV1(resourceManager)
	request := &apiv1beta1.CreateVisualizationRequest{
		Visualization: &apiv1beta1.Visualization{
			Type:      apiv1beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
		Namespace: "ns1",
	}
	_, err := server.CreateVisualizationV1(ctx, request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "there is no user identity header")
}

func TestV2_ValidateCreateVisualizationRequest(t *testing.T) {
	server, close := newV2Server(t)
	defer close()

	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	assert.Nil(t, server.validateCreateVisualizationRequest(request))
}

func TestV2_ValidateCreateVisualizationRequest_ArgumentsAreEmpty(t *testing.T) {
	server, close := newV2Server(t)
	defer close()

	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "",
		},
	}
	assert.Nil(t, server.validateCreateVisualizationRequest(request))
	assert.Equal(t, "{}", request.Visualization.Arguments)
}

func TestV2_ValidateCreateVisualizationRequest_SourceIsEmpty(t *testing.T) {
	server, close := newV2Server(t)
	defer close()

	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "",
			Arguments: "{}",
		},
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires a Source to be provided. Received")
}

func TestV2_ValidateCreateVisualizationRequest_SourceIsEmptyAndTypeIsCustom(t *testing.T) {
	server, close := newV2Server(t)
	defer close()

	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_CUSTOM,
			Arguments: "{}",
		},
	}
	assert.Nil(t, server.validateCreateVisualizationRequest(request))
}

func TestV2_ValidateCreateVisualizationRequest_ArgumentsNotValidJSON(t *testing.T) {
	server, close := newV2Server(t)
	defer close()

	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{",
		},
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires valid JSON to be provided as Arguments. Received {")
}

func TestV2_GenerateVisualization(t *testing.T) {
	server, close := newV2Server(t)
	defer close()
	stopService := startFakeVisualizationService(t, "roc_curve", http.StatusOK)
	defer stopService()

	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	body, err := server.generateVisualization(request)
	assert.Nil(t, err)
	assert.Equal(t, []byte("roc_curve"), body)
}

func TestV2_GenerateVisualization_ServerError(t *testing.T) {
	server, close := newV2Server(t)
	defer close()

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			rw.WriteHeader(http.StatusOK)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	host, port, _ := splitHostPort(srv.Listener.Addr().String())
	viper.Set(visualizationServiceName, host)
	viper.Set(visualizationServicePort, port)
	defer func() {
		viper.Set(visualizationServiceName, "")
		viper.Set(visualizationServicePort, "")
	}()

	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
	}
	body, err := server.generateVisualization(request)
	assert.Nil(t, body)
	assert.Equal(t, "visualization service returned non-OK status: 500 Internal Server Error", err.Error())
}

func TestV2_CreateVisualization_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	server := NewVisualizationServer(resourceManager)
	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
		Namespace: "ns1",
	}
	_, err := server.CreateVisualization(ctx, request)
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbCreate,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeVisualizations,
	}
	assert.EqualError(t, err, util.Wrap(getPermissionDeniedError(userIdentity, resourceAttributes), "Failed to authorize on namespace").Error())
}

func TestV2_CreateVisualization_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	server := NewVisualizationServer(resourceManager)
	request := &apiv2beta1.CreateVisualizationRequest{
		Visualization: &apiv2beta1.Visualization{
			Type:      apiv2beta1.Visualization_ROC_CURVE,
			Source:    "gs://ml-pipeline/roc/data.csv",
			Arguments: "{}",
		},
		Namespace: "ns1",
	}
	_, err := server.CreateVisualization(ctx, request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "there is no user identity header")
}

