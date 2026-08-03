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
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/types/known/structpb"
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

	// Build a skeleton task early so pre-CreateTask failures still produce a
	// FAILED task row and can propagate status to ancestors.
	taskName := opts.TaskName
	if taskName == "" {
		taskName = opts.Task.GetTaskInfo().GetName()
	}
	taskToCreate := &gc.PipelineTask{
		Name:        taskName,
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
	if opts.ParentTask.GetTaskId() != "" {
		taskToCreate.ParentTaskId = util.StringPointer(opts.ParentTask.GetTaskId())
	}
	if opts.IterationIndex >= 0 {
		taskToCreate.TypeAttributes = &gc.PipelineTask_TypeAttributes{
			IterationIndex: util.Int64Pointer(int64(opts.IterationIndex)),
		}
	}
	// Infer stable type before fallible resolution so retry cleanup CreateTask
	// shares logical identity with the eventual successful create (Type is part
	// of the logical key).
	applyInferredDAGTaskType(opts, taskToCreate)

	taskCreated := false
	defer func() {
		if err == nil {
			return
		}
		failedEndTime := timestamppb.New(time.Now())
		statusMetadata := taskToCreate.GetStatusMetadata()
		if statusMetadata == nil {
			statusMetadata = &gc.PipelineTask_StatusMetadata{}
		}
		statusMetadata.Message = err.Error()
		failedAttemptFields := &gc.PipelineTask{
			Pods:           taskToCreate.GetPods(),
			State:          gc.PipelineTask_FAILED,
			EndTime:        failedEndTime,
			StatusMetadata: statusMetadata,
		}
		taskToCreate.State = gc.PipelineTask_FAILED
		taskToCreate.EndTime = failedEndTime
		taskToCreate.StatusMetadata = statusMetadata
		if !taskCreated {
			createdTask, createErr := clientManager.KFPAPIClient().CreateTask(ctx, &gc.CreateTaskRequest{
				Task:  taskToCreate,
				RunId: taskToCreate.GetRunId(),
			})
			if createErr != nil {
				err = fmt.Errorf("%w: failed to create failed DAG task: %v", err, createErr)
				return
			}
			taskToCreate = createdTask
			taskCreated = true
		}
		// Always UpdateTask after CreateTask: on retry, CreateTask returns the
		// existing RUNNING row unchanged and must be finalized to FAILED.
		updatedTask, updateErr := updateTaskAttemptLocalFieldsAfterCreate(
			ctx,
			clientManager.KFPAPIClient(),
			taskToCreate,
			failedAttemptFields,
		)
		if updateErr != nil {
			err = fmt.Errorf("%w: failed to update task after DAG error: %v", err, updateErr)
			return
		}
		taskToCreate = updatedTask
		fullView := gc.GetRunRequest_FULL
		refreshedRun, getRunErr := clientManager.KFPAPIClient().GetRun(ctx, &gc.GetRunRequest{
			RunId: opts.Run.GetRunId(),
			View:  &fullView,
		})
		if getRunErr != nil {
			err = fmt.Errorf("%w: failed to refresh run after DAG failure: %v", err, getRunErr)
			return
		}
		if updateStatusErr := clientManager.KFPAPIClient().UpdateStatuses(
			ctx,
			refreshedRun,
			opts.ScopePath.GetPipelineSpecStruct(),
			taskToCreate,
		); updateStatusErr != nil {
			err = fmt.Errorf("%w: failed to propagate DAG failure status: %v", err, updateStatusErr)
		}
	}()

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
		willTrigger, conditionErr := expr.Condition(executorInput, condition)
		if conditionErr != nil {
			err = conditionErr
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	// Re-apply inferred type and attach resolved iteration count when present.
	// Type was already set before ResolveInputs for retry-safe logical identity;
	// this keeps DisplayName / IterationCount in sync with resolved inputs.
	applyInferredDAGTaskType(opts, taskToCreate)
	if iterationCount != nil {
		count := int64(*iterationCount)
		if taskToCreate.TypeAttributes == nil {
			taskToCreate.TypeAttributes = &gc.PipelineTask_TypeAttributes{}
		}
		taskToCreate.TypeAttributes.IterationCount = &count
		execution.IterationCount = util.IntPointer(int(count))
	}

	isTerminalWithoutChildren := false
	if terminalState, terminal := terminalDAGState(execution, iterationCount, opts.Component); terminal {
		taskToCreate.State = terminalState
		isTerminalWithoutChildren = true
	}
	if isTerminalWithoutChildren {
		taskToCreate.EndTime = timestamppb.Now()
	}
	taskToCreate, err = handleInputTaskParametersCreation(inputs.Parameters, taskToCreate)
	if err != nil {
		return execution, err
	}

	// Dispatch a plugin task for each loop DAG driver, but not the loop's individual iteration DAG drivers.
	var taskPluginInfo *plugins.TaskInfo
	dispatcher := opts.PluginDispatcher
	if dispatcher == nil {
		dispatcher = plugins.NoOpDispatcher{}
	}
	if opts.IterationIndex < 0 {
		applyParentPluginCustomProperties(dispatcher, opts.ParentTask)
		taskPluginInfo = &plugins.TaskInfo{Name: taskName}
		pluginStartResult, dispatchErr := dispatcher.OnTaskStart(ctx, taskPluginInfo)
		if dispatchErr != nil {
			glog.Errorf("Failed to dispatch task start: %v", dispatchErr)
		} else if pluginStartResult != nil {
			statusMetadata := taskToCreate.GetStatusMetadata()
			if statusMetadata == nil {
				statusMetadata = &gc.PipelineTask_StatusMetadata{}
			}
			statusMetadata.CustomProperties = stringMapToStructValues(pluginStartResult.CustomProperties)
			taskToCreate.StatusMetadata = statusMetadata
		}
	} else if parentProperties := opts.ParentTask.GetStatusMetadata().GetCustomProperties(); len(parentProperties) > 0 {
		statusMetadata := taskToCreate.GetStatusMetadata()
		if statusMetadata == nil {
			statusMetadata = &gc.PipelineTask_StatusMetadata{}
		}
		clonedProperties := make(map[string]*structpb.Value, len(parentProperties))
		for key, value := range parentProperties {
			clonedProperties[key] = value
		}
		statusMetadata.CustomProperties = clonedProperties
		taskToCreate.StatusMetadata = statusMetadata
	}
	defer func() {
		if taskPluginInfo != nil {
			state := gc.PipelineTask_SUCCEEDED
			if err != nil {
				state = gc.PipelineTask_FAILED
			}
			taskPluginInfo.UpdateTaskInfoWithMetadata(
				state,
				nil,
				parameterValuesToInterfaces(executorInput.GetInputs().GetParameterValues()),
			)
			dispatchErr := dispatcher.OnTaskEnd(ctx, taskPluginInfo)
			if dispatchErr != nil {
				glog.Errorf("failed to dispatch task end: %v", dispatchErr)
			}
		}
	}()
	if opts.Task.GetArtifactIterator() != nil {
		err = fmt.Errorf("ArtifactIterator is not implemented")
		return execution, err
	}
	isIterator := opts.Task.GetParameterIterator() != nil && opts.IterationIndex < 0
	if execution.WillTrigger() && isIterator {
		iterator := opts.Task.GetParameterIterator()
		report := func(err error) error {
			return fmt.Errorf("iterating on item input %q failed: %w", iterator.GetItemInput(), err)
		}
		itemsSpec := iterator.GetItems()
		var value *structpb.Value
		switch itemsSpec.GetKind().(type) {
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
			var ok bool
			value, ok = executorInput.GetInputs().GetParameterValues()[itemsSpec.GetInputParameter()]
			if !ok {
				err = report(fmt.Errorf("cannot find input parameter"))
				return execution, err
			}
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw:
			var unmarshalledRaw interface{}
			if err = json.Unmarshal([]byte(itemsSpec.GetRaw()), &unmarshalledRaw); err != nil {
				err = fmt.Errorf("error unmarshall raw string: %q", err)
				return execution, err
			}
			value, err = structpb.NewValue(unmarshalledRaw)
			if err != nil {
				err = fmt.Errorf("error converting unmarshalled raw string into protobuf Value type: %q", err)
				return execution, err
			}
			execution.ExecutorInput.Inputs.ParameterValues[iterator.GetItemInput()] = value
		default:
			err = fmt.Errorf("cannot find parameter iterator")
			return execution, err
		}
		items, itemsErr := getItems(value)
		if itemsErr != nil {
			err = report(itemsErr)
			return execution, err
		}
		count := len(items)
		if taskToCreate.TypeAttributes == nil {
			taskToCreate.TypeAttributes = &gc.PipelineTask_TypeAttributes{}
		}
		taskToCreate.TypeAttributes.IterationCount = util.Int64Pointer(int64(count))
		execution.IterationCount = util.IntPointer(count)
	}

	glog.Infof("Creating task: %+v", taskToCreate)
	attemptLocalFields := &gc.PipelineTask{
		Pods:             taskToCreate.GetPods(),
		Inputs:           taskToCreate.GetInputs(),
		CacheFingerprint: taskToCreate.GetCacheFingerprint(),
		State:            taskToCreate.GetState(),
		EndTime:          taskToCreate.GetEndTime(),
		StatusMetadata:   taskToCreate.GetStatusMetadata(),
	}
	createdTask, err := clientManager.KFPAPIClient().CreateTask(ctx, &gc.CreateTaskRequest{
		Task:  taskToCreate,
		RunId: taskToCreate.GetRunId(),
	})
	if err != nil {
		return execution, err
	}
	taskCreated = true
	taskToCreate = createdTask
	execution.TaskID = createdTask.TaskId
	createdTask, err = updateTaskAttemptLocalFieldsAfterCreate(ctx, clientManager.KFPAPIClient(), createdTask, attemptLocalFields)
	if err != nil {
		return execution, err
	}
	taskToCreate = createdTask
	glog.Infof("Created task: %+v", createdTask)

	err = handleInputTaskArtifactsCreation(ctx, opts, inputs.Artifacts, createdTask, clientManager.KFPAPIClient())
	if err != nil {
		return execution, err
	}

	// After retry reset, failed DAG parents lose propagated outputs while
	// successful children are preserved and will not re-propagate. Rebuild
	// those parent outputs before children are (re)driven.
	if !isTerminalWithoutChildren {
		if err := republishPreservedChildOutputsIfNeeded(ctx, opts, createdTask, clientManager); err != nil {
			return execution, err
		}
	}

	if isTerminalWithoutChildren {
		fullView := gc.GetRunRequest_FULL
		refreshedRun, getRunErr := clientManager.KFPAPIClient().GetRun(ctx, &gc.GetRunRequest{
			RunId: opts.Run.GetRunId(),
			View:  &fullView,
		})
		if getRunErr != nil {
			return execution, fmt.Errorf("failed to refresh run before propagating terminal DAG status: %w", getRunErr)
		}
		if updateStatusErr := clientManager.KFPAPIClient().UpdateStatuses(
			ctx,
			refreshedRun,
			opts.ScopePath.GetPipelineSpecStruct(),
			createdTask,
		); updateStatusErr != nil {
			return execution, fmt.Errorf("failed to propagate terminal DAG status: %w", updateStatusErr)
		}
	}

	return execution, nil
}

func republishPreservedChildOutputsIfNeeded(
	ctx context.Context,
	opts common.Options,
	parentTask *gc.PipelineTask,
	clientManager client_manager.ClientManagerInterface,
) error {
	if parentTask == nil || parentTask.GetTaskId() == "" {
		return nil
	}
	pipelineSpecStruct := opts.ScopePath.GetPipelineSpecStruct()
	if pipelineSpecStruct == nil {
		return nil
	}
	if err := component.RepublishPreservedChildOutputsToDAG(ctx, component.DAGOutputRepublishOptions{
		Run:          opts.Run,
		ParentTask:   parentTask,
		ParentScope:  opts.ScopePath,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager); err != nil {
		return fmt.Errorf("failed to republish preserved child outputs: %w", err)
	}
	return nil
}

// applyInferredDAGTaskType sets Type/DisplayName from pipeline structure that is
// available before ResolveInputs. Type participates in logical task identity, so
// retry cleanup must use the same type as a successful create.
func applyInferredDAGTaskType(opts common.Options, task *gc.PipelineTask) {
	if task == nil {
		return
	}
	condition := ""
	if opts.Task != nil && opts.Task.GetTriggerPolicy() != nil {
		condition = opts.Task.GetTriggerPolicy().GetCondition()
	}
	switch {
	case opts.Task != nil && opts.Task.GetParameterIterator() != nil && opts.IterationIndex < 0:
		task.Type = gc.PipelineTask_LOOP
		task.DisplayName = "Loop"
	case condition != "":
		task.Type = gc.PipelineTask_CONDITION_BRANCH
		task.DisplayName = "Condition Branch"
	case strings.HasPrefix(opts.TaskName, "condition") && !strings.HasPrefix(opts.TaskName, "condition-branch"):
		task.Type = gc.PipelineTask_CONDITION
		task.DisplayName = "Condition"
	default:
		task.Type = gc.PipelineTask_DAG
	}
}

func terminalDAGState(
	execution *Execution,
	iterationCount *int,
	component *pipelinespec.ComponentSpec,
) (gc.PipelineTask_TaskState, bool) {
	if !execution.WillTrigger() || iterationCount != nil && *iterationCount == 0 {
		return gc.PipelineTask_SKIPPED, true
	}
	if component.GetDag() != nil && len(component.GetDag().GetTasks()) == 0 {
		return gc.PipelineTask_SUCCEEDED, true
	}
	return gc.PipelineTask_RUNTIME_STATE_UNSPECIFIED, false
}
