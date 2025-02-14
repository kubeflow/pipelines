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
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestCreateExperimentV1(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	result, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &apiv1beta1.Experiment{
		Id:           DefaultFakeUUID,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: ""},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	result, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId:     DefaultFakeUUID,
		DisplayName:      "ex1",
		Description:      "first experiment",
		CreatedAt:        &timestamp.Timestamp{Seconds: 1},
		LastRunCreatedAt: &timestamp.Timestamp{Seconds: 0},
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperimentV1_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to add experiment to experiment table")
}

func TestCreateExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to add experiment to experiment table")
}

func TestCreateExperiment_EmptyName(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid input error: Experiment must have a non-empty name")
}

func TestCreateExperimentV1_EmptyName(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid input error: Experiment must have a non-empty name")
}

func TestCreateExperimentV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, resourceManager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{
		Name:        "exp1",
		Description: "first experiment",
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}

	_, err := server.CreateExperimentV1(ctx, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
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
		"ermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
	)
}

func TestCreateExperimentV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}

	tests := []struct {
		name       string
		fakeId     string
		experiment *apiv1beta1.Experiment
		want       *apiv1beta1.Experiment
		wantError  bool
		errMsg     string
	}{
		{
			"Valid - first experiment",
			DefaultFakeIdOne,
			&apiv1beta1.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			&apiv1beta1.Experiment{
				Id:          DefaultFakeIdOne,
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
				StorageState: apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
			},
			false,
			"",
		},
		{
			"Valid - second experiment",
			DefaultFakeIdTwo,
			&apiv1beta1.Experiment{
				Name:        "exp2",
				Description: "second experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			&apiv1beta1.Experiment{
				Id:          DefaultFakeIdTwo,
				Name:        "exp2",
				Description: "second experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
				StorageState: apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
			},
			false,
			"",
		},
		{
			"Invalid - same name",
			DefaultFakeIdThree,
			&apiv1beta1.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			nil,
			true,
			"The name exp1 already exists. Please specify a new name",
		},
		{
			"Invalid - missing namespace",
			DefaultFakeIdFour,
			&apiv1beta1.Experiment{
				Name:        "exp3",
				Description: "first experiment",
			},
			nil,
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
		},
		{
			"Invalid - missing name",
			DefaultFakeIdFive,
			&apiv1beta1.Experiment{
				Description: "first experiment",
			},
			nil,
			true,
			"Invalid input error: Experiment must have a non-empty name",
		},
		{
			"Invalid - empty namespace",
			DefaultFakeIdSix,
			&apiv1beta1.Experiment{
				Name:        "exp4",
				Description: "first experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: ""},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			nil,
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
		},
		{
			"Valid - multiple namespaces",
			DefaultFakeIdSeven,
			&apiv1beta1.Experiment{
				Name:        "exp5",
				Description: "first experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns2"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			&apiv1beta1.Experiment{
				Id:          DefaultFakeIdSeven,
				Name:        "exp5",
				Description: "first experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
				StorageState: apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
			},
			false,
			"",
		},
		{
			"Invalid - wrong owner",
			DefaultFakeIdEight,
			&apiv1beta1.Experiment{
				Name:        "exp6",
				Description: "first experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp2"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			nil,
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
		},
	}
	for _, tt := range tests {
		clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(tt.fakeId, nil))
		resourceManager = resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
		server = ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
		got, err := server.CreateExperimentV1(ctx, &apiv1beta1.CreateExperimentRequest{Experiment: tt.experiment})
		if tt.wantError {
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		} else {
			assert.Nil(t, err)
			assert.NotNil(t, got.CreatedAt)
			tt.want.CreatedAt = got.CreatedAt
		}
		assert.Equal(t, tt.want, got)
	}
}

func TestCreateExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}

	tests := []struct {
		name       string
		experiment *apiV2beta1.Experiment
		want       *apiV2beta1.Experiment
		wantError  bool
		errMsg     string
	}{
		{
			"Valid",
			&apiV2beta1.Experiment{
				DisplayName:      "exp1",
				Description:      "first experiment",
				LastRunCreatedAt: &timestamp.Timestamp{Seconds: 0},
				Namespace:        "ns1",
			},
			&apiV2beta1.Experiment{
				ExperimentId:     DefaultFakeUUID,
				DisplayName:      "exp1",
				Description:      "first experiment",
				LastRunCreatedAt: &timestamp.Timestamp{Seconds: 0},
				Namespace:        "ns1",
				StorageState:     apiV2beta1.Experiment_AVAILABLE,
			},
			false,
			"",
		},
		{
			"Invalid - missing namespace",
			&apiV2beta1.Experiment{
				DisplayName: "exp1",
				Description: "first experiment",
			},
			nil,
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
		},
		{
			"Invalid - missing name",
			&apiV2beta1.Experiment{
				Description: "first experiment",
				Namespace:   "ns1",
			},
			nil,
			true,
			"Invalid input error: Experiment must have a non-empty name",
		},
	}
	for _, tt := range tests {
		got, err := server.CreateExperiment(ctx, &apiV2beta1.CreateExperimentRequest{Experiment: tt.experiment})
		if tt.wantError {
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		} else {
			assert.Nil(t, err)
			assert.NotNil(t, got.CreatedAt)
			tt.want.CreatedAt = got.CreatedAt
		}
		assert.Equal(t, tt.want, got)
	}
}

func TestGetExperimentV1(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperimentV1(nil, &apiv1beta1.GetExperimentRequest{Id: createResult.Id})
	assert.Nil(t, err)
	expectedExperiment := &apiv1beta1.Experiment{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: ""},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: createResult.ExperimentId})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId:     createResult.ExperimentId,
		DisplayName:      "ex1",
		Description:      "first experiment",
		CreatedAt:        &timestamp.Timestamp{Seconds: 1},
		LastRunCreatedAt: &timestamp.Timestamp{Seconds: 0},
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperimentV1_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperimentV1(nil, &apiv1beta1.GetExperimentRequest{Id: createResult.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestGetExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
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

	_, err := server.GetExperimentV1(ctx, &apiv1beta1.GetExperimentRequest{Id: experiment.UUID})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
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
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
	)
}

func TestGetExperimentV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*apiv1beta1.ResourceReference{
		{
			Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	experiment := &apiv1beta1.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	createResult, err := server.CreateExperimentV1(ctx, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperimentV1(ctx, &apiv1beta1.GetExperimentRequest{Id: createResult.Id})
	assert.Nil(t, err)
	expectedExperiment := &apiv1beta1.Experiment{
		Id:                 createResult.Id,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
		StorageState:       apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
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
		ExperimentId:     createResult.ExperimentId,
		DisplayName:      "exp1",
		Description:      "first experiment",
		CreatedAt:        &timestamp.Timestamp{Seconds: 1},
		LastRunCreatedAt: &timestamp.Timestamp{Seconds: 0},
		Namespace:        "ns1",
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestListExperimentsV1(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperimentsV1(nil, &apiv1beta1.ListExperimentsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: ""},
	})
	expectedExperiment := []*apiv1beta1.Experiment{{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: ""},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}}
	assert.Nil(t, err)
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperiments(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperiments(nil, &apiV2beta1.ListExperimentsRequest{})
	expectedExperiment := []*apiV2beta1.Experiment{{
		ExperimentId:     createResult.ExperimentId,
		DisplayName:      "ex1",
		Description:      "first experiment",
		CreatedAt:        &timestamp.Timestamp{Seconds: 1},
		LastRunCreatedAt: &timestamp.Timestamp{Seconds: 0},
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}}
	assert.Nil(t, err)
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperimentsByLastRunCreation(t *testing.T) {
	// Create experiment and runs/jobs under it.
	clients, manager, experiment1, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()

	// Create another experiment
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeIdTwo, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: manager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "exp2"}
	experiment2, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	// Create a generic run object
	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)
	genericRun := &apiV2beta1.Run{
		PipelineSource: &apiV2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiV2beta1.RuntimeConfig{
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
	}

	// Create a run in experiment 1
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeIdThree, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	runServer := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	genericRun.DisplayName = "run1"
	genericRun.ExperimentId = experiment1.UUID
	_, err = runServer.CreateRun(nil, &apiV2beta1.CreateRunRequest{Run: genericRun})
	assert.Nil(t, err)

	// Create a run in experiment 2
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeIdFour, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	runServer = NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	genericRun.DisplayName = "run2"
	genericRun.ExperimentId = experiment2.ExperimentId
	_, err = runServer.CreateRun(nil, &apiV2beta1.CreateRunRequest{Run: genericRun})
	assert.Nil(t, err)

	// Expected runs, note that because run 2 in experiment 2
	// was created last, experiment 2 has the latest run execution
	experimentServer := ExperimentServer{resourceManager: manager, options: &ExperimentServerOptions{CollectMetrics: false}}
	expected1 := &apiV2beta1.Experiment{
		ExperimentId:     experiment1.UUID,
		DisplayName:      "exp1",
		Description:      "",
		CreatedAt:        &timestamp.Timestamp{Seconds: 1},
		LastRunCreatedAt: &timestamp.Timestamp{Seconds: 6},
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}
	expected2 := &apiV2beta1.Experiment{
		ExperimentId:     experiment2.ExperimentId,
		DisplayName:      "exp2",
		Description:      "",
		CreatedAt:        &timestamp.Timestamp{Seconds: 5},
		LastRunCreatedAt: &timestamp.Timestamp{Seconds: 8},
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}

	// First list runs sorted by last_run_created_at ascending
	listExperimentsRequest := &apiV2beta1.ListExperimentsRequest{SortBy: "last_run_created_at asc"}
	result, err := experimentServer.ListExperiments(nil, listExperimentsRequest)
	assert.Nil(t, err)
	assert.Equal(t, []*apiV2beta1.Experiment{expected1, expected2}, result.Experiments)

	// Then list runs sorted by last_run_created_at descending, note the order is switched
	listExperimentsRequest = &apiV2beta1.ListExperimentsRequest{SortBy: "last_run_created_at desc"}
	result, err = experimentServer.ListExperiments(nil, listExperimentsRequest)
	assert.Equal(t, []*apiV2beta1.Experiment{expected2, expected1}, result.Experiments)
}

func TestListExperimentsV1_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperimentsV1(nil, &apiv1beta1.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed")
}

func TestListExperiments_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperiments(nil, &apiV2beta1.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed")
}

func TestListExperimentsV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.ListExperimentsV1(ctx, &apiv1beta1.ListExperimentsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
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
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
	)
}

func TestListExperimentsV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}

	resourceReferences := []*apiv1beta1.ResourceReference{
		{
			Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	experiment := &apiv1beta1.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	createResult, err := server.CreateExperimentV1(ctx, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	tests := []struct {
		name                string
		request             *apiv1beta1.ListExperimentsRequest
		wantError           bool
		errorMessage        string
		expectedExperiments []*apiv1beta1.Experiment
	}{
		{
			"Valid",
			&apiv1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "ns1",
				},
			},
			false,
			"",
			[]*apiv1beta1.Experiment{{
				Id:                 createResult.Id,
				Name:               "exp1",
				Description:        "first experiment",
				CreatedAt:          &timestamp.Timestamp{Seconds: 1},
				ResourceReferences: resourceReferences,
				StorageState:       apiv1beta1.Experiment_STORAGESTATE_AVAILABLE,
			}},
		},
		{
			"Valid but empty result",
			&apiv1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "ns2",
				},
			},
			false,
			"",
			[]*apiv1beta1.Experiment{},
		},
		{
			"Invalid resource reference key type",
			&apiv1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_EXPERIMENT,
					Id:   "fake_id",
				},
			},
			true,
			"invalid resource reference key",
			nil,
		},
		{
			"Missing resource reference key",
			&apiv1beta1.ListExperimentsRequest{},
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
			nil,
		},
		{
			"Empty namespace",
			&apiv1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "",
				},
			},
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
			nil,
		},
		{
			"No namespace",
			&apiv1beta1.ListExperimentsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "-",
				},
			},
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
			nil,
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
			} else if !cmp.Equal(tc.expectedExperiments, response.Experiments, cmpopts.EquateEmpty(), protocmp.Transform(), cmpopts.IgnoreFields(apiv1beta1.Experiment{}, "CreatedAt")) {
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
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
				ExperimentId:     createResult.ExperimentId,
				DisplayName:      "exp1",
				Description:      "first experiment",
				CreatedAt:        &timestamp.Timestamp{Seconds: 1},
				LastRunCreatedAt: &timestamp.Timestamp{Seconds: 0},
				Namespace:        "ns1",
				StorageState:     apiV2beta1.Experiment_AVAILABLE,
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
			true,
			"An experiment cannot have an empty namespace in multi-user mode",
			nil,
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

func TestArchiveAndUnarchiveExperimentV1(t *testing.T) {
	// Create experiment and runs/jobs under it.
	clients, manager, experiment, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	runServer := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run1 := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err := runServer.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	runServer = NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run2 := &apiv1beta1.Run{
		Name:               "run2",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err = runServer.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run2})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeUUID, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	jobServer := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job1 := &apiv1beta1.Job{
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}},
		},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err = jobServer.CreateJob(nil, &apiv1beta1.CreateJobRequest{Job: job1})
	assert.Nil(t, err)
	jobs, err := jobServer.ListJobs(nil, &apiv1beta1.ListJobsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, true, jobs.Jobs[0].Enabled)

	// Archive the experiment and thus all runs under it.
	experimentServer := NewExperimentServer(manager, &ExperimentServerOptions{CollectMetrics: false})
	_, err = experimentServer.ArchiveExperimentV1(nil, &apiv1beta1.ArchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err := experimentServer.GetExperimentV1(nil, &apiv1beta1.GetExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiv1beta1.Experiment_STORAGESTATE_ARCHIVED, result.StorageState)
	runs, err := runServer.ListRunsV1(nil, &apiv1beta1.ListRunsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err = jobServer.ListJobs(nil, &apiv1beta1.ListJobsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)

	// Unarchive the experiment and thus all runs under it.
	_, err = experimentServer.UnarchiveExperimentV1(nil, &apiv1beta1.UnarchiveExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	result, err = experimentServer.GetExperimentV1(nil, &apiv1beta1.GetExperimentRequest{Id: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiv1beta1.Experiment_STORAGESTATE_AVAILABLE, result.StorageState)
	runs, err = runServer.ListRunsV1(nil, &apiv1beta1.ListRunsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err = jobServer.ListJobs(nil, &apiv1beta1.ListJobsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)
}

func TestArchiveAndUnarchiveExperiment(t *testing.T) {
	// Create experiment and runs/jobs under it.
	clients, manager, experiment, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	runServer := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run1 := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err := runServer.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run1})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	runServer = NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run2 := &apiv1beta1.Run{
		Name:               "run2",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err = runServer.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run2})
	assert.Nil(t, err)
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeUUID, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	jobServer := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job1 := &apiv1beta1.Job{
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}},
		},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	_, err = jobServer.CreateJob(nil, &apiv1beta1.CreateJobRequest{Job: job1})
	assert.Nil(t, err)

	// Archive the experiment and thus all runs under it.
	experimentServer := NewExperimentServer(manager, &ExperimentServerOptions{CollectMetrics: false})
	_, err = experimentServer.ArchiveExperiment(nil, &apiV2beta1.ArchiveExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	result, err := experimentServer.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiV2beta1.Experiment_ARCHIVED, result.StorageState)
	runs, err := runServer.ListRunsV1(nil, &apiv1beta1.ListRunsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err := jobServer.ListJobs(nil, &apiv1beta1.ListJobsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)

	// Unarchive the experiment and thus all runs under it.
	_, err = experimentServer.UnarchiveExperiment(nil, &apiV2beta1.UnarchiveExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	result, err = experimentServer.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: experiment.UUID})
	assert.Nil(t, err)
	assert.Equal(t, apiV2beta1.Experiment_AVAILABLE, result.StorageState)
	runs, err = runServer.ListRunsV1(nil, &apiv1beta1.ListRunsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs.Runs))
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[0].StorageState)
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED, runs.Runs[1].StorageState)
	jobs, err = jobServer.ListJobs(nil, &apiv1beta1.ListJobsRequest{ResourceReferenceKey: &apiv1beta1.ResourceKey{Id: experiment.UUID, Type: apiv1beta1.ResourceType_EXPERIMENT}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs.Jobs))
	assert.Equal(t, false, jobs.Jobs[0].Enabled)
}

// TestDeleteExperiments_SingleUser tests (1) deleting an existing experiment, and
// deleting an experiment that does not exist in single user mode, for V2 api.
func TestDeleteExperiments_SingleUser(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &apiv1beta1.Experiment{Name: "ex1", Description: "first experiment"}
	resultExperiment, err := server.CreateExperimentV1(nil, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperimentV1(nil, &apiv1beta1.DeleteExperimentRequest{Id: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperimentV1(nil, &apiv1beta1.DeleteExperimentRequest{Id: resultExperiment.Id})
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*apiv1beta1.ResourceReference{
		{
			Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	experiment := &apiv1beta1.Experiment{
		Name:               "ex1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}
	resultExperiment, err := server.CreateExperimentV1(ctx, &apiv1beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperimentV1(ctx, &apiv1beta1.DeleteExperimentRequest{Id: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperimentV1(ctx, &apiv1beta1.DeleteExperimentRequest{Id: resultExperiment.Id})
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
	_, err := server.ListExperimentsV1(ctx, &apiv1beta1.ListExperimentsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
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
