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

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDriverRecoveryMetadataProjections(t *testing.T) {
	payload := strings.Repeat("x", 128*1024)
	stored := &model.Task{UUID: "task", StatusMetadata: model.JSONData{
		"message": "failure details", "customProperties": map[string]interface{}{
			"visible":                     "user value",
			util.DriverRetryGenerationKey: "0", util.DriverRetryAttemptKey: "1",
			util.DriverCheckpointKey: payload, util.DriverCachedOutputsKey: payload,
			util.DriverRetrySourceTaskKey: "source", util.DriverRetrySourceAttemptKey: "0",
		},
	}}
	for _, tc := range []struct {
		view  string
		count int
	}{
		{"", 1}, {"unknown", 1}, {util.DriverRecoveryViewOwnership, 3}, {util.DriverRecoveryViewFull, 5},
	} {
		t.Run(tc.view, func(t *testing.T) {
			task, err := toAPITaskWithRecoveryView(stored, nil, tc.view)
			require.NoError(t, err)
			require.Len(t, task.GetStatusMetadata().GetCustomProperties(), tc.count)
			require.Equal(t, "failure details", task.GetStatusMetadata().GetMessage())
			require.Equal(t, "user value", task.GetStatusMetadata().GetCustomProperties()["visible"].GetStringValue())
			if tc.view != util.DriverRecoveryViewFull {
				require.Less(t, proto.Size(task), 1024)
			}
		})
	}
	require.Len(t, stored.StatusMetadata["customProperties"], 7, "projection must not mutate persistence")
	// A full run used to multiply two 128KiB payloads by every loop task.
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

func TestTaskRecoveryPayloadsStayPrivateAcrossReadAndBulkAPIs(t *testing.T) {
	clients, manager, run := initWithOneTimeRunV2(t)
	defer clients.Close()
	server := createRunServer(manager)
	full := metadata.NewIncomingContext(context.Background(), metadata.Pairs(util.DriverRecoveryViewHeader, util.DriverRecoveryViewFull))
	owner := metadata.NewIncomingContext(context.Background(), metadata.Pairs(util.DriverRecoveryViewHeader, util.DriverRecoveryViewOwnership))
	task := &api.PipelineTask{RunId: run.UUID, Name: "retry-task", State: api.PipelineTask_RUNNING, Type: api.PipelineTask_RUNTIME,
		StatusMetadata: &api.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{
			util.DriverRetryGenerationKey: structpb.NewStringValue("0"), util.DriverRetryAttemptKey: structpb.NewStringValue("0"),
			util.DriverCheckpointKey: structpb.NewStringValue("saved handoff"), util.DriverCachedOutputsKey: structpb.NewStringValue("saved outputs"),
			"visible": structpb.NewStringValue("user value"),
		}},
	}
	created, err := server.CreateTask(full, &api.CreateTaskRequest{RunId: run.UUID, Task: task})
	require.NoError(t, err)
	require.Len(t, created.GetStatusMetadata().GetCustomProperties(), 5)
	get := &api.GetTaskRequest{RunId: run.UUID, TaskId: created.TaskId}
	public, err := server.GetTask(context.Background(), get)
	require.NoError(t, err)
	require.Len(t, public.GetStatusMetadata().GetCustomProperties(), 1)
	owned, err := server.GetTask(owner, get)
	require.NoError(t, err)
	require.Len(t, owned.GetStatusMetadata().GetCustomProperties(), 3)
	// Launcher updates carry ownership without echoing durable payloads.
	owned.State = api.PipelineTask_SUCCEEDED
	updated, err := server.UpdateTask(owner, &api.UpdateTaskRequest{RunId: run.UUID, TaskId: created.TaskId, Task: owned})
	require.NoError(t, err)
	require.Len(t, updated.GetStatusMetadata().GetCustomProperties(), 3)
	restored, err := server.GetTask(full, get)
	require.NoError(t, err)
	require.Equal(t, "saved handoff", restored.GetStatusMetadata().GetCustomProperties()[util.DriverCheckpointKey].GetStringValue())
	require.Equal(t, "saved outputs", restored.GetStatusMetadata().GetCustomProperties()[util.DriverCachedOutputsKey].GetStringValue())
	listed, err := server.ListTasks(full, &api.ListTasksRequest{RunId: run.UUID})
	require.NoError(t, err)
	require.Len(t, listed.Tasks, 1)
	require.Len(t, listed.Tasks[0].GetStatusMetadata().GetCustomProperties(), 1)
	bulk, err := server.UpdateTasksBulk(full, &api.UpdateTasksBulkRequest{RunId: run.UUID, Tasks: map[string]*api.PipelineTask{created.TaskId: owned}})
	require.NoError(t, err)
	require.Len(t, bulk.Tasks[created.TaskId].GetStatusMetadata().GetCustomProperties(), 1)
}
