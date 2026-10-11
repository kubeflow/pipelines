// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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

	runtimeapi "github.com/kubeflow/pipelines/backend/api/runtime/go_client"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDriverRecoveryNeverEntersPublicResponses(t *testing.T) {
	payload := strings.Repeat("x", 128*1024)
	stored := &model.Task{UUID: "task", DriverRetryGeneration: proto.Int64(0), DriverRetryAttempt: proto.Int64(1),
		DriverCheckpoint: &payload, DriverCachedOutputs: &payload,
		StatusMetadata: model.JSONData{"message": "failure details", "customProperties": map[string]interface{}{
			"visible": "user value", util.DriverCheckpointKey: payload, "_kfp_driver_future": payload,
		}},
	}
	task, err := toAPITask(stored, nil)
	require.NoError(t, err)
	require.Len(t, task.GetStatusMetadata().GetCustomProperties(), 1)
	require.Equal(t, "failure details", task.GetStatusMetadata().GetMessage())
	require.Less(t, proto.Size(task), 1024)
	require.Len(t, stored.StatusMetadata["customProperties"], 3, "projection must not mutate persistence")
	run := &model.Run{}
	for i := 0; i < 64; i++ {
		task := *stored
		task.UUID = fmt.Sprintf("task-%d", i)
		run.Tasks = append(run.Tasks, &task)
	}
	response := toApiRunWithPipelineSourcePreference(run, true)
	require.Len(t, response.GetTasks(), 64)
	require.Less(t, proto.Size(response), 64*1024)
}

type runtimeServerClients struct {
	*resource.FakeClientManager
	reviewer client.TokenReviewInterface
}

func (c *runtimeServerClients) TokenReviewClient() client.TokenReviewInterface { return c.reviewer }

type runtimeServerTokenReviewer struct{ runID, namespace string }

func (r runtimeServerTokenReviewer) Create(_ context.Context, request *authv1.TokenReview, _ metav1.CreateOptions) (*authv1.TokenReview, error) {
	return &authv1.TokenReview{Status: authv1.TokenReviewStatus{Authenticated: request.Spec.Token == "runtime-token", Audiences: []string{common.TokenAudienceForRun(r.runID)}, User: authv1.UserInfo{Username: "system:serviceaccount:" + r.namespace + ":pipeline-runner"}}}, nil
}

type directRuntimeClient struct {
	runtimeapi.DriverTaskServiceClient
	server  *DriverTaskServer
	context context.Context
}

func (c directRuntimeClient) UpdateTask(_ context.Context, request *runtimeapi.WriteTaskRequest, _ ...grpc.CallOption) (*runtimeapi.TaskResponse, error) {
	return c.server.UpdateTask(c.context, request)
}

func TestRuntimeTasksSeparateChildAuthorityFromParentRecovery(t *testing.T) {
	clients, _, run := initWithOneTimeRunV2(t)
	defer clients.Close()
	manager := resource.NewResourceManager(&runtimeServerClients{clients, runtimeServerTokenReviewer{run.UUID, run.Namespace}}, &resource.ResourceManagerOptions{})
	public := createRunServer(manager)
	runtime := NewDriverTaskServer(public)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer runtime-token"))
	task := &api.PipelineTask{RunId: run.UUID, Name: "parent", State: api.PipelineTask_RUNNING, Type: api.PipelineTask_RUNTIME,
		StatusMetadata: &api.PipelineTask_StatusMetadata{Message: "preserved message", CustomProperties: map[string]*structpb.Value{"visible": structpb.NewStringValue("user value")}},
	}
	checkpoint, cached := `{"executor_input":{}}`, `{"result":1}`
	parent, err := runtime.CreateTask(ctx, &runtimeapi.WriteTaskRequest{RunId: run.UUID, Task: task, Authority: &runtimeapi.TaskAuthority{Generation: proto.Int64(0)}, Recovery: &runtimeapi.DriverRecovery{Generation: proto.Int64(0), Attempt: proto.Int64(1), CheckpointJson: &checkpoint, CachedOutputsJson: &cached}, Claim: true, UpdateRecovery: true, IncludeRecovery: true})
	require.NoError(t, err)
	require.Equal(t, checkpoint, parent.Recovery.GetCheckpointJson())
	child, err := runtime.CreateTask(ctx, &runtimeapi.WriteTaskRequest{RunId: run.UUID,
		Task:      &api.PipelineTask{RunId: run.UUID, Name: "child", ParentTaskId: &parent.Task.TaskId, State: api.PipelineTask_RUNNING, Type: api.PipelineTask_RUNTIME},
		Authority: &runtimeapi.TaskAuthority{Generation: proto.Int64(0)},
	})
	require.NoError(t, err)
	require.Nil(t, child.Recovery.Attempt)
	// Untagged child status/output propagation must neither steal nor erase the
	// claimed parent's recovery. Nil metadata preserves the existing message.
	parent.Task.StatusMetadata = nil
	parent.Task.State = api.PipelineTask_SUCCEEDED
	adapter := kfpapi.NewWithRetryGeneration(&apiclient.Client{DriverTask: directRuntimeClient{server: runtime, context: ctx}}, 0)
	util.CopyDriverRetryGeneration(parent.Task, child.Task)
	propagated, err := adapter.UpdateTask(context.Background(), &api.UpdateTaskRequest{RunId: run.UUID, TaskId: parent.Task.TaskId, Task: parent.Task})
	require.NoError(t, err)
	require.Equal(t, "preserved message", propagated.GetStatusMetadata().GetMessage())
	updated, err := runtime.GetTask(ctx, &runtimeapi.GetTaskRequest{RunId: run.UUID, TaskId: parent.Task.TaskId, IncludeRecovery: true})
	require.NoError(t, err)
	require.Equal(t, api.PipelineTask_SUCCEEDED, updated.Task.State)
	require.Equal(t, "preserved message", updated.Task.GetStatusMetadata().GetMessage())
	require.EqualValues(t, 1, updated.Recovery.GetAttempt())
	require.Equal(t, checkpoint, updated.Recovery.GetCheckpointJson())
	require.Equal(t, cached, updated.Recovery.GetCachedOutputsJson())
	stored, err := clients.TaskStore().GetTask(parent.Task.TaskId)
	require.NoError(t, err)
	require.NotContains(t, stored.StatusMetadata["customProperties"], util.DriverRetryGenerationKey)
	require.NotContains(t, stored.StatusMetadata["customProperties"], util.DriverCheckpointKey)

	header := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-kfp-driver-recovery-view", "full"))
	visible, err := public.GetTask(header, &api.GetTaskRequest{RunId: run.UUID, TaskId: parent.Task.TaskId})
	require.NoError(t, err)
	require.Len(t, visible.GetStatusMetadata().GetCustomProperties(), 1)
	_, err = runtime.GetTask(header, &runtimeapi.GetTaskRequest{RunId: run.UUID, TaskId: parent.Task.TaskId, IncludeRecovery: true})
	require.Error(t, err, "a projection header is not runtime authentication")
	visible.StatusMetadata.CustomProperties[util.DriverRetryAttemptKey] = structpb.NewStringValue("2")
	_, err = public.UpdateTask(header, &api.UpdateTaskRequest{RunId: run.UUID, TaskId: visible.TaskId, Task: visible})
	require.ErrorContains(t, err, "reserved")
	delete(visible.StatusMetadata.CustomProperties, util.DriverRetryAttemptKey)
	_, err = public.UpdateTask(header, &api.UpdateTaskRequest{RunId: run.UUID, TaskId: visible.TaskId, Task: visible})
	require.Error(t, err, "public writes cannot erase a runtime claim")
}

func TestRuntimeTaskModelRejectsAmbiguousRecoveryRequests(t *testing.T) {
	base := &runtimeapi.WriteTaskRequest{RunId: "run", Task: &api.PipelineTask{TaskId: "task", RunId: "run"}, Authority: &runtimeapi.TaskAuthority{Generation: proto.Int64(1)}}
	for _, mutate := range []func(*runtimeapi.WriteTaskRequest){
		func(r *runtimeapi.WriteTaskRequest) { r.Authority = nil },
		func(r *runtimeapi.WriteTaskRequest) { r.Claim = true },
		func(r *runtimeapi.WriteTaskRequest) {
			r.Recovery = &runtimeapi.DriverRecovery{Generation: proto.Int64(2), Attempt: proto.Int64(0)}
		},
		func(r *runtimeapi.WriteTaskRequest) {
			r.UpdateRecovery = true
			r.Recovery = &runtimeapi.DriverRecovery{CheckpointJson: proto.String("invalid")}
		},
		func(r *runtimeapi.WriteTaskRequest) {
			r.Task.StatusMetadata = &api.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{util.DriverRetryGenerationKey: structpb.NewStringValue("1")}}
		},
	} {
		request := proto.Clone(base).(*runtimeapi.WriteTaskRequest)
		mutate(request)
		_, err := runtimeTaskModel(request, false)
		require.Error(t, err)
	}
}
