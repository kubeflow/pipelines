// Copyright 2018-2023 The Kubeflow Authors
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
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

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
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
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
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
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
