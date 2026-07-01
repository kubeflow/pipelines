// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func DAG(ctx context.Context, opts common.Options, clientManager client_manager.ClientManagerInterface) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAG(%s) failed: %w", opts.Info(), err)
		}
	}()

	b, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	glog.V(4).Info("DAG opts: ", string(b))
	if err = validateDAG(opts); err != nil {
		return nil, err
	}

	if clientManager == nil {
		return nil, fmt.Errorf("ClientManager is nil")
	}

	expr, err := expression.New()
	if err != nil {
		return nil, err
	}

	inputs, iterationCount, err := resolver.ResolveInputs(ctx, opts)
	if err != nil {
		return nil, err
	}

	executorInput, err := pipelineTaskInputsToExecutorInputs(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert inputs to executor inputs: %w", err)
	}

	// ExecutorInput is not required for DAG/root execution, but keeping the
	// resolved view on Execution remains useful for tests and debugging.
	glog.Infof("executorInput value: %+v", executorInput)
	execution = &Execution{ExecutorInput: executorInput}

	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	taskToCreate := &gc.PipelineTask{
		Name:        opts.TaskName,
		DisplayName: opts.Task.GetTaskInfo().GetName(),
		RunId:       opts.Run.GetRunId(),
		// Default to DAG
		Type:       gc.PipelineTask_DAG,
		State:      gc.PipelineTask_RUNNING,
		ScopePath:  opts.ScopePath.DotNotation(),
		CreateTime: timestamppb.Now(),
		Pods: []*gc.PipelineTask_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: gc.PipelineTask_DRIVER,
			},
		},
	}

	// Determine type of DAG task.
	// In the future the KFP Sdk should add a Task Type enum to the task Info proto
	// to assist with inferring type. For now, we infer the type based on attribute
	// heuristics.
	switch {
	case iterationCount != nil:
		count := int64(*iterationCount)
		taskToCreate.TypeAttributes = &gc.PipelineTask_TypeAttributes{IterationCount: &count}
		taskToCreate.Type = gc.PipelineTask_LOOP
		taskToCreate.DisplayName = "Loop"
		execution.IterationCount = util.IntPointer(int(count))
	case condition != "":
		taskToCreate.Type = gc.PipelineTask_CONDITION_BRANCH
		taskToCreate.DisplayName = "Condition Branch"
	case strings.HasPrefix(opts.TaskName, "condition") && !strings.HasPrefix(opts.TaskName, "condition-branch"):
		taskToCreate.Type = gc.PipelineTask_CONDITION
		taskToCreate.DisplayName = "Condition"
	default:
		taskToCreate.Type = gc.PipelineTask_DAG
	}

	if opts.IterationIndex >= 0 {
		if taskToCreate.TypeAttributes == nil {
			taskToCreate.TypeAttributes = &gc.PipelineTask_TypeAttributes{}
		}
		taskToCreate.TypeAttributes.IterationIndex = util.Int64Pointer(int64(opts.IterationIndex))
	}

	if opts.ParentTask.GetTaskId() != "" {
		taskToCreate.ParentTaskId = util.StringPointer(opts.ParentTask.GetTaskId())
	}
	if !execution.WillTrigger() {
		taskToCreate.State = gc.PipelineTask_SKIPPED
	}
	taskToCreate, err = handleInputTaskParametersCreation(inputs.Parameters, taskToCreate)
	if err != nil {
		return execution, err
	}

	taskCreated := false
	defer func() {
		if err == nil || !taskCreated {
			return
		}
		taskToCreate.State = gc.PipelineTask_FAILED
		taskToCreate.EndTime = timestamppb.New(time.Now())
		taskToCreate.StatusMetadata = &gc.PipelineTask_StatusMetadata{
			Message: err.Error(),
		}
		_, updateErr := clientManager.KFPAPIClient().UpdateTask(ctx, &gc.UpdateTaskRequest{
			TaskId: taskToCreate.GetTaskId(),
			Task:   taskToCreate,
			RunId:  taskToCreate.GetRunId(),
		})
		if updateErr != nil {
			err = fmt.Errorf("%w: failed to update task after DAG error: %v", err, updateErr)
		}
	}()
	glog.Infof("Creating task: %+v", taskToCreate)
	createdTask, err := clientManager.KFPAPIClient().CreateTask(ctx, &gc.CreateTaskRequest{
		Task:  taskToCreate,
		RunId: taskToCreate.GetRunId(),
	})
	if err != nil {
		return execution, err
	}
	glog.Infof("Created task: %+v", createdTask)
	taskCreated = true
	taskToCreate = createdTask
	execution.TaskID = createdTask.TaskId

	err = handleInputTaskArtifactsCreation(ctx, opts, inputs.Artifacts, createdTask, clientManager.KFPAPIClient())
	if err != nil {
		return execution, err
	}

	return execution, nil
}
