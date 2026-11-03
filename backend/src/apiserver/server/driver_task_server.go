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

package server

import (
	"context"
	"encoding/json"

	runtimeapi "github.com/kubeflow/pipelines/backend/api/runtime/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/types/known/emptypb"
)

// DriverTaskServer is registered only on gRPC, without an HTTP gateway. Its
// typed recovery state never enters PipelineTask.StatusMetadata.
type DriverTaskServer struct {
	runtimeapi.UnimplementedDriverTaskServiceServer
	runs *RunServer
}

func NewDriverTaskServer(runs *RunServer) *DriverTaskServer {
	return &DriverTaskServer{runs: runs}
}

func (s *DriverTaskServer) authorize(ctx context.Context, runID, verb string) error {
	return s.runs.resourceManager.AuthorizeRuntimeTask(ctx, runID, verb)
}

func (s *DriverTaskServer) CreateTask(ctx context.Context, req *runtimeapi.WriteTaskRequest) (*runtimeapi.TaskResponse, error) {
	return s.writeTask(ctx, req, true)
}

func (s *DriverTaskServer) UpdateTask(ctx context.Context, req *runtimeapi.WriteTaskRequest) (*runtimeapi.TaskResponse, error) {
	return s.writeTask(ctx, req, false)
}

func runtimeTaskModel(req *runtimeapi.WriteTaskRequest, create bool) (*model.Task, error) {
	if req == nil || req.Task == nil || req.Authority == nil || req.Authority.Generation == nil || req.Authority.GetGeneration() < 0 {
		return nil, util.NewInvalidInputError("Runtime task and immutable run generation are required")
	}
	if err := validateTaskRunIDInRequest(req.Task.GetRunId(), req.RunId); err != nil {
		return nil, err
	}
	if !create && req.Task.GetTaskId() == "" {
		return nil, util.NewInvalidInputError("Task ID is required")
	}
	if req.Claim && (!create || req.Recovery == nil || req.Recovery.Attempt == nil || req.Authority.SourceTaskId != "") {
		return nil, util.NewInvalidInputError("Only a driver's CreateTask may claim an attempt")
	}
	if req.Authority.SourceAttempt != nil && (req.Authority.SourceTaskId == "" || *req.Authority.SourceAttempt < 0) {
		return nil, util.NewInvalidInputError("Source attempt requires a source task")
	}
	task, err := toModelTask(req.Task)
	if err != nil {
		return nil, err
	}
	task.RunUUID = req.RunId
	task.DriverWriteAuthority = &model.DriverTaskAuthority{Generation: req.Authority.GetGeneration(), SourceTaskID: req.Authority.SourceTaskId, SourceAttempt: req.Authority.SourceAttempt}
	task.DriverClaim = req.Claim
	task.DriverRecoveryUpdate = req.UpdateRecovery
	if recovery := req.Recovery; recovery != nil {
		if recovery.Generation == nil && recovery.Attempt != nil || recovery.Generation != nil && *recovery.Generation != req.Authority.GetGeneration() || recovery.Attempt != nil && *recovery.Attempt < 0 {
			return nil, util.NewInvalidInputError("Recovery owner must match the caller's run generation")
		}
		task.DriverRetryGeneration, task.DriverRetryAttempt = recovery.Generation, recovery.Attempt
		if req.Authority.SourceTaskId == "" {
			task.DriverWriteAuthority.SourceAttempt = recovery.Attempt
		}
		for _, payload := range []*string{recovery.CheckpointJson, recovery.CachedOutputsJson} {
			if payload != nil && (!req.UpdateRecovery || !json.Valid([]byte(*payload))) {
				return nil, util.NewInvalidInputError("Recovery payloads require an explicit recovery write and valid JSON")
			}
		}
		task.DriverCheckpoint, task.DriverCachedOutputs = recovery.CheckpointJson, recovery.CachedOutputsJson
	}
	return task, nil
}

func (s *DriverTaskServer) writeTask(ctx context.Context, req *runtimeapi.WriteTaskRequest, create bool) (*runtimeapi.TaskResponse, error) {
	if err := s.authorize(ctx, req.GetRunId(), common.RbacResourceVerbUpdate); err != nil {
		return nil, err
	}
	return s.writeAuthorizedTask(req, create)
}

func (s *DriverTaskServer) writeAuthorizedTask(req *runtimeapi.WriteTaskRequest, create bool) (*runtimeapi.TaskResponse, error) {
	task, err := runtimeTaskModel(req, create)
	if err != nil {
		return nil, err
	}
	if !create {
		stored, err := s.runs.resourceManager.GetTask(task.UUID)
		if err != nil {
			return nil, err
		}
		if stored.RunUUID != task.RunUUID {
			return nil, util.NewInvalidInputError("Task does not belong to the requested run")
		}
	}
	if err := s.runs.validateParentTaskOwnership(task.ParentTaskUUID, task.RunUUID); err != nil {
		return nil, err
	}
	var stored *model.Task
	if create {
		stored, err = s.runs.resourceManager.CreateTask(task)
	} else {
		stored, err = s.runs.resourceManager.UpdateTask(task)
	}
	if err != nil {
		return nil, err
	}
	return s.taskResponse(stored, req.IncludeRecovery)
}

func (s *DriverTaskServer) UpdateTasksBulk(ctx context.Context, req *runtimeapi.WriteTasksRequest) (*runtimeapi.TasksResponse, error) {
	if err := s.authorize(ctx, req.GetRunId(), common.RbacResourceVerbUpdate); err != nil {
		return nil, err
	}
	// Check the entire batch's identity before applying any writes.
	for _, task := range req.GetTasks() {
		if task.GetRunId() != req.RunId || task.GetClaim() || task.GetIncludeRecovery() {
			return nil, util.NewInvalidInputError("Bulk task writes must belong to one run and cannot claim or return recovery payloads")
		}
		converted, err := runtimeTaskModel(task, false)
		if err != nil {
			return nil, err
		}
		stored, err := s.runs.resourceManager.GetTask(converted.UUID)
		if err != nil {
			return nil, err
		}
		if stored.RunUUID != req.RunId {
			return nil, util.NewInvalidInputError("Task does not belong to the requested run")
		}
	}
	response := &runtimeapi.TasksResponse{}
	for _, task := range req.GetTasks() {
		updated, err := s.writeAuthorizedTask(task, false)
		if err != nil {
			return nil, err
		}
		updated.Recovery = nil
		response.Tasks = append(response.Tasks, updated)
	}
	return response, nil
}

func (s *DriverTaskServer) GetTask(ctx context.Context, req *runtimeapi.GetTaskRequest) (*runtimeapi.TaskResponse, error) {
	if err := s.authorize(ctx, req.GetRunId(), common.RbacResourceVerbGet); err != nil {
		return nil, err
	}
	task, err := s.runs.resourceManager.GetTask(req.GetTaskId())
	if err != nil {
		return nil, err
	}
	if task.RunUUID != req.RunId {
		return nil, util.NewInvalidInputError("Task does not belong to the requested run")
	}
	return s.taskResponse(task, req.IncludeRecovery)
}

func (s *DriverTaskServer) taskResponse(task *model.Task, full bool) (*runtimeapi.TaskResponse, error) {
	children, err := s.runs.resourceManager.GetTaskChildren(task.UUID)
	if err != nil {
		return nil, err
	}
	apiTask, err := toAPITask(task, filterTaskChildrenByRun(children, task.RunUUID))
	if err != nil {
		return nil, err
	}
	recovery := &runtimeapi.DriverRecovery{Generation: task.DriverRetryGeneration, Attempt: task.DriverRetryAttempt}
	if full {
		recovery.CheckpointJson, recovery.CachedOutputsJson = task.DriverCheckpoint, task.DriverCachedOutputs
	}
	return &runtimeapi.TaskResponse{Task: apiTask, Recovery: recovery}, nil
}

func (s *DriverTaskServer) FinalizeStoppedDriver(ctx context.Context, req *runtimeapi.FinalizeStoppedDriverRequest) (*emptypb.Empty, error) {
	if err := s.authorize(ctx, req.GetRunId(), common.RbacResourceVerbUpdate); err != nil {
		return nil, err
	}
	if req.Generation < 0 || req.TaskName == "" || req.IterationIndex != nil && *req.IterationIndex < 0 {
		return nil, util.NewInvalidInputError("Stopped driver requires a valid task identity and generation")
	}
	if err := s.runs.resourceManager.FinalizeStoppedDriver(req.RunId, req.Generation, req.TaskName, req.ParentTaskId, req.IterationIndex); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
