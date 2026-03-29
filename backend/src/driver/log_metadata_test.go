// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"testing"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type logMetadataAPI struct {
	kfpapi.API
	task      *gc.PipelineTask
	update    *gc.UpdateTaskRequest
	getErr    error
	updateErr error
}

func (a *logMetadataAPI) GetTask(_ context.Context, _ *gc.GetTaskRequest) (*gc.PipelineTask, error) {
	return a.task, a.getErr
}

func (a *logMetadataAPI) UpdateTask(_ context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTask, error) {
	a.update = req
	return req.Task, a.updateErr
}

func TestRegisterDriverLogPreservesTaskMetadata(t *testing.T) {
	for _, metadata := range []*gc.PipelineTask_StatusMetadata{
		nil,
		{Message: "driver error", CustomProperties: map[string]*structpb.Value{
			"mlflow_run_id": structpb.NewStringValue("mlflow-task-run"),
		}},
	} {
		api := &logMetadataAPI{task: &gc.PipelineTask{
			TaskId: "task-id", RunId: "run-id", State: gc.PipelineTask_FAILED, StatusMetadata: metadata,
		}}
		original := proto.Clone(api.task)
		err := registerDriverLog(context.Background(), api, "run-id", "task-id", "s3://bucket/task/driver-logs", "{}")
		require.NoError(t, err)
		require.NotNil(t, api.update)
		assert.True(t, proto.Equal(original, api.task), "read task must not be mutated")
		assert.Equal(t, "task-id", api.update.TaskId)
		assert.Equal(t, "run-id", api.update.RunId)
		updated := api.update.Task.GetStatusMetadata()
		assert.Equal(t, "s3://bucket/task/driver-logs", updated.CustomProperties["driver_logs_uri"].GetStringValue())
		assert.Equal(t, "{}", updated.CustomProperties["store_session_info"].GetStringValue())
		assert.Equal(t, metadata.GetMessage(), updated.Message)
		if metadata != nil {
			assert.Equal(t, "mlflow-task-run", updated.CustomProperties["mlflow_run_id"].GetStringValue())
		}
		assert.Equal(t, gc.PipelineTask_RUNTIME_STATE_UNSPECIFIED, api.update.Task.State, "do not rewrite concurrent task state")
		assert.Nil(t, api.update.Task.Pods)
		assert.Nil(t, api.update.Task.Inputs)
	}
}

func TestRegisterDriverLogReportsAPIErrors(t *testing.T) {
	api := &logMetadataAPI{getErr: assert.AnError}
	assert.ErrorIs(t, registerDriverLog(context.Background(), api, "run", "task", "uri", "{}"), assert.AnError)
	assert.Nil(t, api.update)
	api = &logMetadataAPI{task: &gc.PipelineTask{}, updateErr: assert.AnError}
	assert.ErrorIs(t, registerDriverLog(context.Background(), api, "run", "task", "uri", "{}"), assert.AnError)
}
