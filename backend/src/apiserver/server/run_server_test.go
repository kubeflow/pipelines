package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/testing/protocmp"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	kfpauth "github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
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

var metric = &api.RunMetric{
	Name:   "metric-1",
	NodeId: "node-1",
	Value: &api.RunMetric_NumberValue{
		NumberValue: 0.88,
	},
	Format: api.RunMetric_RAW,
}

func TestCreateRun_V1Params(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	resource.AddRuntimeMetadata(expectedRuntimeWorkflow)
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

	expectedRunDetail := api.RunDetail{
		Run: &api.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			Status:         "Running",
			PipelineSpec: &api.PipelineSpec{
				WorkflowManifest: testWorkflow.ToStringForStore(),
				Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
			},
			ResourceReferences: []*api.ResourceReference{
				{
					Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: api.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &api.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, *runDetail)
}

func TestCreateRun_RuntimeParams(t *testing.T) {
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

	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			PipelineManifest: v2SpecHelloWorld,
			RuntimeConfig: &api.PipelineSpec_RuntimeConfig{
				Parameters:   v2RuntimeParams,
				PipelineRoot: "model-pipeline-root",
			},
		},
	}
	runDetail, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRunDetail := api.RunDetail{
		Run: &api.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			// Status:         "Running",
			Status: "",
			PipelineSpec: &api.PipelineSpec{
				PipelineManifest: v2SpecHelloWorld,
				RuntimeConfig: &api.PipelineSpec_RuntimeConfig{
					Parameters:   v2RuntimeParams,
					PipelineRoot: "model-pipeline-root",
				},
			},
			ResourceReferences: []*api.ResourceReference{
				{
					Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: api.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &api.PipelineRuntime{},
	}
	assert.EqualValues(t, expectedRunDetail, *runDetail)
}

func TestCreateRunPatch(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflowPatch.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "test-default-bucket"},
				{Name: "param2", Value: "test-project-id"}},
		},
	}
	runDetail, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflowPatch.DeepCopy()
	resource.AddRuntimeMetadata(expectedRuntimeWorkflow)
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

	expectedRunDetail := api.RunDetail{
		Run: &api.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			PipelineSpec: &api.PipelineSpec{
				WorkflowManifest: testWorkflowPatch.ToStringForStore(),
				Parameters: []*api.Parameter{
					{Name: "param1", Value: "test-default-bucket"},
					{Name: "param2", Value: "test-project-id"},
				},
			},
			ResourceReferences: []*api.ResourceReference{
				{
					Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: api.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &api.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, *runDetail)
}

func TestCreateRun_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: run})
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

func TestCreateRun_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, "default-editor")
	defer viper.Set(common.MultiUserMode, "false")
	defer viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, "pipeline-runner")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	resource.AddRuntimeMetadata(expectedRuntimeWorkflow)
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

	expectedRunDetail := api.RunDetail{
		Run: &api.Run{
			Id:             "123e4567-e89b-12d3-a456-426655440000",
			Name:           "run1",
			Status:         "Running",
			ServiceAccount: "default-editor",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
			FinishedAt:     &timestamp.Timestamp{},
			PipelineSpec: &api.PipelineSpec{
				WorkflowManifest: testWorkflow.ToStringForStore(),
				Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
			},
			ResourceReferences: []*api.ResourceReference{
				{
					Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "exp1", Relationship: api.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &api.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, *runDetail)
}

func TestListRun(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &api.Run{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   api.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
		FinishedAt:     &timestamp.Timestamp{},
		Status:         "Running",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Name: "exp1", Relationship: api.Relationship_OWNER,
			},
		},
	}
	listRunsResponse, err := server.ListRuns(nil, &api.ListRunsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse.Runs))
	assert.Equal(t, expectedRun, listRunsResponse.Runs[0])
}

func TestListRuns_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	_, err := server.ListRuns(ctx, &api.ListRunsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
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

func TestListRuns_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuns := []*api.Run{{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "run1",
		ServiceAccount: "pipeline-runner",
		StorageState:   api.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:    &timestamp.Timestamp{Seconds: 2},
		FinishedAt:     &timestamp.Timestamp{},
		Status:         "Running",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Name: "exp1", Relationship: api.Relationship_OWNER,
			},
		},
	}}
	expectedRunsEmpty := []*api.Run{}

	tests := []struct {
		name         string
		request      *api.ListRunsRequest
		wantError    bool
		errorMessage string
		expectedRuns []*api.Run
	}{
		{
			"Valid - filter by experiment",
			&api.ListRunsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_EXPERIMENT,
					Id:   "123e4567-e89b-12d3-a456-426655440000",
				},
			},
			false,
			"",
			expectedRuns,
		},
		{
			"Valid - filter by namespace",
			&api.ListRunsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   "ns1",
				},
			},
			false,
			"",
			expectedRuns,
		},
		{
			"Vailid - filter by namespace - no result",
			&api.ListRunsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   "no-such-ns",
				},
			},
			false,
			"",
			expectedRunsEmpty,
		},
		{
			"Invalid - no filter",
			&api.ListRunsRequest{},
			true,
			"ListRuns must filter by resource reference",
			nil,
		},
		{
			"Inalid - invalid filter type",
			&api.ListRunsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_UNKNOWN_RESOURCE_TYPE,
					Id:   "unknown",
				},
			},
			true,
			"Unrecognized resource reference type",
			nil,
		},
	}

	for _, tc := range tests {
		response, err := server.ListRuns(ctx, tc.request)

		if tc.wantError {
			if err == nil {
				t.Errorf("TestListRuns_Multiuser(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestListRuns_Multiusert(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TestListRuns_Multiuser(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedRuns, response.Runs, cmpopts.EquateEmpty(), protocmp.Transform(), cmpopts.IgnoreFields(api.Run{}, "ScheduledAt", "FinishedAt", "CreatedAt")) {
				t.Errorf("TestListRuns_Multiuser(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedRuns, response.Runs)
			}
		}
	}

}

func TestValidateCreateRunRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_WithPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_EmptyName(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The run name is empty")
}

func TestValidateCreateRunRequest_InvalidPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: referencesOfExperimentAndInvalidPipelineVersion,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed.")
}

func TestValidateCreateRunRequest_NoExperiment(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: nil,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_NilPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
}

func TestValidateCreateRunRequest_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidateCreateRunRequest_InvalidPipelineSpec(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			PipelineId:       resource.DefaultFakeUUID,
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidateCreateRunRequest_TooMuchParameters(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})

	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       params,
		},
	}

	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestReportRunMetrics_RunNotFound(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	_, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId: "1",
	})
	AssertUserError(t, err, codes.NotFound)
}

func TestReportRunMetrics_Succeed(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager, resourceManager, runDetails := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	response, err := runServer.ReportRunMetrics(ctx, &api.ReportRunMetricsRequest{
		RunId:   runDetails.UUID,
		Metrics: []*api.RunMetric{metric},
	})
	assert.Nil(t, err)
	expectedResponse := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   metric.Name,
				MetricNodeId: metric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
		},
	}
	assert.Equal(t, expectedResponse, response)

	run, err := runServer.GetRun(ctx, &api.GetRunRequest{
		RunId: runDetails.UUID,
	})
	assert.Nil(t, err)
	assert.Equal(t, []*api.RunMetric{metric}, run.GetRun().GetMetrics())
}

func TestReportRunMetrics_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager, resourceManager, runDetails := initWithOneTimeRun(t)
	defer clientManager.Close()
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager = resource.NewResourceManager(clientManager)
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	_, err := runServer.ReportRunMetrics(ctx, &api.ReportRunMetricsRequest{
		RunId:   runDetails.UUID,
		Metrics: []*api.RunMetric{metric},
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

func TestReportRunMetrics_PartialFailures(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, runDetail := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	validMetric := metric
	invalidNameMetric := &api.RunMetric{
		Name:   "$metric-1",
		NodeId: "node-1",
	}
	invalidNodeIDMetric := &api.RunMetric{
		Name: "metric-1",
	}
	response, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId:   runDetail.UUID,
		Metrics: []*api.RunMetric{validMetric, invalidNameMetric, invalidNodeIDMetric},
	})
	assert.Nil(t, err)
	expectedResponse := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   validMetric.Name,
				MetricNodeId: validMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
			{
				MetricName:   invalidNameMetric.Name,
				MetricNodeId: invalidNameMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
			{
				MetricName:   invalidNodeIDMetric.Name,
				MetricNodeId: invalidNodeIDMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
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

	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns"},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
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

	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
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

	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns"},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
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
	artifact := &api.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	response, err := runServer.ReadArtifact(ctx, artifact)
	assert.Nil(t, err)

	expectedResponse := &api.ReadArtifactResponse{
		Data: []byte(expectedContent),
	}
	assert.Equal(t, expectedResponse, response)
}

func TestReadArtifacts_Unauthorized(t *testing.T) {
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
	artifact := &api.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	_, err := runServer.ReadArtifact(ctx, artifact)
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

func TestReadArtifacts_Run_NotFound(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(clientManager)
	runServer := RunServer{resourceManager: manager, options: &RunServerOptions{CollectMetrics: false}}
	artifact := &api.ReadArtifactRequest{
		RunId:        "Wrong_RUN_UUID",
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	_, err := runServer.ReadArtifact(context.Background(), artifact)
	assert.NotNil(t, err)
	err = err.(*util.UserError)

	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}

func TestReadArtifacts_Resource_NotFound(t *testing.T) {
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
	artifactRequest := &api.ReadArtifactRequest{
		RunId:        run.UUID,
		NodeId:       "node-1",
		ArtifactName: "artifact-1",
	}
	_, err = runServer.ReadArtifact(context.Background(), artifactRequest)
	assert.NotNil(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
}
