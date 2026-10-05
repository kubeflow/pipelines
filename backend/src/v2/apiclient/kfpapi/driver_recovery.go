// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kfpapi

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	runtimeapi "github.com/kubeflow/pipelines/backend/api/runtime/go_client"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type driverRecoveryContextKey struct{}

// WithDriverRecovery requests payloads only on runtime single-task operations.
func WithDriverRecovery(ctx context.Context) context.Context {
	return context.WithValue(ctx, driverRecoveryContextKey{}, true)
}
func includeDriverRecovery(ctx context.Context) bool {
	full, _ := ctx.Value(driverRecoveryContextKey{}).(bool)
	return full
}

// Recovery properties are an in-process compatibility representation for the
// driver, never part of the public PipelineTask sent over the wire.
func (k *clientAdapter) runtimeTaskRequest(ctx context.Context, runID string, task *api.PipelineTask, create bool) (*runtimeapi.WriteTaskRequest, error) {
	if task == nil {
		return nil, fmt.Errorf("runtime task is required")
	}
	copied := proto.Clone(task).(*api.PipelineTask)
	properties := copied.GetStatusMetadata().GetCustomProperties()
	decimal := func(key string) (*int64, error) {
		value, present := properties[key]
		if !present {
			return nil, nil
		}
		number, err := strconv.ParseInt(value.GetStringValue(), 10, 64)
		if err != nil || number < 0 || strconv.FormatInt(number, 10) != value.GetStringValue() {
			return nil, fmt.Errorf("invalid runtime property %s", key)
		}
		return &number, nil
	}
	generation, err := decimal(util.DriverRetryGenerationKey)
	if err != nil {
		return nil, err
	}
	attempt, err := decimal(util.DriverRetryAttemptKey)
	if err != nil {
		return nil, err
	}
	sourceAttempt, err := decimal(util.DriverRetrySourceAttemptKey)
	if err != nil {
		return nil, err
	}
	full := includeDriverRecovery(ctx)
	recovery := &runtimeapi.DriverRecovery{Generation: generation, Attempt: attempt}
	if full {
		if value, present := properties[util.DriverCheckpointKey]; present {
			recovery.CheckpointJson = proto.String(value.GetStringValue())
		}
		if value, present := properties[util.DriverCachedOutputsKey]; present {
			recovery.CachedOutputsJson = proto.String(value.GetStringValue())
		}
	}
	sourceID := properties[util.DriverRetrySourceTaskKey].GetStringValue()
	if sourceID != "" {
		// A child never supplies target ownership from a refreshed parent read.
		recovery = nil
	}
	strippedRecovery := false
	for key := range properties {
		if strings.HasPrefix(key, "_kfp_driver_") {
			delete(properties, key)
			strippedRecovery = true
		}
	}
	if strippedRecovery && proto.Size(copied.GetStatusMetadata()) == 0 {
		// CopyDriverRetryGeneration may allocate metadata solely for authority.
		// Preserve patch omission after moving that authority to the typed field.
		copied.StatusMetadata = nil
	}
	return &runtimeapi.WriteTaskRequest{
		RunId: runID, Task: copied,
		Authority: &runtimeapi.TaskAuthority{Generation: proto.Int64(k.generation), SourceTaskId: sourceID, SourceAttempt: sourceAttempt},
		Recovery:  recovery, Claim: create && attempt != nil && sourceID == "", UpdateRecovery: full && sourceID == "" && attempt != nil, IncludeRecovery: full,
	}, nil
}

func taskFromRuntime(response *runtimeapi.TaskResponse) *api.PipelineTask {
	if response == nil || response.Task == nil {
		return nil
	}
	task := response.Task
	recovery := response.Recovery
	if recovery == nil || recovery.Generation == nil {
		return task
	}
	if task.StatusMetadata == nil {
		task.StatusMetadata = &api.PipelineTask_StatusMetadata{}
	}
	if task.StatusMetadata.CustomProperties == nil {
		task.StatusMetadata.CustomProperties = make(map[string]*structpb.Value)
	}
	properties := task.StatusMetadata.CustomProperties
	properties[util.DriverRetryGenerationKey] = structpb.NewStringValue(strconv.FormatInt(*recovery.Generation, 10))
	if recovery.Attempt != nil {
		properties[util.DriverRetryAttemptKey] = structpb.NewStringValue(strconv.FormatInt(*recovery.Attempt, 10))
	}
	if recovery.CheckpointJson != nil {
		properties[util.DriverCheckpointKey] = structpb.NewStringValue(*recovery.CheckpointJson)
	}
	if recovery.CachedOutputsJson != nil {
		properties[util.DriverCachedOutputsKey] = structpb.NewStringValue(*recovery.CachedOutputsJson)
	}
	return task
}

func (k *clientAdapter) CreateTask(ctx context.Context, req *api.CreateTaskRequest) (*api.PipelineTask, error) {
	request, err := k.runtimeTaskRequest(ctx, req.GetRunId(), req.GetTask(), true)
	if err != nil {
		return nil, err
	}
	response, err := k.c.DriverTask.CreateTask(ctx, request)
	return taskFromRuntime(response), err
}
func (k *clientAdapter) UpdateTask(ctx context.Context, req *api.UpdateTaskRequest) (*api.PipelineTask, error) {
	request, err := k.runtimeTaskRequest(ctx, req.GetRunId(), req.GetTask(), false)
	if err != nil {
		return nil, err
	}
	if request.Task.TaskId != "" && request.Task.TaskId != req.TaskId {
		return nil, fmt.Errorf("task ID does not match update target")
	}
	request.Task.TaskId = req.TaskId
	response, err := k.c.DriverTask.UpdateTask(ctx, request)
	return taskFromRuntime(response), err
}
func (k *clientAdapter) GetTask(ctx context.Context, req *api.GetTaskRequest) (*api.PipelineTask, error) {
	response, err := k.c.DriverTask.GetTask(ctx, &runtimeapi.GetTaskRequest{RunId: req.GetRunId(), TaskId: req.GetTaskId(), IncludeRecovery: includeDriverRecovery(ctx)})
	return taskFromRuntime(response), err
}
func (k *clientAdapter) UpdateTasksBulk(ctx context.Context, req *api.UpdateTasksBulkRequest) (*api.UpdateTasksBulkResponse, error) {
	request := &runtimeapi.WriteTasksRequest{RunId: req.GetRunId()}
	ids := make([]string, 0, len(req.GetTasks()))
	for id := range req.GetTasks() {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		task, err := k.runtimeTaskRequest(ctx, req.RunId, req.Tasks[id], false)
		if err != nil {
			return nil, err
		}
		if task.Task.TaskId != "" && task.Task.TaskId != id {
			return nil, fmt.Errorf("task ID does not match bulk update key")
		}
		task.Task.TaskId = id
		task.IncludeRecovery = false
		request.Tasks = append(request.Tasks, task)
	}
	result, err := k.c.DriverTask.UpdateTasksBulk(ctx, request)
	if err != nil {
		return nil, err
	}
	response := &api.UpdateTasksBulkResponse{Tasks: make(map[string]*api.PipelineTask)}
	for _, updated := range result.GetTasks() {
		// Bulk callers need public state only; do not multiply recovery payloads.
		response.Tasks[updated.GetTask().GetTaskId()] = updated.GetTask()
	}
	return response, nil
}
func (k *clientAdapter) FinalizeStoppedDriver(ctx context.Context, runID string, generation int64, taskName, parentTaskID string, iterationIndex *int64) error {
	_, err := k.c.DriverTask.FinalizeStoppedDriver(ctx, &runtimeapi.FinalizeStoppedDriverRequest{RunId: runID, Generation: generation, TaskName: taskName, ParentTaskId: parentTaskID, IterationIndex: iterationIndex})
	return err
}
