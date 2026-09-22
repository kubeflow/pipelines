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
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/testing/protocmp"
)

func createExperimentServer(resourceManager *resource.ResourceManager) *ExperimentServer {
	return &ExperimentServer{
		BaseExperimentServer: &BaseExperimentServer{
			resourceManager: resourceManager,
			options: &ExperimentServerOptions{
				CollectMetrics: false,
			},
		},
	}
}

func TestCreateExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	result, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId:     DefaultFakeUUID,
		DisplayName:      "ex1",
		Description:      "first experiment",
		CreatedAt:        timestamppb.New(time.Unix(1, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(0, 0)),
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to add experiment to experiment table")
}

func TestCreateExperiment_EmptyName(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid input error: Experiment must have a non-empty name")
}

func TestCreateExperiment_LengthValidation(t *testing.T) {
	const (
		maxName = 128
		maxNS   = 63
	)

	longName := strings.Repeat("n", maxName+1)
	longNS := strings.Repeat("s", maxNS+1)

	tests := []struct {
		multiUser bool
		name      string
		namespace string
		wantErr   bool
		wantMsg   string
	}{
		{false, "good-name", "good_namespace", false, ""},
		{false, longName, "good_namespace", true, "Experiment.Name length cannot exceed"},
		{true, "good-name", "good_namespace", false, ""},
		{true, "good-name", longNS, true, "Experiment.Namespace length cannot exceed"},
		{true, longName, "good_namespace", true, "Experiment.Name length cannot exceed"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("multi=%t_nameLen=%d_nsLen=%d",
			tc.multiUser, len(tc.name), len(tc.namespace)),
			func(t *testing.T) {
				if tc.multiUser {
					viper.Set(common.MultiUserMode, "true")
				} else {
					viper.Set(common.MultiUserMode, "false")
				}
				defer viper.Set(common.MultiUserMode, "false")

				md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
				ctx := metadata.NewIncomingContext(context.Background(), md)

				clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
				resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
				server := createExperimentServer(resourceManager)

				req := &apiV2beta1.CreateExperimentRequest{
					Experiment: &apiV2beta1.Experiment{
						DisplayName: tc.name,
						Namespace:   tc.namespace,
					},
				}
				_, err := server.CreateExperiment(ctx, req)
				if tc.wantErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.wantMsg)
				} else {
					assert.NoError(t, err)
				}
			})
	}
}

func TestCreateExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, resourceManager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := createExperimentServer(resourceManager)
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

func TestCreateExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)

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
				LastRunCreatedAt: timestamppb.New(time.Unix(0, 0)),
				Namespace:        "ns1",
			},
			&apiV2beta1.Experiment{
				ExperimentId:     DefaultFakeUUID,
				DisplayName:      "exp1",
				Description:      "first experiment",
				LastRunCreatedAt: timestamppb.New(time.Unix(0, 0)),
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

func TestGetExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: createResult.ExperimentId})
	assert.Nil(t, err)
	expectedExperiment := &apiV2beta1.Experiment{
		ExperimentId:     createResult.ExperimentId,
		DisplayName:      "ex1",
		Description:      "first experiment",
		CreatedAt:        timestamppb.New(time.Unix(1, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(0, 0)),
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: createResult.ExperimentId})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestGetExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := createExperimentServer(manager)

	_, err := server.GetExperiment(ctx, &apiV2beta1.GetExperimentRequest{ExperimentId: experiment.UUID})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
	)
}

func TestGetExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
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
		CreatedAt:        timestamppb.New(time.Unix(1, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(0, 0)),
		Namespace:        "ns1",
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestListExperiments(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperiments(nil, &apiV2beta1.ListExperimentsRequest{})
	expectedExperiment := []*apiV2beta1.Experiment{{
		ExperimentId:     createResult.ExperimentId,
		DisplayName:      "ex1",
		Description:      "first experiment",
		CreatedAt:        timestamppb.New(time.Unix(1, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(0, 0)),
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}}
	assert.Nil(t, err)
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperimentsByLastRunCreation(t *testing.T) {
	// Create experiment and runs/jobs under it.
	clients, _, experiment1, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()

	// Create another experiment
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeIdTwo, nil))
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(manager)
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
	experimentServer := createExperimentServer(manager)
	expected1 := &apiV2beta1.Experiment{
		ExperimentId:     experiment1.UUID,
		DisplayName:      "exp1",
		Description:      "",
		CreatedAt:        timestamppb.New(time.Unix(1, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(5, 0)),
		StorageState:     apiV2beta1.Experiment_AVAILABLE,
		Namespace:        "",
	}
	expected2 := &apiV2beta1.Experiment{
		ExperimentId:     experiment2.ExperimentId,
		DisplayName:      "exp2",
		Description:      "",
		CreatedAt:        timestamppb.New(time.Unix(4, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(7, 0)),
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
	assert.NoError(t, err)
}

func TestListExperiments_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperiments(nil, &apiV2beta1.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed")
}

func TestListExperiments_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := createExperimentServer(manager)
	_, err := server.ListExperiments(ctx, &apiV2beta1.ListExperimentsRequest{Namespace: "ns1"})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason: this is not allowed",
	)
}

func TestListExperiments_Multiuser_NoDefault(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
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
				CreatedAt:        timestamppb.New(time.Unix(1, 0)),
				LastRunCreatedAt: timestamppb.New(time.Unix(0, 0)),
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

// TestDeleteExperiments_SingleUser tests (1) deleting an existing experiment, and
// deleting an experiment that does not exist in single user mode, for V2 api.
func TestDeleteExperiments_SingleUser(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment"}
	resultExperiment, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperiment(nil, &apiV2beta1.DeleteExperimentRequest{ExperimentId: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperiment(nil, &apiV2beta1.DeleteExperimentRequest{ExperimentId: resultExperiment.ExperimentId})
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
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{DisplayName: "ex1", Description: "first experiment", Namespace: "ns1"}
	resultExperiment, err := server.CreateExperiment(ctx, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	_, err = server.DeleteExperiment(ctx, &apiV2beta1.DeleteExperimentRequest{ExperimentId: "ex2"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = server.DeleteExperiment(ctx, &apiV2beta1.DeleteExperimentRequest{ExperimentId: resultExperiment.ExperimentId})
	assert.Nil(t, err)
}

func TestGetExperiment_JsonOmitEmpty(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createExperimentServer(resourceManager)
	experiment := &apiV2beta1.Experiment{
		DisplayName: "exp1",
		Description: "test description",
	}

	result, err := server.CreateExperiment(nil, &apiV2beta1.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)

	getResult, err := server.GetExperiment(nil, &apiV2beta1.GetExperimentRequest{ExperimentId: result.ExperimentId})
	assert.Nil(t, err)

	// Convert to JSON using the custom marshaler used by runtime servers
	customMarshaler := common.CustomMarshaler()
	jsonBytes, err := customMarshaler.Marshal(getResult)
	assert.Nil(t, err)

	// Verify JSON doesn't contain empty/unset fields
	jsonString := string(jsonBytes)
	assert.Contains(t, jsonString, "description")
	assert.Contains(t, jsonString, "display_name")
	assert.Contains(t, jsonString, "experiment_id") // created by server
	assert.NotContains(t, jsonString, "namespace")  // if we don't specify this it should be omitted
}
