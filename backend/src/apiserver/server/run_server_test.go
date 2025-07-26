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
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

func createRunServerV1(resourceManager *resource.ResourceManager) *RunServerV1 {
	return &RunServerV1{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false},
		},
	}
}

func createRunServer(resourceManager *resource.ResourceManager) *RunServer {
	return &RunServer{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false},
		},
	}
}

var metricV1 = &apiv1beta1.RunMetric{
	Name:   "metric-1",
	NodeId: "node-1",
	Value: &apiv1beta1.RunMetric_NumberValue{
		NumberValue: 0.88,
	},
	Format: apiv1beta1.RunMetric_RAW,
}

func TestCreateRunV1_empty_name(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Nil(t, runDetail)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestCreateRunV1_invalid_pipeline_version(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId: "invalid_id",
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Nil(t, runDetail)
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateRunV1_no_experiment(t *testing.T) {
	clients, manager, exp := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name: "run1",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	manager.SetDefaultExperimentId(exp.UUID)
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
	assert.NotNil(t, runDetail)
	check := false
	for _, ref := range runDetail.GetRun().GetResourceReferences() {
		if ref.GetKey().GetId() == exp.UUID {
			check = true
			break
		}
	}
	assert.True(t, check)
	assert.Empty(t, runDetail.GetRun().GetError())
}

func TestCreateRunV1_no_pipeline_source(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Nil(t, runDetail)
	assert.Contains(t, err.Error(), "Failed to fetch a template with an empty pipeline spec manifest")
}

func TestCreateRunV1_invalid_spec(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: "invalid pipeline spec",
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Nil(t, runDetail)
	assert.Contains(t, err.Error(), "unknown template format")
}

func TestCreateRunV1_too_many_params(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
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
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Nil(t, runDetail)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestCreateRunV1_pipeline(t *testing.T) {
	clients, manager, exp, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name: "run1",
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_EXPERIMENT,
					Id:   DefaultFakeUUID,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_PIPELINE,
					Id:   DefaultFakeUUID,
				},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
	assert.Equal(t, DefaultFakeUUID, runDetail.Run.PipelineSpec.PipelineId)
	assert.Equal(t, apiv1beta1.ResourceType_EXPERIMENT, runDetail.Run.ResourceReferences[0].Key.Type)
	assert.Equal(t, exp.UUID, runDetail.Run.ResourceReferences[0].Key.Id)
}

func TestCreateRunV1_pipelineversion(t *testing.T) {
	clients, manager, exp, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
	assert.Equal(t, DefaultFakeUUID, runDetail.Run.PipelineSpec.PipelineId)
	assert.Equal(t, apiv1beta1.ResourceType_EXPERIMENT, runDetail.Run.ResourceReferences[0].Key.Type)
	assert.Equal(t, exp.UUID, runDetail.Run.ResourceReferences[0].Key.Id)
}

func TestCreateRunV1_Manifest_and_pipeline_version(t *testing.T) {
	clients, manager, exp, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
	assert.Equal(t, DefaultFakeUUID, runDetail.Run.PipelineSpec.PipelineId)
	assert.Equal(t, apiv1beta1.ResourceType_EXPERIMENT, runDetail.Run.ResourceReferences[0].Key.Type)
	assert.Equal(t, exp.UUID, runDetail.Run.ResourceReferences[0].Key.Id)
	assert.Equal(t, testWorkflow.ToStringForStore(), runDetail.Run.PipelineSpec.WorkflowManifest)
}

func TestCreateRunV1_V1Params(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
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
		{Name: "param1", Value: v1alpha1.AnyStringPtr("world")},
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

	expectedRunDetail := &apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      timestamppb.New(time.Unix(4, 0)),
			ScheduledAt:    timestamppb.New(time.Unix(4, 0)),
			FinishedAt:     timestamppb.New(time.Unix(0, 0)),
			Status:         "Pending",
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

	matched := 0
	for _, resRef := range expectedRunDetail.GetRun().GetResourceReferences() {
		for _, resRef2 := range runDetail.GetRun().GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(expectedRunDetail.GetRun().GetResourceReferences()), matched)
	expectedRunDetail.Run.ResourceReferences = runDetail.GetRun().GetResourceReferences()

	expectedRunDetail.Run.PipelineSpec.PipelineId = runDetail.GetRun().GetPipelineSpec().GetPipelineId()
	expectedRunDetail.Run.PipelineSpec.PipelineName = runDetail.GetRun().GetPipelineSpec().GetPipelineName()
	expectedRunDetail.Run.CreatedAt = runDetail.GetRun().GetCreatedAt()
	expectedRunDetail.Run.ScheduledAt = runDetail.GetRun().GetScheduledAt()
	expectedRunDetail.Run.FinishedAt = runDetail.GetRun().GetFinishedAt()

	assert.Equal(t, expectedRunDetail, runDetail)
}

func TestCreateRunV1_RuntimeParams_V2Spec(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": {Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": {Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": {Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": {Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineManifest: v2SpecHelloWorldParams,
			RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters:   v2RuntimeParams,
				PipelineRoot: "model-pipeline-root",
			},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRunDetail := &apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      timestamppb.New(time.Unix(2, 0)),
			ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
			FinishedAt:     timestamppb.New(time.Unix(0, 0)),
			Status:         "Pending",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				PipelineManifest: v2SpecHelloWorldParams,
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
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			PipelineManifest: v2SpecHelloWorld,
		},
	}

	matched := 0
	for _, resRef := range expectedRunDetail.GetRun().GetResourceReferences() {
		for _, resRef2 := range runDetail.GetRun().GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(expectedRunDetail.GetRun().GetResourceReferences()), matched)
	expectedRunDetail.Run.ResourceReferences = runDetail.GetRun().GetResourceReferences()

	expectedRunDetail.Run.PipelineSpec.PipelineId = runDetail.GetRun().GetPipelineSpec().GetPipelineId()
	expectedRunDetail.Run.PipelineSpec.PipelineName = runDetail.GetRun().GetPipelineSpec().GetPipelineName()
	expectedRunDetail.Run.CreatedAt = runDetail.GetRun().GetCreatedAt()
	expectedRunDetail.Run.ScheduledAt = runDetail.GetRun().GetScheduledAt()
	expectedRunDetail.Run.FinishedAt = runDetail.GetRun().GetFinishedAt()
	expectedRunDetail.PipelineRuntime = runDetail.GetPipelineRuntime()

	assert.EqualValues(t, expectedRunDetail, runDetail)
}

func TestCreateRunV1Patch(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflowPatch.ToStringForStore(),
			Parameters: []*apiv1beta1.Parameter{
				{Name: "param1", Value: "test-default-bucket"},
				{Name: "param2", Value: "test-project-id"},
			},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)
	expectedRunDetail := &apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			Status:         "Pending",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      timestamppb.New(time.Unix(2, 0)),
			ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
			FinishedAt:     timestamppb.New(time.Unix(0, 0)),
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
			WorkflowManifest: "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"name\":\"workflow-name\",\"namespace\":\"ns1\",\"uid\":\"workflow2\",\"creationTimestamp\":null,\"labels\":{\"pipeline/runid\":\"123e4567-e89b-12d3-a456-426655440000\"},\"annotations\":{\"pipelines.kubeflow.org/run_name\":\"run1\"}},\"spec\":{\"templates\":[{\"name\":\"testy\",\"inputs\":{},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"},\"labels\":{\"pipelines.kubeflow.org/cache_enabled\":\"true\"}},\"container\":{\"name\":\"\",\"image\":\"docker/whalesay\",\"command\":[\"cowsay\"],\"args\":[\"hello world\"],\"resources\":{}}}],\"entrypoint\":\"testy\",\"arguments\":{\"parameters\":[{\"name\":\"param1\",\"value\":\"test-default-bucket\"},{\"name\":\"param2\",\"value\":\"test-project-id\"}]},\"serviceAccountName\":\"pipeline-runner\",\"podMetadata\":{\"labels\":{\"pipeline/runid\":\"123e4567-e89b-12d3-a456-426655440000\"}}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		},
	}

	matched := 0
	for _, resRef := range expectedRunDetail.GetRun().GetResourceReferences() {
		for _, resRef2 := range runDetail.GetRun().GetResourceReferences() {
			if resRef2.Key.Type == apiv1beta1.ResourceType_NAMESPACE {
				assert.Equal(t, resRef2.Key.Id, experiment.Namespace)
			}
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(expectedRunDetail.GetRun().GetResourceReferences()), matched)
	expectedRunDetail.Run.ResourceReferences = runDetail.GetRun().GetResourceReferences()

	expectedRunDetail.Run.PipelineSpec.PipelineId = runDetail.GetRun().GetPipelineSpec().GetPipelineId()
	expectedRunDetail.Run.PipelineSpec.PipelineName = runDetail.GetRun().GetPipelineSpec().GetPipelineName()
	expectedRunDetail.Run.CreatedAt = runDetail.GetRun().GetCreatedAt()
	expectedRunDetail.Run.ScheduledAt = runDetail.GetRun().GetScheduledAt()
	expectedRunDetail.Run.FinishedAt = runDetail.GetRun().GetFinishedAt()

	assert.Equal(t, expectedRunDetail, runDetail)
}

func TestCreateRunV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := createRunServerV1(manager)
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
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason",
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
	server := createRunServerV1(manager)
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
		{Name: "param1", Value: v1alpha1.AnyStringPtr("world")},
	}
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

	expectedRunDetail := &apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			Status:         "Pending",
			ServiceAccount: "default-editor",
			StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      timestamppb.New(time.Unix(4, 0)),
			ScheduledAt:    timestamppb.New(time.Unix(4, 0)),
			FinishedAt:     timestamppb.New(time.Unix(0, 0)),
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: testWorkflow.ToStringForStore(),
				Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
				},
				{
					Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
					Name: "ns1", Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}

	matched := 0
	for _, resRef := range expectedRunDetail.GetRun().GetResourceReferences() {
		for _, resRef2 := range runDetail.GetRun().GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(expectedRunDetail.GetRun().GetResourceReferences()), matched)
	expectedRunDetail.Run.ResourceReferences = runDetail.GetRun().GetResourceReferences()

	expectedRunDetail.Run.PipelineSpec.PipelineId = runDetail.GetRun().GetPipelineSpec().GetPipelineId()
	expectedRunDetail.Run.PipelineSpec.PipelineName = runDetail.GetRun().GetPipelineSpec().GetPipelineName()
	expectedRunDetail.Run.CreatedAt = runDetail.GetRun().GetCreatedAt()
	expectedRunDetail.Run.ScheduledAt = runDetail.GetRun().GetScheduledAt()
	expectedRunDetail.Run.FinishedAt = runDetail.GetRun().GetFinishedAt()

	assert.Equal(t, expectedRunDetail, runDetail)
}

func TestRunServer_CreateRun_SingleUser(t *testing.T) {
	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)
	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorldParams), pipelineSpecStruct)
	runtimeParams := map[string]*structpb.Value{
		"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": {Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": {Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": {Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": {Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}
	runtimeParamsWithExtra := map[string]*structpb.Value{
		"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": {Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": {Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": {Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": {Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
		"param6": structpb.NewStringValue("hello"),
		"param7": structpb.NewStringValue("world"),
	}
	tests := []struct {
		name    string
		args    *apiv2beta1.CreateRunRequest
		want    *apiv2beta1.Run
		wantErr bool
		errMsg  string
	}{
		{
			"Valid V2 - basic",
			&apiv2beta1.CreateRunRequest{
				Run: &apiv2beta1.Run{
					DisplayName:  "run1",
					ExperimentId: DefaultFakeUUID,
					PipelineSource: &apiv2beta1.Run_PipelineSpec{
						PipelineSpec: pipelineSpecStruct,
					},
					RuntimeConfig: &apiv2beta1.RuntimeConfig{
						Parameters:   runtimeParams,
						PipelineRoot: "model-pipeline-root",
					},
				},
			},
			&apiv2beta1.Run{
				RunId:          "123e4567-e89b-12d3-a456-426655440000",
				ExperimentId:   DefaultFakeUUID,
				DisplayName:    "run1",
				ServiceAccount: "pipeline-runner",
				StorageState:   apiv2beta1.Run_AVAILABLE,
				CreatedAt:      timestamppb.New(time.Unix(2, 0)),
				ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
				FinishedAt:     timestamppb.New(time.Unix(0, 0)),
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: nil,
				},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters:   runtimeParams,
					PipelineRoot: "model-pipeline-root",
				},
				State: apiv2beta1.RuntimeState_PENDING,
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: timestamppb.New(time.Unix(3, 0)),
						State:      apiv2beta1.RuntimeState_PENDING,
					},
				},
			},
			false,
			"",
		},
		{
			"Valid V2 - no experiment",
			&apiv2beta1.CreateRunRequest{
				Run: &apiv2beta1.Run{
					DisplayName: "run1",
					PipelineSource: &apiv2beta1.Run_PipelineSpec{
						PipelineSpec: pipelineSpecStruct,
					},
					RuntimeConfig: &apiv2beta1.RuntimeConfig{
						Parameters:   runtimeParams,
						PipelineRoot: "model-pipeline-root",
					},
				},
			},
			&apiv2beta1.Run{
				RunId:          "123e4567-e89b-12d3-a456-426655440000",
				ExperimentId:   DefaultFakeUUID,
				DisplayName:    "run1",
				ServiceAccount: "pipeline-runner",
				StorageState:   apiv2beta1.Run_AVAILABLE,
				CreatedAt:      timestamppb.New(time.Unix(2, 0)),
				ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
				FinishedAt:     timestamppb.New(time.Unix(0, 0)),
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: nil,
				},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters:   runtimeParams,
					PipelineRoot: "model-pipeline-root",
				},
				State: apiv2beta1.RuntimeState_PENDING,
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: timestamppb.New(time.Unix(3, 0)),
						State:      apiv2beta1.RuntimeState_PENDING,
					},
				},
			},
			false,
			"",
		},
		{
			"Invalid V2 - missing parameters",
			&apiv2beta1.CreateRunRequest{
				Run: &apiv2beta1.Run{
					DisplayName:  "run1",
					ExperimentId: DefaultFakeUUID,
					PipelineSource: &apiv2beta1.Run_PipelineSpec{
						PipelineSpec: pipelineSpecStruct,
					},
					RuntimeConfig: &apiv2beta1.RuntimeConfig{
						Parameters:   map[string]*structpb.Value{},
						PipelineRoot: "model-pipeline-root",
					},
				},
			},
			nil,
			true,
			"is not optional, yet has neither default value nor user provided value",
		},
		{
			"Invalid V2 - extra parameter",
			&apiv2beta1.CreateRunRequest{
				Run: &apiv2beta1.Run{
					DisplayName:  "run1",
					ExperimentId: DefaultFakeUUID,
					PipelineSource: &apiv2beta1.Run_PipelineSpec{
						PipelineSpec: pipelineSpecStruct,
					},
					RuntimeConfig: &apiv2beta1.RuntimeConfig{
						Parameters:   runtimeParamsWithExtra,
						PipelineRoot: "model-pipeline-root",
					},
				},
			},
			nil,
			true,
			"parameter(s) provided are not required by pipeline:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clients, manager, _ := initWithExperiment(t)
			server := createRunServer(manager)
			server.resourceManager.SetDefaultExperimentId(DefaultFakeUUID)
			got, err := server.CreateRun(context.Background(), tt.args)
			if tt.wantErr {
				assert.Nil(t, got)
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				tt.want.PipelineSource = &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: got.GetPipelineSpec(),
				}
				assert.EqualValues(t, tt.want, got)
			}
			clients.Close()
		})
	}
}

func TestGetRunV1(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name: "run1",
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_EXPERIMENT,
					Id:   DefaultFakeUUID,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_PIPELINE,
					Id:   DefaultFakeUUID,
				},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		},
	}
	runDetail, err := server.CreateRunV1(nil, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	returnedRun, err := server.GetRunV1(context.Background(), &apiv1beta1.GetRunRequest{RunId: runDetail.GetRun().GetId()})
	assert.Nil(t, err)
	assert.Equal(t, runDetail, returnedRun)

	returnedRun2, err := server.GetRunV1(context.Background(), &apiv1beta1.GetRunRequest{RunId: "wrong-id"})
	assert.NotNil(t, err)
	assert.Nil(t, returnedRun2)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetRun(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServer(manager)

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": {Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": {Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": {Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": {Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorldParams), pipelineSpecStruct)

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
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
		FinishedAt:     timestamppb.New(time.Unix(0, 0)),
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: returnedRun.GetPipelineSpec(),
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters:   v2RuntimeParams,
			PipelineRoot: "model-pipeline-root",
		},
		State: apiv2beta1.RuntimeState_PENDING,
		StateHistory: []*apiv2beta1.RuntimeStatus{
			{
				UpdateTime: timestamppb.New(time.Unix(3, 0)),
				State:      apiv2beta1.RuntimeState_PENDING,
			},
		},
	}

	newRun, err := server.GetRun(nil, &apiv2beta1.GetRunRequest{RunId: returnedRun.RunId})
	assert.Nil(t, err)
	assert.EqualValues(t, expectedRun, newRun)
}

func TestListRunsV1(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRunV1(context.Background(), &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &apiv1beta1.Run{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
		FinishedAt:     timestamppb.New(time.Unix(0, 0)),
		Status:         "Pending",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	listRunsResponse, err := server.ListRunsV1(context.Background(), &apiv1beta1.ListRunsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse.Runs))

	matched := 0
	for _, resRef := range expectedRun.GetResourceReferences() {
		for _, resRef2 := range listRunsResponse.Runs[0].GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(expectedRun.GetResourceReferences()), matched)
	expectedRun.ResourceReferences = listRunsResponse.Runs[0].GetResourceReferences()
	expectedRun.PipelineSpec.PipelineId = listRunsResponse.Runs[0].GetPipelineSpec().GetPipelineId()
	expectedRun.PipelineSpec.PipelineName = listRunsResponse.Runs[0].GetPipelineSpec().GetPipelineName()
	expectedRun.PipelineSpec.PipelineManifest = listRunsResponse.Runs[0].GetPipelineSpec().GetPipelineManifest()
	expectedRun.CreatedAt = listRunsResponse.Runs[0].GetCreatedAt()
	expectedRun.ScheduledAt = listRunsResponse.Runs[0].GetScheduledAt()
	expectedRun.FinishedAt = listRunsResponse.Runs[0].GetFinishedAt()
	assert.Equal(t, expectedRun, listRunsResponse.Runs[0])

	listRunsResponse2, err := server.ListRunsV1(context.Background(), &apiv1beta1.ListRunsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse2.Runs))
	assert.Equal(t, expectedRun, listRunsResponse2.Runs[0])
	assert.Equal(t, listRunsResponse.Runs[0], listRunsResponse2.Runs[0])
}

func TestListRunsV1_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
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

	expectedRun := &apiv1beta1.Run{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
		FinishedAt:     timestamppb.New(time.Unix(0, 0)),
		Status:         "Pending",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: experiment.Namespace},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	// Empty request must fail in multi-user mode
	listRunsResponse, err := server.ListRunsV1(ctx, &apiv1beta1.ListRunsRequest{})
	assert.NotNil(t, err)
	assert.Nil(t, listRunsResponse)

	listRunsResponse2, err := server.ListRunsV1(ctx, &apiv1beta1.ListRunsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse2.Runs))
	assert.Equal(t, expectedRun, listRunsResponse2.Runs[0])
}

func TestListRunsV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := createRunServerV1(manager)
	_, err := server.ListRunsV1(ctx, &apiv1beta1.ListRunsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"Check if you have permission to access namespace ns1",
	)
}

func TestListRunsV1_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServerV1(manager)
	run := &apiv1beta1.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	createdRun, err := server.CreateRunV1(ctx, &apiv1beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuns := []*apiv1beta1.Run{{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv1beta1.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
		FinishedAt:     timestamppb.New(time.Unix(0, 0)),
		Status:         "Pending",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       createdRun.Run.PipelineSpec.GetPipelineId(),
			PipelineName:     createdRun.Run.PipelineSpec.GetPipelineName(),
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: experiment.Namespace},
				Relationship: apiv1beta1.Relationship_OWNER,
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
			"Valid - filter by namespace - no result",
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
			"Invalid - empty request",
			&apiv1beta1.ListRunsRequest{},
			true,
			"A run cannot have an empty namespace in multi-user mode",
			nil,
		},
		{
			"Invalid - invalid filter type",
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
	server := createRunServer(manager)
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
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
	}
	createdRun, err := server.CreateRun(nil, &apiv2beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &apiv2beta1.Run{
		RunId:          "123e4567-e89b-12d3-a456-426655440000",
		ExperimentId:   experiment.UUID,
		DisplayName:    "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   apiv2beta1.Run_AVAILABLE,
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		ScheduledAt:    timestamppb.New(time.Unix(2, 0)),
		FinishedAt:     timestamppb.New(time.Unix(0, 0)),
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: createdRun.GetPipelineSpec(),
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		State: apiv2beta1.RuntimeState_PENDING,
		StateHistory: []*apiv2beta1.RuntimeStatus{
			{
				UpdateTime: timestamppb.New(time.Unix(3, 0)),
				State:      apiv2beta1.RuntimeState_PENDING,
			},
		},
	}

	listRunsResponse, err := server.ListRuns(nil, &apiv2beta1.ListRunsRequest{
		ExperimentId: experiment.UUID,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse.Runs))
	listRunsResponse.Runs[0].RuntimeConfig.Parameters = map[string]*structpb.Value{
		"param1": structpb.NewStringValue("world"),
	}
	assert.Equal(t, expectedRun, listRunsResponse.Runs[0])
}

func TestReportRunMetricsV1_RunNotFound(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := createRunServerV1(resourceManager)

	_, err := runServer.ReportRunMetricsV1(context.Background(), &apiv1beta1.ReportRunMetricsRequest{
		RunId: "1",
	})
	AssertUserError(t, err, codes.NotFound)
}

func TestReportRunMetricsV1_Succeed_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager, resourceManager, runDetails := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := createRunServerV1(resourceManager)

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
	resourceManager = resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	runServer := createRunServerV1(resourceManager)

	_, err := runServer.ReportRunMetricsV1(ctx, &apiv1beta1.ReportRunMetricsRequest{
		RunId:   runDetails.UUID,
		Metrics: []*apiv1beta1.RunMetric{metricV1},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized",
	)
}

func TestReportRunMetricsV1_PartialFailures(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, runDetail := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := createRunServerV1(resourceManager)

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

// Test length validation for ReportRunMetricsV1: Name and NodeId exceeding max length should return INVALID_ARGUMENT.
func TestReportRunMetricsV1_LengthValidation(t *testing.T) {
	httpServer := getMockServer(t)
	defer httpServer.Close()

	clientManager, resourceManager, runDetail := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := createRunServerV1(resourceManager)

	// Prepare metrics: one valid, one with Name overflow, one with NodeId overflow.
	validMetric := metricV1
	longName := strings.Repeat("a", 192)
	nameOverflow := &apiv1beta1.RunMetric{
		Name:   longName,
		NodeId: validMetric.NodeId,
		Value:  validMetric.Value,
		Format: validMetric.Format,
	}
	longNodeID := strings.Repeat("a", 192)
	nodeOverflow := &apiv1beta1.RunMetric{
		Name:   validMetric.Name,
		NodeId: longNodeID,
		Value:  validMetric.Value,
		Format: validMetric.Format,
	}

	response, err := runServer.ReportRunMetricsV1(context.Background(), &apiv1beta1.ReportRunMetricsRequest{
		RunId:   runDetail.UUID,
		Metrics: []*apiv1beta1.RunMetric{validMetric, nameOverflow, nodeOverflow},
	})
	assert.Nil(t, err)

	// Check statuses: valid metric OK, others INVALID_ARGUMENT.
	results := response.GetResults()
	assert.Equal(t, apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK, results[0].GetStatus())
	assert.Equal(t, apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT, results[1].GetStatus())
	assert.Equal(t, apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT, results[2].GetStatus())
}

func TestCanAccessRun_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()
	runServer := createRunServerV1(manager)

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
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	modelRun, _ := toModelRun(apiRun)
	modelRun.Namespace = experiment.Namespace
	runDetail, err := manager.CreateRun(ctx, modelRun)
	assert.Nil(t, err)

	err = runServer.canAccessRun(ctx, runDetail.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"User 'user@google.com' is not authorized with reason: this is not allowed",
	)
}

func TestCanAccessRun_Authorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, oneTimeRun := initWithOneTimeRun(t)
	defer clients.Close()
	runServer := createRunServerV1(manager)

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	err := runServer.canAccessRun(ctx, oneTimeRun.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.Nil(t, err)
}

func TestCanAccessRun_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	runServer := createRunServerV1(manager)

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
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: experiment.Namespace},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	modelRun, err := toModelRun(apiRun)
	assert.Nil(t, err)
	runDetail, _ := manager.CreateRun(context.Background(), modelRun)
	assert.Nil(t, err)

	err = runServer.canAccessRun(ctx, runDetail.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"Unauthenticated: Request header error: there is no user identity header",
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
	resourceManager.ObjectStore().AddFile(context.TODO(), []byte(expectedContent), filePath)
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runServer := createRunServerV1(manager)
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

	clientManager, _, run := initWithOneTimeRun(t)

	// make the following request unauthorized
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	runServer := createRunServerV1(resourceManager)
	artifact := &apiv1beta1.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	_, err := runServer.ReadArtifactV1(ctx, artifact)
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"User 'user@google.com' is not authorized with reason: this is not allowed",
	)
}

func TestReadArtifactsV1_Run_NotFound(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	runServer := createRunServerV1(manager)
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runServer := createRunServerV1(manager)
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
	resourceManager.ObjectStore().AddFile(context.TODO(), []byte(expectedContent), filePath)
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
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	runServer := createRunServer(manager)
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

func TestRetryRun(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createRunServer(manager)

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": {Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": {Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": {Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": {Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorldParams), pipelineSpecStruct)

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

	_, err = server.RetryRun(nil, &apiv2beta1.RetryRunRequest{RunId: run.RunId})
}
