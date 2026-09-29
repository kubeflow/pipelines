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
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type recordingSubjectAccessReviewClient struct {
	allowed   bool
	authorize func(authorizationv1.ResourceAttributes) bool
	requests  []authorizationv1.ResourceAttributes
}

func (c *recordingSubjectAccessReviewClient) Create(_ context.Context, review *authorizationv1.SubjectAccessReview, _ metav1.CreateOptions) (*authorizationv1.SubjectAccessReview, error) {
	allowed := c.allowed
	if review.Spec.ResourceAttributes != nil {
		attributes := *review.Spec.ResourceAttributes.DeepCopy()
		c.requests = append(c.requests, attributes)
		if c.authorize != nil {
			allowed = c.authorize(attributes)
		}
	}
	return &authorizationv1.SubjectAccessReview{
		Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed, Reason: "test authorization decision"},
	}, nil
}

func createRunServer(resourceManager *resource.ResourceManager) *RunServer {
	return &RunServer{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false},
		},
	}
}

func TestCanAccessReferencedPipeline_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// newManagerWithPipeline returns a resource manager holding a single pipeline
	// in the given namespace plus one pipeline version, along with their IDs. When
	// authorized is false the SubjectAccessReview client denies every request.
	newManagerWithPipeline := func(namespace string, authorized bool) (*resource.ResourceManager, string, string) {
		initEnvVars()
		clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
		t.Cleanup(func() { clientManager.Close() })
		if !authorized {
			clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
		}
		resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
		pipeline, err := resourceManager.CreatePipeline(&model.Pipeline{Name: "p1", Namespace: namespace})
		assert.Nil(t, err)
		pipelineVersion, err := resourceManager.CreatePipelineVersion(&model.PipelineVersion{
			Name:         "p1",
			PipelineId:   pipeline.UUID,
			PipelineSpec: model.LargeText(testIRPipeline),
		})
		assert.Nil(t, err)
		return resourceManager, pipeline.UUID, pipelineVersion.UUID
	}

	// A caller without access to another namespace's private pipeline is denied
	// when referencing it by version ID or by pipeline ID.
	rmPrivate, privatePipelineID, privateVersionID := newManagerWithPipeline("ns2", false)
	assert.Error(t, canAccessReferencedPipeline(ctx, rmPrivate, &model.PipelineSpec{PipelineVersionId: privateVersionID}, "ns1"))
	assert.Error(t, canAccessReferencedPipeline(ctx, rmPrivate, &model.PipelineSpec{PipelineId: privatePipelineID}, "ns1"))

	// Shared (empty-namespace) pipelines remain readable by any authenticated
	// user, even one the SubjectAccessReview would deny.
	rmShared, sharedPipelineID, sharedVersionID := newManagerWithPipeline("", false)
	assert.NoError(t, canAccessReferencedPipeline(ctx, rmShared, &model.PipelineSpec{PipelineVersionId: sharedVersionID}, "ns1"))
	assert.NoError(t, canAccessReferencedPipeline(ctx, rmShared, &model.PipelineSpec{PipelineId: sharedPipelineID}, "ns1"))

	// An inline manifest references no existing pipeline and needs no pipeline
	// authorization at this layer.
	assert.NoError(t, canAccessReferencedPipeline(ctx, rmShared, &model.PipelineSpec{}, "ns1"))

	// Pairing an accessible pipeline ID with another namespace's private version
	// ID must not authorize against the accessible pipeline: the referenced version
	// is resolved first and its owning pipeline governs authorization. Build one
	// manager holding both a shared pipeline and a private ns2 pipeline.
	initEnvVars()
	mixedClientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer mixedClientManager.Close()
	mixedClientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	mixedRM := resource.NewResourceManager(mixedClientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	sharedPipeline, err := mixedRM.CreatePipeline(&model.Pipeline{Name: "shared", Namespace: ""})
	assert.Nil(t, err)
	_, err = mixedRM.CreatePipelineVersion(&model.PipelineVersion{Name: "shared", PipelineId: sharedPipeline.UUID, PipelineSpec: model.LargeText(testIRPipeline)})
	assert.Nil(t, err)
	mixedClientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(NonDefaultFakeUUID, nil))
	mixedRM = resource.NewResourceManager(mixedClientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	privatePipeline, err := mixedRM.CreatePipeline(&model.Pipeline{Name: "private", Namespace: "ns2"})
	assert.Nil(t, err)
	privateVersion, err := mixedRM.CreatePipelineVersion(&model.PipelineVersion{Name: "private", PipelineId: privatePipeline.UUID, PipelineSpec: model.LargeText(testIRPipeline)})
	assert.Nil(t, err)
	err = canAccessReferencedPipeline(ctx, mixedRM, &model.PipelineSpec{PipelineId: sharedPipeline.UUID, PipelineVersionId: privateVersion.UUID}, "ns2")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied))
	err = canAccessReferencedPipeline(ctx, mixedRM, &model.PipelineSpec{
		PipelineId:   sharedPipeline.UUID,
		PipelineName: "pipelineversions/" + privateVersion.UUID,
	}, "ns2")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied))

	reviewClient := &recordingSubjectAccessReviewClient{allowed: true}
	mixedClientManager.SubjectAccessReviewClientFake = reviewClient
	mixedRM = resource.NewResourceManager(mixedClientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	err = canAccessReferencedPipeline(ctx, mixedRM, &model.PipelineSpec{PipelineId: sharedPipeline.UUID, PipelineVersionId: privateVersion.UUID}, "ns2")
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument))
	require.Len(t, reviewClient.requests, 1)
	assert.Equal(t, authorizationv1.ResourceAttributes{
		Namespace: "ns2",
		Verb:      common.RbacResourceVerbGet,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypePipelines,
		Name:      privatePipeline.Name,
	}, reviewClient.requests[0])

	// A caller authorized for the owning namespace is allowed.
	rmAuthorized, _, authorizedVersionID := newManagerWithPipeline("ns2", true)
	assert.NoError(t, canAccessReferencedPipeline(ctx, rmAuthorized, &model.PipelineSpec{PipelineVersionId: authorizedVersionID}, "ns2"))
	assert.Error(t, canAccessReferencedPipeline(ctx, rmAuthorized, &model.PipelineSpec{PipelineVersionId: authorizedVersionID}, "ns1"))
	viper.Set(common.MultiUserModeSharedReadAccess, "true")
	assert.NoError(t, canAccessReferencedPipeline(ctx, rmAuthorized, &model.PipelineSpec{PipelineVersionId: authorizedVersionID}, "ns1"))
	viper.Set(common.MultiUserModeSharedReadAccess, "false")

	// The pipeline/version can also be encoded in the full pipeline name; the same
	// authorization must apply on that parsing path so it cannot silently fail open.
	rmNamePrivate, _, namePrivateVersionID := newManagerWithPipeline("ns2", false)
	assert.Error(t, canAccessReferencedPipeline(ctx, rmNamePrivate, &model.PipelineSpec{PipelineName: "pipelineversions/" + namePrivateVersionID}, "ns1"))
	rmNameAuthorized, _, nameAuthorizedVersionID := newManagerWithPipeline("ns2", true)
	assert.NoError(t, canAccessReferencedPipeline(ctx, rmNameAuthorized, &model.PipelineSpec{PipelineName: "pipelineversions/" + nameAuthorizedVersionID}, "ns2"))
}

func TestCreateRun_MultiuserRecurringRunNamespaceBoundToOwner(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	jobID := "recurring-run-in-ns2"
	_, err := clients.JobStore().CreateJob(&model.Job{
		UUID:         jobID,
		DisplayName:  "recurring run",
		K8SName:      "recurring-run",
		Namespace:    "ns2",
		ExperimentId: experiment.UUID,
	})
	require.NoError(t, err)

	server := createRunServer(manager)
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err = server.createRun(ctx, &model.Run{
		DisplayName:    "controller-created-run",
		ExperimentId:   experiment.UUID,
		RecurringRunId: jobID,
	})
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied))
	assert.Contains(t, err.Error(), "A recurring run can only create runs in its own namespace")
}

func TestValidateRecurringRunNamespace_MultiuserLegacyOwner(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	jobID := "legacy-recurring-run"
	_, err := clients.JobStore().CreateJob(&model.Job{
		UUID:         jobID,
		DisplayName:  "legacy recurring run",
		K8SName:      "legacy-recurring-run",
		Namespace:    model.NoNamespace,
		ExperimentId: experiment.UUID,
	})
	require.NoError(t, err)

	err = validateRecurringRunNamespace(manager, &model.Run{
		Namespace:      "ns1",
		RecurringRunId: jobID,
	})
	require.NoError(t, err)
}

func TestValidateRecurringRunNamespace_MultiuserLegacyOwnerWithoutNamespace(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	jobID := "legacy-recurring-run-without-owner"
	_, err := clients.JobStore().CreateJob(&model.Job{
		UUID:        jobID,
		DisplayName: "legacy recurring run without owner",
		K8SName:     "legacy-recurring-run-without-owner",
		Namespace:   model.NoNamespace,
	})
	require.NoError(t, err)

	err = validateRecurringRunNamespace(manager, &model.Run{
		Namespace:      "ns1",
		RecurringRunId: jobID,
	})
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied))
	assert.Contains(t, err.Error(), "recreate it in a namespaced experiment")
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

func TestCanAccessRun_Authorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, oneTimeRun := initWithOneTimeRun(t)
	defer clients.Close()
	runServer := createRunServer(manager)

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	err := runServer.canAccessRun(ctx, oneTimeRun.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.Nil(t, err)
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
	run, err := server.CreateRun(context.Background(), &apiv2beta1.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	// RetryRun requires the workflow to be in Failed/Error state, so expect an error.
	_, err = server.RetryRun(context.Background(), &apiv2beta1.RetryRunRequest{RunId: run.RunId})
	assert.NotNil(t, err)
}

func TestArchiveRun(t *testing.T) {
	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()
	server := createRunServer(manager)
	_, err := server.ArchiveRun(context.Background(), &apiv2beta1.ArchiveRunRequest{RunId: run.UUID})
	assert.Nil(t, err)
}

func TestUnarchiveRun(t *testing.T) {
	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()
	server := createRunServer(manager)
	// Archive first, then unarchive.
	_, err := server.ArchiveRun(context.Background(), &apiv2beta1.ArchiveRunRequest{RunId: run.UUID})
	assert.Nil(t, err)
	_, err = server.UnarchiveRun(context.Background(), &apiv2beta1.UnarchiveRunRequest{RunId: run.UUID})
	assert.Nil(t, err)
}

func TestDeleteRun(t *testing.T) {
	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()
	server := createRunServer(manager)
	_, err := server.DeleteRun(context.Background(), &apiv2beta1.DeleteRunRequest{RunId: run.UUID})
	assert.Nil(t, err)
	// Verify the run is gone.
	_, err = manager.GetRun(run.UUID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTerminateRun(t *testing.T) {
	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()
	server := createRunServer(manager)
	_, err := server.TerminateRun(context.Background(), &apiv2beta1.TerminateRunRequest{RunId: run.UUID})
	assert.Nil(t, err)
}
