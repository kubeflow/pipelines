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
	"strings"
	"testing"

	"google.golang.org/protobuf/testing/protocmp"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	apiV1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestCreateExperimentV1(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	result, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &apiV1beta1.Experiment{
		Id:           DefaultFakeUUID,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	result, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId: DefaultFakeUUID,
		DisplayName:  "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiV2beta1.Experiment_AVAILABLE,
		Namespace:    "default",
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperimentV1_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to add experiment to experiment table")
}

func TestCreateExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to add experiment to experiment table")
}

// func TestCreateExperimentV1_SingleUser_NamespaceNotAllowed(t *testing.T) {
// 	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
// 	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
// 	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
// 	resourceReferences := []*apiV1beta1.ResourceReference{
// 		{
// 			Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
// 			Relationship: apiV1beta1.Relationship_OWNER,
// 		},
// 	}
// 	experiment := &apiV1beta1.Experiment{
// 		Name:               "exp1",
// 		Description:        "first experiment",
// 		ResourceReferences: resourceReferences,
// 	}

// 	_, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
// 	assert.NotNil(t, err)
// 	assert.Contains(t, err.Error(), "In single-user mode, CreateExperimentRequest shouldn't contain resource references")
// }
//
// func TestCreateExperiment_SingleUser_NamespaceNotAllowed(t *testing.T) {
// 	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
// 	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
// 	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
// 	experiment := &apiV2beta1.Experiment{DisplayName: "exp1", Description: "first experiment", Namespace: "ns1"}
// 	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
// 	assert.NotNil(t, err)
// 	assert.Contains(t, err.Error(), "In single-user mode, CreateExperimentRequest shouldn't contain namespace")
// }

func TestCreateExperimentV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, resourceManager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{
		Name:        "exp1",
		Description: "first experiment",
		ResourceReferences: []*apiV1beta1.ResourceReference{
			{
				Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
				Relationship: apiV1beta1.Relationship_OWNER,
			},
		}}

	_, err := server.CreateExperimentV1(ctx, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed (request: &ResourceAttributes{Namespace:ns1,Verb:create,Group:pipelines.kubeflow.org,Version:v1beta1,Resource:experiments,Subresource:,Name:exp1,})",
	)
}

func TestCreateExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, resourceManager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{
		DisplayName: "exp1",
		Description: "first experiment",
		Namespace:   "ns1",
	}

	_, err := server.CreateExperiment(ctx, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"ermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed (request: &ResourceAttributes{Namespace:ns1,Verb:create,Group:pipelines.kubeflow.org,Version:v1beta1,Resource:experiments,Subresource:,Name:exp1,})",
	)
}

func TestCreateExperimentV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*apiV1beta1.ResourceReference{
		{
			Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: apiV1beta1.Relationship_OWNER,
		},
	}
	experiment := &apiV1beta1.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	result, err := server.CreateExperimentV1(ctx, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &apiV1beta1.Experiment{
		Id:                 DefaultFakeUUID,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
		StorageState:       apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{
		DisplayName: "exp1",
		Description: "first experiment",
		Namespace:   "ns1",
	}

	result, err := server.CreateExperiment(ctx, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId: DefaultFakeUUID,
		DisplayName:  "exp1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		Namespace:    "ns1",
		StorageState: apiV2beta1.Experiment_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperimentV1(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperimentV1(nil, &apiV1beta1.GetExperimentRequest{Id: createResult.Id})
	assert.Nil(t, err)
	expectedExperiment := &apiV1beta1.Experiment{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: createResult.ExperimentId})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId: createResult.ExperimentId,
		DisplayName:  "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiV2beta1.Experiment_AVAILABLE,
		Namespace:    "default",
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperimentV1_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperimentV1(nil, &apiV1beta1.GetExperimentRequest{Id: createResult.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestGetExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: createResult.ExperimentId})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestGetExperimentV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.GetExperimentV1(ctx, &apiV1beta1.GetExperimentRequest{Id: experiment.UUID})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed (request: &ResourceAttributes{Namespace:ns1,Verb:get,Group:pipelines.kubeflow.org,Version:v1beta1,Resource:experiments,Subresource:,Name:exp1,})",
	)
}

func TestGetExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.GetExperiment(ctx, &apiV2beta1.GetExperimentRequest{ExperimentId: experiment.UUID})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed (request: &ResourceAttributes{Namespace:ns1,Verb:get,Group:pipelines.kubeflow.org,Version:v1beta1,Resource:experiments,Subresource:,Name:exp1,})",
	)
}

func TestGetExperimentV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*apiV1beta1.ResourceReference{
		{
			Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: apiV1beta1.Relationship_OWNER,
		},
	}
	experiment := &apiV1beta1.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	createResult, err := server.CreateExperimentV1(ctx, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperimentV1(ctx, &apiV1beta1.GetExperimentRequest{Id: createResult.Id})
	assert.Nil(t, err)
	expectedExperiment := &apiV1beta1.Experiment{
		Id:                 createResult.Id,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
		StorageState:       apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{
		DisplayName: "exp1",
		Description: "first experiment",
		Namespace:   "ns1",
	}

	createResult, err := server.CreateExperiment(ctx, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(ctx, &apiV2beta1.GetExperimentRequest{ExperimentId: createResult.ExperimentId})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId: createResult.ExperimentId,
		DisplayName:  "exp1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		Namespace:    "ns1",
		StorageState: apiV2beta1.Experiment_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestListExperimentsV1(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperimentsV1(nil, &apiV1beta1.ListExperimentsRequest{
		ResourceReferenceKey: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "default"},
	})
	expectedExperiment := []*apiV1beta1.Experiment{{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
		ResourceReferences: []*apiV1beta1.ResourceReference{
			{
				Key:          &apiV1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}}
	assert.Nil(t, err)
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperiments(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperiments(nil, &apiV2beta1.ListExperimentsRequest{})
	expectedExperiment := []*apiV2beta1.Experiment{{
		ExperimentId: createResult.ExperimentId,
		DisplayName:  "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiV2beta1.Experiment_AVAILABLE,
		Namespace:    "default",
	}}
	assert.Nil(t, err)
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperimentsV1_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperimentsV1(nil, &apiV1beta1.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed")
}

func TestListExperiments_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperiments(nil, &apiV2beta1.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed")
}

// func TestListExperimentsV1_SingleUser_NamespaceNotAllowed(t *testing.T) {
// 	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
// 	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
// 	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
// 	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}

// 	_, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
// 	assert.Nil(t, err)
// 	_, err = server.ListExperimentsV1(nil, &apiV1beta1.ListExperimentsRequest{
// 		ResourceReferenceKey: &apiV1beta1.ResourceKey{
// 			Type: apiV1beta1.ResourceType_NAMESPACE,
// 			Id:   "ns1",
// 		},
// 	})
// 	assert.NotNil(t, err)
// 	assert.Contains(t, err.Error(), "In single-user mode, ListExperimentsV1 cannot filter by namespace")
// }

// func TestListExperiments_SingleUser_NamespaceNotAllowed(t *testing.T) {
// 	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
// 	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
// 	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
// 	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

// 	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
// 	assert.Nil(t, err)
// 	_, err = server.ListExperiments(nil, &apiV2beta1.ListExperimentsRequest{Namespace: "ns1"})
// 	assert.NotNil(t, err)
// 	assert.Contains(t, err.Error(), "Invalid ListExperiments request. Namespace should not be provided in single-user mode")
// }

func TestListExperimentsV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.ListExperimentsV1(ctx, &apiV1beta1.ListExperimentsRequest{
		ResourceReferenceKey: &apiV1beta1.ResourceKey{
			Type: apiV1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed (request: &ResourceAttributes{Namespace:ns1,Verb:list,Group:pipelines.kubeflow.org,Version:v1beta1,Resource:experiments,Subresource:,Name:,})",
	)
}

func TestListExperiments_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}
	_, err := server.ListExperiments(ctx, &apiV2beta1.ListExperimentsRequest{Namespace: "ns1"})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed (request: &ResourceAttributes{Namespace:ns1,Verb:list,Group:pipelines.kubeflow.org,Version:v1beta1,Resource:experiments,Subresource:,Name:,})",
	)
}

func TestListExperimentsV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "ns1", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}

	resourceReferences := []*apiV1beta1.ResourceReference{
		{
			Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: apiV1beta1.Relationship_OWNER,
		},
	}
	experiment := &apiV1beta1.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	createResult, err := server.CreateExperimentV1(ctx, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	tests := []struct {
		name                string
		request             *apiV1beta1.ListExperimentsRequest
		wantError           bool
		errorMessage        string
		expectedExperiments []*apiV1beta1.Experiment
	}{
		{
			"Valid",
			&apiV1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiV1beta1.ResourceKey{
					Type: apiV1beta1.ResourceType_NAMESPACE,
					Id:   "ns1",
				},
			},
			false,
			"",
			[]*apiV1beta1.Experiment{{
				Id:                 createResult.Id,
				Name:               "exp1",
				Description:        "first experiment",
				CreatedAt:          &timestamp.Timestamp{Seconds: 1},
				ResourceReferences: resourceReferences,
				StorageState:       apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
			}},
		},
		{
			"Valid but empty result",
			&apiV1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiV1beta1.ResourceKey{
					Type: apiV1beta1.ResourceType_NAMESPACE,
					Id:   "ns2",
				},
			},
			false,
			"",
			[]*apiV1beta1.Experiment{},
		},
		{
			"Invalid resource reference key type",
			&apiV1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiV1beta1.ResourceKey{
					Type: apiV1beta1.ResourceType_EXPERIMENT,
					Id:   "fake_id",
				},
			},
			true,
			"invalid resource reference key",
			nil,
		},
		{
			"Missing resource reference key",
			&apiV1beta1.ListExperimentsRequest{},
			false,
			"",
			[]*apiV1beta1.Experiment{{
				Id:                 createResult.Id,
				Name:               "exp1",
				Description:        "first experiment",
				CreatedAt:          &timestamp.Timestamp{Seconds: 1},
				ResourceReferences: resourceReferences,
				StorageState:       apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
			}},
		},
		{
			"Empty namespace",
			&apiV1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiV1beta1.ResourceKey{
					Type: apiV1beta1.ResourceType_NAMESPACE,
					Id:   "",
				},
			},
			false,
			"",
			[]*apiV1beta1.Experiment{{
				Id:                 createResult.Id,
				Name:               "exp1",
				Description:        "first experiment",
				CreatedAt:          &timestamp.Timestamp{Seconds: 1},
				ResourceReferences: resourceReferences,
				StorageState:       apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
			}},
		},
		{
			"No namespace",
			&apiV1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiV1beta1.ResourceKey{
					Type: apiV1beta1.ResourceType_NAMESPACE,
					Id:   "-",
				},
			},
			false,
			"",
			[]*apiV1beta1.Experiment{{
				Id:                 createResult.Id,
				Name:               "exp1",
				Description:        "first experiment",
				CreatedAt:          &timestamp.Timestamp{Seconds: 1},
				ResourceReferences: resourceReferences,
				StorageState:       apiV1beta1.Experiment_STORAGESTATE_AVAILABLE,
			}},
		},
	}

	for _, tc := range tests {
		response, err := server.ListExperimentsV1(ctx, tc.request)
		if tc.wantError {
			if err == nil {
				t.Errorf("TestListExperimentsV1_Multiuser(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestListExperimentsV1_Multiusert(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TestListExperimentsV1_Multiuser(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedExperiments, response.Experiments, cmpopts.EquateEmpty(), protocmp.Transform(), cmpopts.IgnoreFields(apiV1beta1.Experiment{}, "CreatedAt")) {
				t.Errorf("TestListExperimentsV1_Multiuser(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedExperiments, response.Experiments)
			}
		}
	}
}

func TestListExperiments_Multiuser_NoDefault(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{
		DisplayName: "exp1",
		Description: "first experiment",
		Namespace:   "ns1",
	}

	createResult, err := server.CreateExperiment(ctx, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	tests := []struct {
		name                string
		request             *apiV2beta1.ListExperimentsRequest
		wantError           bool
		errorMessage        string
		expectedExperiments []*apiV2beta1.Experiment
	}{
		{
			"Valid",
			&apiV2beta1.ListExperimentsRequest{Namespace: "ns1"},
			false,
			"",
			[]*apiV2beta1.Experiment{{
				ExperimentId: createResult.ExperimentId,
				DisplayName:  "exp1",
				Description:  "first experiment",
				CreatedAt:    &timestamp.Timestamp{Seconds: 1},
				Namespace:    "ns1",
				StorageState: apiV2beta1.Experiment_AVAILABLE,
			}},
		},
		{
			"Valid but empty result",
			&apiV2beta1.ListExperimentsRequest{Namespace: "ns2"},
			false,
			"",
			[]*apiV2beta1.Experiment{},
		},
		{
			"Missing namespace",
			&apiV2beta1.ListExperimentsRequest{},
			false,
			"",
			[]*apiV2beta1.Experiment{},
		},
	}

	for _, tc := range tests {
		response, err := server.ListExperiments(ctx, tc.request)
		if tc.wantError {
			if err == nil {
				t.Errorf("TestListExperiments_Multiuser(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestListExperiments_Multiusert(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TestListExperiments_Multiuser(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedExperiments, response.Experiments, cmpopts.EquateEmpty(), protocmp.Transform(), cmpopts.IgnoreFields(apiV2beta1.Experiment{}, "CreatedAt")) {
				t.Errorf("TestListExperiments_Multiuser(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedExperiments, response.Experiments)
			}
		}
	}
}

// func TestValidateCreateExperimentRequestV1(t *testing.T) {
// 	tests := []struct {
// 		name         string
// 		experiment   *apiV1beta1.Experiment
// 		wantError    bool
// 		errorMessage string
// 	}{
// 		{
// 			"Valid",
// 			&apiV1beta1.Experiment{Name: "exp1", Description: "first experiment"},
// 			false,
// 			"",
// 		},
// 		{
// 			"Empty name",
// 			&apiV1beta1.Experiment{Description: "first experiment"},
// 			true,
// 			"name is empty",
// 		},
// 	}

// 	for _, tc := range tests {
// 		err := validateCreateExperimentRequestV1(&apiV1beta1.CreateExperimentRequest{Experiment: tc.experiment})
// 		if !tc.wantError && err != nil {
// 			t.Errorf("TestValidateCreateExperimentRequestV1(%v) expect no error but got %v", tc.name, err)
// 		}
// 		if tc.wantError {
// 			if err == nil {
// 				t.Errorf("TestValidateCreateExperimentRequestV1(%v) expect error but got nil", tc.name)
// 			} else if !strings.Contains(err.Error(), tc.errorMessage) {
// 				t.Errorf("TestValidateCreateExperimentRequestV1(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
// 			}
// 		}
// 	}
// // }

// func TestValidateCreateExperimentRequest(t *testing.T) {
// 	tests := []struct {
// 		name         string
// 		experiment   *apiV2beta1.Experiment
// 		wantError    bool
// 		errorMessage string
// 	}{
// 		{
// 			"Valid",
// 			&apiV2beta1.Experiment{DisplayName: "exp1", Description: "first experiment"},
// 			false,
// 			"",
// 		},
// 		{
// 			"Empty name",
// 			&apiV2beta1.Experiment{Description: "first experiment"},
// 			true,
// 			"name is empty",
// 		},
// 	}

// 	for _, tc := range tests {
// 		err := ValidateCreateExperimentRequest(&apiV2beta1.CreateExperimentRequest{Experiment: tc.experiment})
// 		if !tc.wantError && err != nil {
// 			t.Errorf("TestValidateCreateExperimentRequest(%v) expect no error but got %v", tc.name, err)
// 		}
// 		if tc.wantError {
// 			if err == nil {
// 				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error but got nil", tc.name)
// 			} else if !strings.Contains(err.Error(), tc.errorMessage) {
// 				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
// 			}
// 		}
// 	}
// }

// func TestValidateCreateExperimentRequestV1_Multiuser(t *testing.T) {
// 	viper.Set(common.MultiUserMode, "true")
// 	defer viper.Set(common.MultiUserMode, "false")
// 	tests := []struct {
// 		name         string
// 		experiment   *apiV1beta1.Experiment
// 		wantError    bool
// 		errorMessage string
// 	}{
// 		{
// 			"Valid",
// 			&apiV1beta1.Experiment{
// 				Name:        "exp1",
// 				Description: "first experiment",
// 				ResourceReferences: []*apiV1beta1.ResourceReference{
// 					{
// 						Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
// 						Relationship: apiV1beta1.Relationship_OWNER,
// 					},
// 				},
// 			},
// 			false,
// 			"",
// 		},
// 		{
// 			"Missing namespace",
// 			&apiV1beta1.Experiment{
// 				Name:        "exp1",
// 				Description: "first experiment",
// 			},
// 			true,
// 			"Invalid resource references for experiment",
// 		},
// 		{
// 			"Empty namespace",
// 			&apiV1beta1.Experiment{
// 				Name:        "exp1",
// 				Description: "first experiment",
// 				ResourceReferences: []*apiV1beta1.ResourceReference{
// 					{
// 						Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: ""},
// 						Relationship: apiV1beta1.Relationship_OWNER,
// 					},
// 				},
// 			},
// 			true,
// 			"Invalid resource references for experiment. Namespace is empty",
// 		},
// 		{
// 			"Multiple namespace",
// 			&apiV1beta1.Experiment{
// 				Name:        "exp1",
// 				Description: "first experiment",
// 				ResourceReferences: []*apiV1beta1.ResourceReference{
// 					{
// 						Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
// 						Relationship: apiV1beta1.Relationship_OWNER,
// 					},
// 					{
// 						Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns2"},
// 						Relationship: apiV1beta1.Relationship_OWNER,
// 					},
// 				},
// 			},
// 			true,
// 			"Invalid resource references for experiment",
// 		},
// 		{
// 			"Invalid resource type",
// 			&apiV1beta1.Experiment{
// 				Name:        "exp1",
// 				Description: "first experiment",
// 				ResourceReferences: []*apiV1beta1.ResourceReference{
// 					{
// 						Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: "exp2"},
// 						Relationship: apiV1beta1.Relationship_OWNER,
// 					},
// 				},
// 			},
// 			true,
// 			"Invalid resource references for experiment",
// 		},
// 	}

// 	for _, tc := range tests {
// 		err := ValidateCreateExperimentRequestV1(&apiV1beta1.CreateExperimentRequest{Experiment: tc.experiment})
// 		if !tc.wantError && err != nil {
// 			t.Errorf("TestValidateCreateExperimentRequestV1(%v) expect no error but got %v", tc.name, err)
// 		}
// 		if tc.wantError {
// 			if err == nil {
// 				t.Errorf("TestValidateCreateExperimentRequestV1(%v) expect error but got nil", tc.name)
// 			} else if !strings.Contains(err.Error(), tc.errorMessage) {
// 				t.Errorf("TestValidateCreateExperimentRequestV1(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
// 			}
// 		}
// 	}
// }

// func TestValidateCreateExperimentRequest_Multiuser(t *testing.T) {
// 	viper.Set(common.MultiUserMode, "true")
// 	defer viper.Set(common.MultiUserMode, "false")
// 	tests := []struct {
// 		name         string
// 		experiment   *apiV2beta1.Experiment
// 		wantError    bool
// 		errorMessage string
// 	}{
// 		{
// 			"Valid",
// 			&apiV2beta1.Experiment{
// 				DisplayName: "exp1",
// 				Description: "first experiment",
// 				Namespace:   "ns1",
// 			},
// 			false,
// 			"",
// 		},
// 		{
// 			"Missing namespace",
// 			&apiV2beta1.Experiment{
// 				DisplayName: "exp1",
// 				Description: "first experiment",
// 			},
// 			true,
// 			"In multi-user mode, experiment namespace is empty. Please specify a valid namespace",
// 		},
// 	}

// 	for _, tc := range tests {
// 		err := ValidateCreateExperimentRequest(&apiV2beta1.CreateExperimentRequest{Experiment: tc.experiment})
// 		if !tc.wantError && err != nil {
// 			t.Errorf("TestValidateCreateExperimentRequest(%v) expect no error but got %v", tc.name, err)
// 		}
// 		if tc.wantError {
// 			if err == nil {
// 				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error but got nil", tc.name)
// 			} else if !strings.Contains(err.Error(), tc.errorMessage) {
// 				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
// 			}
// 		}
// 	}
// }

func TestArchiveAndUnarchiveExperimentV1(t *testing.T) {
	// Create experiment and runs/jobs under it.
	clients, manager, experiment := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	runServer := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run1 := &apiV1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err := runServer.CreateRunV1(nil, &apiV1beta1.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	manager = resource.NewResourceManager(clients, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	runServer = NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run2 := &apiV1beta1.Run{
		Name:               "run2",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err = runServer.CreateRunV1(nil, &apiV1beta1.CreateRunRequest{Run: run2})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeUUID, nil))
	manager = resource.NewResourceManager(clients, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	jobServer := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job1 := &apiV1beta1.Job{
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiV1beta1.Trigger{
			Trigger: &apiV1beta1.Trigger_CronSchedule{CronSchedule: &apiV1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err = jobServer.CreateJob(nil, &apiV1beta1.CreateJobRequest{Job: job1})
	assert.Nil(t, err)
	jobs, err := jobServer.ListJobs(nil, &apiV1beta1.ListJobsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, true, jobs.Jobs[0].Enabled)

	// Archive the experiment and thus all runs under it.
	experimentServer := NewExperimentServer(manager, &ExperimentServerOptions{CollectMetrics: false})
	_, err = experimentServer.ArchiveExperimentV1(nil, &apiV1beta1.ArchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err := experimentServer.GetExperimentV1(nil, &apiV1beta1.GetExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiV1beta1.Experiment_STORAGESTATE_ARCHIVED, result.StorageState)
	runs, err := runServer.ListRunsV1(nil, &apiV1beta1.ListRunsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err = jobServer.ListJobs(nil, &apiV1beta1.ListJobsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)

	// Unarchive the experiment and thus all runs under it.
	_, err = experimentServer.UnarchiveExperimentV1(nil, &apiV1beta1.UnarchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err = experimentServer.GetExperimentV1(nil, &apiV1beta1.GetExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiV1beta1.Experiment_STORAGESTATE_AVAILABLE, result.StorageState)
	runs, err = runServer.ListRunsV1(nil, &apiV1beta1.ListRunsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err = jobServer.ListJobs(nil, &apiV1beta1.ListJobsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)
}

func TestArchiveAndUnarchiveExperiment(t *testing.T) {
	// Create experiment and runs/jobs under it.
	clients, manager, experiment := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	runServer := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run1 := &apiV1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	// err := runServer.validateCreateRunRequestV1(&apiV1beta1.CreateRunRequest{Run: run1})
	// assert.Nil(t, err)
	_, err := runServer.CreateRunV1(nil, &apiV1beta1.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	manager = resource.NewResourceManager(clients, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	runServer = NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run2 := &apiV1beta1.Run{
		Name:               "run2",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	// err = runServer.validateCreateRunRequestV1(&apiV1beta1.CreateRunRequest{Run: run2})
	// assert.Nil(t, err)
	_, err = runServer.CreateRunV1(nil, &apiV1beta1.CreateRunRequest{Run: run2})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeUUID, nil))
	manager = resource.NewResourceManager(clients, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	jobServer := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job1 := &apiV1beta1.Job{
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiV1beta1.Trigger{
			Trigger: &apiV1beta1.Trigger_CronSchedule{CronSchedule: &apiV1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	// err = jobServer.validateCreateJobRequest(&apiV1beta1.CreateJobRequest{Job: job1})
	// assert.Nil(t, err)
	_, err = jobServer.CreateJob(nil, &apiV1beta1.CreateJobRequest{Job: job1})
	assert.Nil(t, err)

	// Archive the experiment and thus all runs under it.
	experimentServer := NewExperimentServer(manager, &ExperimentServerOptions{CollectMetrics: false})
	_, err = experimentServer.ArchiveExperiment(nil, &apiV2beta1.ArchiveExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	result, err := experimentServer.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiV2beta1.Experiment_ARCHIVED, result.StorageState)
	runs, err := runServer.ListRunsV1(nil, &apiV1beta1.ListRunsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err := jobServer.ListJobs(nil, &apiV1beta1.ListJobsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)

	// Unarchive the experiment and thus all runs under it.
	_, err = experimentServer.UnarchiveExperiment(nil, &apiV2beta1.UnarchiveExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	result, err = experimentServer.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiV2beta1.Experiment_AVAILABLE, result.StorageState)
	runs, err = runServer.ListRunsV1(nil, &apiV1beta1.ListRunsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiV1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err = jobServer.ListJobs(nil, &apiV1beta1.ListJobsRequest{ResourceReferenceKey: &apiV1beta1.ResourceKey{Id: experiment.UUID, Type: apiV1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)
}

// TestDeleteExperiments_SingleUser tests (1) deleting an existing experiment, and
// deleting an experiment that does not exist in single user mode, for V2 api.
func TestDeleteExperiments_SingleUser(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}
	resultExperiment, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperiment(nil, &apiV2beta1.DeleteExperimentRequest{ExperimentId: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperiment(nil, &apiV2beta1.DeleteExperimentRequest{ExperimentId: resultExperiment.ExperimentId})
	assert.Nil(t, err)
}

// TestDeleteExperimentsV1_SingleUser tests (1) deleting an existing experiment, and
// deleting an experiment that does not exist in single user mode, for V1 api.
func TestDeleteExperimentsV1_SingleUser(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV1beta1.Experiment{Name: "ex1", Description: "first experiment"}
	resultExperiment, err := server.CreateExperimentV1(nil, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperimentV1(nil, &apiV1beta1.DeleteExperimentRequest{Id: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperimentV1(nil, &apiV1beta1.DeleteExperimentRequest{Id: resultExperiment.Id})
	assert.Nil(t, err)
}

// TestDeleteExperiments_MultiUser tests (1) deleting an existing experiment, and
// deleting an experiment that does not exist in ,multi user mode, for V2 api.
func TestDeleteExperiments_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment", Namespace: "ns1"}
	resultExperiment, err := server.CreateExperiment(ctx, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperiment(ctx, &apiV2beta1.DeleteExperimentRequest{ExperimentId: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperiment(ctx, &apiV2beta1.DeleteExperimentRequest{ExperimentId: resultExperiment.ExperimentId})
	assert.Nil(t, err)
}

// TestDeleteExperimentsV1_MultiUser tests (1) deleting an existing experiment, and
// deleting an experiment that does not exist in multi user mode, for V1 api.
func TestDeleteExperimentsV1_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*apiV1beta1.ResourceReference{
		{
			Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: apiV1beta1.Relationship_OWNER,
		},
	}
	experiment := &apiV1beta1.Experiment{
		Name:               "ex1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences}
	resultExperiment, err := server.CreateExperimentV1(ctx, &apiV1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperimentV1(ctx, &apiV1beta1.DeleteExperimentRequest{Id: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperimentV1(ctx, &apiV1beta1.DeleteExperimentRequest{Id: resultExperiment.Id})
	assert.Nil(t, err)
}

func TestListExperimentsV1_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}
	_, err := server.ListExperimentsV1(ctx, &apiV1beta1.ListExperimentsRequest{
		ResourceReferenceKey: &apiV1beta1.ResourceKey{
			Type: apiV1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"User identity is empty in the request header",
	)
}
