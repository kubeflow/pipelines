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
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/testing/protocmp"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	kfpauth "github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var metricV1 = &apiv1beta1.RunMetric{
	Name:   "metric-1",
	NodeId: "node-1",
	Value: &apiv1beta1.RunMetric_NumberValue{
		NumberValue: 0.88,
	},
	Format: apiv1beta1.RunMetric_RAW,
}

var metric = &apiv2beta1.RunMetric{
	DisplayName: "metric-1",
	NodeId:      "node-1",
	Value: &apiv2beta1.RunMetric_NumberValue{
		NumberValue: 0.88,
	},
	Format: apiv2beta1.RunMetric_RAW,
}

func TestCreateRunV1_V1Params(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{
		{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "pipeline-runner"
	template := expectedRuntimeWorkflow.Spec.Templates[0]
	expectedRuntimeWorkflow.Spec.Templates[0] = template
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000",
		},
	}

	expectedRunDetail := apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			Status:         "Running",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: testWorkflow.ToStringForStore(),
				Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, *runDetail)
}

func TestCreateRunV1_RuntimeParams(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineManifest: v2SpecHelloWorld,
			RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters:   v2RuntimeParams,
				PipelineRoot: "model-pipeline-root",
			},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRunDetail := apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			// Status:         "Running",
			Status: "",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				PipelineManifest: v2SpecHelloWorld,
				RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
					Parameters:   v2RuntimeParams,
					PipelineRoot: "model-pipeline-root",
				},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{},
	}
	assert.EqualValues(t, expectedRunDetail, *runDetail)
}

func TestCreateRunV1Patch(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflowPatch.ToStringForStore(),
			Parameters: []*apiv1beta1.Parameter{
				{Name: "param1", Value: "test-default-bucket"},
				{Name: "param2", Value: "test-project-id"}},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflowPatch.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{
		{Name: "param1", Value: v1alpha1.AnyStringPtr("test-default-bucket")},
		{Name: "param2", Value: v1alpha1.AnyStringPtr("test-project-id")},
	}
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "pipeline-runner"
	template := expectedRuntimeWorkflow.Spec.Templates[0]
	expectedRuntimeWorkflow.Spec.Templates[0] = template
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000",
		},
	}

	expectedRunDetail := apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: testWorkflowPatch.ToStringForStore(),
				Parameters: []*apiv1beta1.Parameter{
					{Name: "param1", Value: "test-default-bucket"},
					{Name: "param2", Value: "test-project-id"},
				},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, *runDetail)
}

func TestCreateRunV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRunV1(ctx, &apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbCreate,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
		Name:      "run1",
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestCreateRunV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, "default-editor")
	defer viper.Set(common.MultiUserMode, "false")
	defer viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, "pipeline-runner")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRunV1(ctx, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	template.AddRuntimeMetadata(expectedRuntimeWorkflow)
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{
		{Name: "param1", Value: v1alpha1.AnyStringPtr("world")}}
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "default-editor" // In multi-user mode, we use default service account.
	template := expectedRuntimeWorkflow.Spec.Templates[0]
	expectedRuntimeWorkflow.Spec.Templates[0] = template
	expectedRuntimeWorkflow.Spec.PodMetadata = &v1alpha1.Metadata{
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000",
		},
	}

	expectedRunDetail := apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			Status:         "Running",
			ServiceAccount: "default-editor",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: testWorkflow.ToStringForStore(),
				Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, *runDetail)
}

func TestCreateRun(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	run := &apiv2beta1.Run{
		DisplayName:  "run1",
		ExperimentId: experiment.UUID,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters:   v2RuntimeParams,
			PipelineRoot: "model-pipeline-root",
		},
	}
	run, err := server.CreateRun(nil, &apiv2beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &apiv2beta1.Run{
		RunId:          "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv2beta1.Run_AVAILABLE,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
		FinishedAt:     &timestamp.Timestamp{},
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters:   v2RuntimeParams,
			PipelineRoot: "model-pipeline-root",
		},
	}
	assert.EqualValues(t, expectedRun, run)
}

func TestGetRun(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	run := &apiv2beta1.Run{
		DisplayName:  "run1",
		ExperimentId: experiment.UUID,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters:   v2RuntimeParams,
			PipelineRoot: "model-pipeline-root",
		},
	}
	returnedRun, err := server.CreateRun(nil, &apiv2beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &apiv2beta1.Run{
		RunId:          "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv2beta1.Run_AVAILABLE,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
		FinishedAt:     &timestamp.Timestamp{},
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters:   v2RuntimeParams,
			PipelineRoot: "model-pipeline-root",
		},
	}

	newRun, err := server.GetRun(nil, &apiv2beta1.GetRunRequest{RunId: returnedRun.RunId})
	assert.Nil(t, err)
	assert.EqualValues(t, expectedRun, newRun)
}

func TestListRunsV1(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &apiv1beta1.Run{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
		FinishedAt:     &timestamp.Timestamp{},
		Status:         "Running",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	listRunsResponse, err := server.ListRunsV1(nil, &apiv1beta1.ListRunsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse.Runs))
	assert.Equal(t, expectedRun, listRunsResponse.Runs[0])
}

func TestListRunsV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	_, err := server.ListRunsV1(ctx, &apiv1beta1.ListRunsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbList,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
	}
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes)),
			"Failed to authorize with namespace resource reference.").Error(),
	)
}

func TestListRunsV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRunV1(ctx, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuns := []*apiv1beta1.Run{{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
		FinishedAt:     &timestamp.Timestamp{},
		Status:         "Running",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}}
	expectedRunsEmpty := []*apiv1beta1.Run{}

	tests := []struct {
		name         string
		request      *apiv1beta1.ListRunsRequest
		wantError    bool
		errorMessage string
		expectedRuns []*apiv1beta1.Run
	}{
		{
			"Valid - filter by experiment",
			&apiv1beta1.ListRunsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_EXPERIMENT,
					Id:   "123e4567-e89b-12d3-a456-426655440000",
				},
			},
			false,
			"",
			expectedRuns,
		},
		{
			"Valid - filter by namespace",
			&apiv1beta1.ListRunsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "ns1",
				},
			},
			false,
			"",
			expectedRuns,
		},
		{
			"Vailid - filter by namespace - no result",
			&apiv1beta1.ListRunsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "no-such-ns",
				},
			},
			false,
			"",
			expectedRunsEmpty,
		},
		{
			"Invalid - no filter",
			&apiv1beta1.ListRunsRequest{},
			true,
			"ListRuns must filter by resource reference",
			nil,
		},
		{
			"Inalid - invalid filter type",
			&apiv1beta1.ListRunsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE,
					Id:   "unknown",
				},
			},
			true,
			"Unrecognized resource reference type",
			nil,
		},
	}

	for _, tc := range tests {
		response, err := server.ListRunsV1(ctx, tc.request)

		if tc.wantError {
			if err == nil {
				t.Errorf("TestListRuns_Multiuser(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestListRuns_Multiusert(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TestListRuns_Multiuser(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedRuns, response.Runs, cmpopts.EquateEmpty(), protocmp.Transform(), cmpopts.IgnoreFields(apiv1beta1.Run{}, "ScheduledAt", "FinishedAt", "CreatedAt")) {
				t.Errorf("TestListRuns_Multiuser(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedRuns, response.Runs)
			}
		}
	}

}

func TestListRuns(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	run := &apiv2beta1.Run{
		DisplayName:  "run1",
		ExperimentId: experiment.UUID,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}
	_, err := server.CreateRun(nil, &apiv2beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &apiv2beta1.Run{
		RunId:          "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv2beta1.Run_AVAILABLE,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
		FinishedAt:     &timestamp.Timestamp{},
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}

	listRunsResponse, err := server.ListRuns(nil, &apiv2beta1.ListRunsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse.Runs))
	assert.Equal(t, expectedRun, listRunsResponse.Runs[0])
}

func TestvalidateCreateRunRequestV1(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestvalidateCreateRunRequestV1_WithPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestvalidateCreateRunRequestV1_EmptyName(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The run name is empty")
}

func TestvalidateCreateRunRequestV1_InvalidPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: referencesOfExperimentAndInvalidPipelineVersion,
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed.")
}

func TestvalidateCreateRunRequestV1_NoExperiment(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: nil,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestvalidateCreateRunRequestV1_NilPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
}

func TestvalidateCreateRunRequestV1_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestvalidateCreateRunRequestV1_InvalidPipelineSpec(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       resource.DefaultFakeUUID,
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestvalidateCreateRunRequestV1_TooMuchParameters(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})

	var params []*apiv1beta1.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &apiv1beta1.Parameter{Name: "param2", Value: "world"})
	}
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       params,
		},
	}

	err := server.validateCreateRunRequestV1(&apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestvalidateCreateRunRequest(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	run := &apiv2beta1.Run{
		DisplayName:  "run1",
		ExperimentId: experiment.UUID,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStruct,
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}
	err := server.validateCreateRunRequest(&apiv2beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestReportRunMetricsV1_RunNotFound(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	_, err := runServer.ReportRunMetricsV1(context.Background(), &apiv1beta1.ReportRunMetricsRequest{
		RunId: "1",
	})
	AssertUserError(t, err, codes.NotFound)
}

func TestReportRunMetricsV1_Succeed(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager, resourceManager, runDetails := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	response, err := runServer.ReportRunMetricsV1(ctx, &apiv1beta1.ReportRunMetricsRequest{
		RunId:   runDetails.UUID,
		Metrics: []*apiv1beta1.RunMetric{metricV1},
	})
	assert.Nil(t, err)
	expectedResponse := &apiv1beta1.ReportRunMetricsResponse{
		Results: []*apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   metricV1.Name,
				MetricNodeId: metricV1.NodeId,
				Status:       apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
		},
	}
	assert.Equal(t, expectedResponse, response)

	run, err := runServer.GetRunV1(ctx, &apiv1beta1.GetRunRequest{
		RunId: runDetails.UUID,
	})
	assert.Nil(t, err)
	assert.Equal(t, []*apiv1beta1.RunMetric{metricV1}, run.GetRun().GetMetrics())
}

func TestReportRunMetricsV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager, resourceManager, runDetails := initWithOneTimeRun(t)
	defer clientManager.Close()
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager = resource.NewResourceManager(clientManager, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	_, err := runServer.ReportRunMetricsV1(ctx, &apiv1beta1.ReportRunMetricsRequest{
		RunId:   runDetails.UUID,
		Metrics: []*apiv1beta1.RunMetric{metricV1},
	})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: runDetails.Namespace,
		Verb:      common.RbacResourceVerbReportMetrics,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
		Name:      runDetails.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)

}

func TestReportRunMetricsV1_PartialFailures(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, runDetail := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	validMetric := metricV1
	invalidNameMetric := &apiv1beta1.RunMetric{
		Name:   "$metric-1",
		NodeId: "node-1",
	}
	invalidNodeIDMetric := &apiv1beta1.RunMetric{
		Name: "metric-1",
	}
	response, err := runServer.ReportRunMetricsV1(context.Background(), &apiv1beta1.ReportRunMetricsRequest{
		RunId:   runDetail.UUID,
		Metrics: []*apiv1beta1.RunMetric{validMetric, invalidNameMetric, invalidNodeIDMetric},
	})
	assert.Nil(t, err)
	expectedResponse := &apiv1beta1.ReportRunMetricsResponse{
		Results: []*apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   validMetric.Name,
				MetricNodeId: validMetric.NodeId,
				Status:       apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
			{
				MetricName:   invalidNameMetric.Name,
				MetricNodeId: invalidNameMetric.NodeId,
				Status:       apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
			{
				MetricName:   invalidNodeIDMetric.Name,
				MetricNodeId: invalidNodeIDMetric.NodeId,
				Status:       apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
		},
	}
	// Message fields, which are not reliable, are ignored from the test.
	for _, result := range response.Results {
		result.Message = ""
	}
	assert.Equal(t, expectedResponse, response)
}

func TestCanAccessRun_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()
	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	apiRun := &apiv1beta1.Run{
		Name: "run1",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*apiv1beta1.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	runDetail, _ := manager.CreateRun(context.Background(), apiRun)

	err := runServer.canAccessRun(ctx, runDetail.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: runDetail.Namespace,
		Verb:      common.RbacResourceVerbGet,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
		Name:      runDetail.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes)).Error(),
	)
}

func TestCanAccessRun_Authorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	apiRun := &apiv1beta1.Run{
		Name: "run1",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*apiv1beta1.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	runDetail, _ := manager.CreateRun(context.Background(), apiRun)

	err := runServer.canAccessRun(ctx, runDetail.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.Nil(t, err)
}

func TestCanAccessRun_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	apiRun := &apiv1beta1.Run{
		Name: "run1",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*apiv1beta1.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	runDetail, _ := manager.CreateRun(context.Background(), apiRun)

	err := runServer.canAccessRun(ctx, runDetail.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.NotNil(t, err)
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzApiResourcesError(kfpauth.IdentityHeaderMissingError).Error(),
	)
}

func TestReadArtifactsV1_Succeed(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	expectedContent := "test"
	filePath := "test/file.txt"
	resourceManager, manager, run := initWithOneTimeRun(t)
	resourceManager.ObjectStore().AddFile([]byte(expectedContent), filePath)
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "workflow-name",
			Namespace:         "ns1",
			UID:               "workflow1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "Workflow",
				Name:       "workflow-name",
				UID:        types.UID(run.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{
			Nodes: map[string]v1alpha1.NodeStatus{
				"node-1": {
					Outputs: &v1alpha1.Outputs{
						Artifacts: []v1alpha1.Artifact{
							{
								Name: "artifact-1",
								ArtifactLocation: v1alpha1.ArtifactLocation{
									S3: &v1alpha1.S3Artifact{
										Key: filePath,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}
	artifact := &apiv1beta1.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	response, err := runServer.ReadArtifactV1(ctx, artifact)
	assert.Nil(t, err)

	expectedResponse := &apiv1beta1.ReadArtifactResponse{
		Data: []byte(expectedContent),
	}
	assert.Equal(t, expectedResponse, response)
}

func TestReadArtifactsV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager, resourceManager, run := initWithOneTimeRun(t)

	//make the following request unauthorized
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager = resource.NewResourceManager(clientManager)

	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}
	artifact := &apiv1beta1.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	_, err := runServer.ReadArtifactV1(ctx, artifact)
	assert.NotNil(t, err)

	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: run.Namespace,
		Verb:      common.RbacResourceVerbReadArtifact,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
		Name:      run.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestReadArtifactsV1_Run_NotFound(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(clientManager)
	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}
	artifact := &apiv1beta1.ReadArtifactRequest{
		RunId:        "Wrong_RUN_UUID",
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	_, err := runServer.ReadArtifactV1(context.Background(), artifact)
	assert.NotNil(t, err)
	err = err.(*util.UserError)

	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestReadArtifactsV1_Resource_NotFound(t *testing.T) {
	_, manager, run := initWithOneTimeRun(t)

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "workflow-name",
			Namespace:         "ns1",
			UID:               "workflow1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "Workflow",
				Name:       "workflow-name",
				UID:        types.UID(run.UUID),
			}},
		},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}
	//`artifactRequest` search for node that does not exist
	artifactRequest := &apiv1beta1.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	_, err = runServer.ReadArtifactV1(context.Background(), artifactRequest)
	assert.NotNil(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestReadArtifacts_Succeed(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	expectedContent := "test"
	filePath := "test/file.txt"
	resourceManager, manager, run := initWithOneTimeRun(t)
	resourceManager.ObjectStore().AddFile([]byte(expectedContent), filePath)
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "workflow-name",
			Namespace:         "ns1",
			UID:               "workflow1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "Workflow",
				Name:       "workflow-name",
				UID:        types.UID(run.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{
			Nodes: map[string]v1alpha1.NodeStatus{
				"node-1": {
					Outputs: &v1alpha1.Outputs{
						Artifacts: []v1alpha1.Artifact{
							{
								Name: "artifact-1",
								ArtifactLocation: v1alpha1.ArtifactLocation{
									S3: &v1alpha1.S3Artifact{
										Key: filePath,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}
	artifact := &apiv2beta1.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	response, err := runServer.ReadArtifact(ctx, artifact)
	assert.Nil(t, err)

	expectedResponse := &apiv2beta1.ReadArtifactResponse{
		Data: []byte(expectedContent),
	}
	assert.Equal(t, expectedResponse, response)
}
