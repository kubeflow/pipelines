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

	// Determine this Task's Type
	inputs, iterationCount, err := resolver.ResolveInputs(ctx, opts)
	if err != nil {
		return nil, err
	}

	executorInput, err := pipelineTaskInputsToExecutorInputs(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert inputs to executor inputs: %w", err)
	}

	// TODO(HumairAK) this doesn't seem used in dag case (or root)
	// consider removing it. ExecutorInput is only required by Runtimes.
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

	taskToCreate := &gc.PipelineTaskDetail{
		Name:        opts.TaskName,
		DisplayName: opts.Task.GetTaskInfo().GetName(),
		RunId:       opts.Run.GetRunId(),
		// Default to DAG
		Type:       gc.PipelineTaskDetail_DAG,
		State:      gc.PipelineTaskDetail_RUNNING,
		ScopePath:  opts.ScopePath.DotNotation(),
		CreateTime: timestamppb.Now(),
		Pods: []*gc.PipelineTaskDetail_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: gc.PipelineTaskDetail_DRIVER,
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
		taskToCreate.TypeAttributes = &gc.PipelineTaskDetail_TypeAttributes{IterationCount: &count}
		taskToCreate.Type = gc.PipelineTaskDetail_LOOP
		taskToCreate.DisplayName = "Loop"
		execution.IterationCount = util.IntPointer(int(count))
	case condition != "":
		taskToCreate.Type = gc.PipelineTaskDetail_CONDITION_BRANCH
		taskToCreate.DisplayName = "Condition Branch"
	case strings.HasPrefix(opts.TaskName, "condition") && !strings.HasPrefix(opts.TaskName, "condition-branch"):
		taskToCreate.Type = gc.PipelineTaskDetail_CONDITION
		taskToCreate.DisplayName = "Condition"
	default:
		taskToCreate.Type = gc.PipelineTaskDetail_DAG
	}

	if opts.ParentTask.GetTaskId() != "" {
		taskToCreate.ParentTaskId = util.StringPointer(opts.ParentTask.GetTaskId())
	}
	taskToCreate, err = handleInputTaskParametersCreation(inputs.Parameters, taskToCreate)
	if err != nil {
		return execution, err
	}
	glog.Infof("Creating task: %+v", taskToCreate)
	createdTask, err := clientManager.KFPAPIClient().CreateTask(ctx, &gc.CreateTaskRequest{Task: taskToCreate})
	if err != nil {
		return execution, err
	}
	glog.Infof("Created task: %+v", createdTask)
	execution.TaskID = createdTask.TaskId

	err = handleInputTaskArtifactsCreation(ctx, opts, inputs.Artifacts, createdTask, clientManager.KFPAPIClient())
	if err != nil {
		return execution, err
	}

	return execution, nil
}
