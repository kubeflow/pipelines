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

// Package driver resolves pipeline tasks and prepares them for execution.
package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	driverCheckpointKey    = "_kfp_driver_checkpoint"
	driverCachedOutputsKey = "_kfp_driver_cached_outputs"
)

// driverCheckpoint preserves the handoff if the pod dies after driver work but
// before Argo collects its output files. Protobuf payloads use protobuf JSON.
type driverCheckpoint struct {
	Version        int             `json:"version"`
	TaskID         string          `json:"taskId"`
	ExecutorInput  json.RawMessage `json:"executorInput,omitempty"`
	IterationCount *int            `json:"iterationCount,omitempty"`
	Condition      *bool           `json:"condition,omitempty"`
	Cached         *bool           `json:"cached,omitempty"`
	PodSpecPatch   string          `json:"podSpecPatch,omitempty"`
}

type driverOperation func(context.Context, common.Options, client_manager.ClientManagerInterface) (*Execution, error)

func recoverDriver(ctx context.Context, opts common.Options, manager client_manager.ClientManagerInterface, operation driverOperation) (*Execution, error) {
	if manager == nil || manager.KFPAPIClient() == nil {
		return nil, fmt.Errorf("API client is required for driver recovery")
	}
	if opts.Run.GetRunId() == "" || opts.ParentTask.GetTaskId() == "" || opts.TaskName == "" || opts.ScopePath.DotNotation() == "" {
		return nil, fmt.Errorf("run, parent task, task name and scope are required for driver recovery")
	}
	if opts.DriverRetryGeneration < 0 || opts.DriverRetryMaxCount < 0 || opts.DriverRetryAttempt < 0 || opts.DriverRetryAttempt > opts.DriverRetryMaxCount {
		return nil, fmt.Errorf("invalid driver retry attempt, limit or generation; recompile the pipeline with valid retry settings")
	}
	if err := plugins.ValidateDriverRetry(opts.PluginDispatcher); err != nil {
		return nil, err
	}
	task := &api.PipelineTask{
		RunId: opts.Run.GetRunId(), Name: opts.TaskName,
		DisplayName:  opts.Task.GetTaskInfo().GetName(),
		ParentTaskId: util.StringPointer(opts.ParentTask.GetTaskId()),
		ScopePath:    opts.ScopePath.DotNotation(),
		Type:         api.PipelineTask_RUNTIME, State: api.PipelineTask_RUNNING,
		CreateTime: timestamppb.Now(),
		Pods:       []*api.PipelineTask_TaskPod{{Name: opts.PodName, Uid: opts.PodUID, Type: api.PipelineTask_DRIVER}},
	}
	if opts.IterationIndex >= 0 {
		task.TypeAttributes = &api.PipelineTask_TypeAttributes{IterationIndex: util.Int64Pointer(int64(opts.IterationIndex))}
	}
	if opts.DriverType == "DAG" {
		task.Type = api.PipelineTask_DAG
		applyInferredDAGTaskType(opts, task)
	}
	generation := strconv.FormatInt(opts.DriverRetryGeneration, 10)
	setDriverRetryOwner(task, opts)
	stored, err := manager.KFPAPIClient().CreateTask(ctx, &api.CreateTaskRequest{RunId: task.RunId, Task: task})
	if err != nil {
		return nil, fmt.Errorf("failed to recover logical driver task: %w", err)
	}
	if stored == nil || stored.GetTaskId() == "" {
		return nil, fmt.Errorf("driver recovery returned an empty task identity")
	}
	stored = proto.Clone(stored).(*api.PipelineTask)
	metadata := driverRecoveryMetadata(stored)
	storedGeneration := metadata.CustomProperties[util.DriverRetryGenerationKey].GetStringValue()
	if storedGeneration != "" && storedGeneration != generation {
		previousGeneration, parseErr := strconv.ParseInt(storedGeneration, 10, 64)
		preserved := stored.GetState() == api.PipelineTask_SUCCEEDED || stored.GetState() == api.PipelineTask_CACHED || stored.GetState() == api.PipelineTask_SKIPPED
		// RetryRun preserves completed native tasks even when their driver pod
		// failed to publish the handoff. CreateTask has already fenced this
		// request against the current run generation, so a newer manual retry
		// can finish acknowledging that preserved result.
		if parseErr != nil || previousGeneration < 0 || strconv.FormatInt(previousGeneration, 10) != storedGeneration || previousGeneration >= opts.DriverRetryGeneration || !preserved {
			return nil, fmt.Errorf("driver task belongs to retry generation %s, not %s; retry the run through the API", storedGeneration, generation)
		}
	}
	setDriverRetryOwner(stored, opts)
	if checkpoint := metadata.CustomProperties[driverCheckpointKey].GetStringValue(); checkpoint != "" {
		execution, err := restoreDriverCheckpoint(checkpoint, stored.GetTaskId())
		if err != nil {
			return nil, err
		}
		stored.Pods = appendDriverPod(stored.GetPods(), task.Pods[0])
		_, err = updateDriverTask(ctx, manager.KFPAPIClient(), stored)
		return execution, err
	}
	// Keep the native task nonterminal while Argo owns retry scheduling. The
	// guarded terminal workflow report closes unfinished tasks after any form
	// of exhaustion, including policy rejection, deadlines and pod deletion.
	stored.State = api.PipelineTask_RUNNING
	stored.EndTime = nil
	metadata.Message = ""
	stored.Pods = appendDriverPod(stored.GetPods(), task.Pods[0])
	stored, err = updateDriverTask(ctx, manager.KFPAPIClient(), stored)
	if err != nil {
		return nil, err
	}
	opts.DriverRetryTask = stored
	execution, driveErr := operation(ctx, opts, manager)
	latest, readErr := manager.KFPAPIClient().GetTask(ctx, &api.GetTaskRequest{RunId: stored.RunId, TaskId: stored.TaskId})
	if readErr != nil {
		return execution, errors.Join(driveErr, fmt.Errorf("failed to read driver recovery state: %w", readErr))
	}
	if latest == nil || latest.GetTaskId() != stored.TaskId {
		return execution, errors.Join(driveErr, fmt.Errorf("driver recovery returned an invalid task identity"))
	}
	latest = proto.Clone(latest).(*api.PipelineTask)
	latestMetadata := driverRecoveryMetadata(latest)
	setDriverRetryOwner(latest, opts)
	if driveErr != nil {
		latest.State = api.PipelineTask_RUNNING
		latest.EndTime = nil
		latestMetadata.Message = driveErr.Error()
		_, updateErr := updateDriverTask(ctx, manager.KFPAPIClient(), latest)
		return execution, errors.Join(driveErr, updateErr)
	}
	if execution == nil || execution.TaskID != stored.TaskId {
		return nil, fmt.Errorf("driver handoff does not match its recovered task identity")
	}
	checkpoint, err := marshalDriverCheckpoint(execution)
	if err != nil {
		return execution, err
	}
	latestMetadata.Message = ""
	latestMetadata.CustomProperties[driverCheckpointKey] = structpb.NewStringValue(checkpoint)
	_, err = updateDriverTask(ctx, manager.KFPAPIClient(), latest)
	if err != nil {
		return execution, fmt.Errorf("failed to persist driver handoff: %w", err)
	}
	return execution, nil
}

func driverRecoveryMetadata(task *api.PipelineTask) *api.PipelineTask_StatusMetadata {
	if task.StatusMetadata == nil {
		task.StatusMetadata = &api.PipelineTask_StatusMetadata{}
	}
	if task.StatusMetadata.CustomProperties == nil {
		task.StatusMetadata.CustomProperties = make(map[string]*structpb.Value)
	}
	return task.StatusMetadata
}

func setDriverRetryOwner(task *api.PipelineTask, opts common.Options) {
	properties := driverRecoveryMetadata(task).CustomProperties
	properties[util.DriverRetryGenerationKey] = structpb.NewStringValue(strconv.FormatInt(opts.DriverRetryGeneration, 10))
	properties[util.DriverRetryAttemptKey] = structpb.NewStringValue(strconv.Itoa(opts.DriverRetryAttempt))
	delete(properties, util.DriverRetrySourceTaskKey)
	delete(properties, util.DriverRetrySourceAttemptKey)
}

func isDriverRecoveryProperty(key string) bool {
	return key == util.DriverRetryGenerationKey || key == util.DriverRetryAttemptKey ||
		key == util.DriverRetrySourceTaskKey || key == util.DriverRetrySourceAttemptKey ||
		key == driverCheckpointKey || key == driverCachedOutputsKey
}

func driverRecoveryProperties(task *api.PipelineTask) map[string]*structpb.Value {
	properties := make(map[string]*structpb.Value)
	for _, key := range []string{util.DriverRetryGenerationKey, util.DriverRetryAttemptKey, driverCheckpointKey, driverCachedOutputsKey} {
		if value, ok := task.GetStatusMetadata().GetCustomProperties()[key]; ok {
			properties[key] = proto.Clone(value).(*structpb.Value)
		}
	}
	return properties
}

func updateDriverTask(ctx context.Context, client kfpapi.API, task *api.PipelineTask) (*api.PipelineTask, error) {
	properties := task.GetStatusMetadata().GetCustomProperties()
	generation := properties[util.DriverRetryGenerationKey].GetStringValue()
	attempt := properties[util.DriverRetryAttemptKey].GetStringValue()
	updated, err := client.UpdateTask(ctx, &api.UpdateTaskRequest{RunId: task.RunId, TaskId: task.TaskId, Task: task})
	if err != nil {
		return nil, fmt.Errorf("failed to persist driver recovery state: %w", err)
	}
	if updated == nil || updated.GetTaskId() != task.GetTaskId() {
		return nil, fmt.Errorf("driver recovery update returned an invalid task identity")
	}
	// The response may be hydrated after another attempt has claimed the task.
	// Do not let a refreshed response transfer that attempt's write authority.
	updatedProperties := updated.GetStatusMetadata().GetCustomProperties()
	if attempt != "" && (updatedProperties[util.DriverRetryGenerationKey].GetStringValue() != generation ||
		updatedProperties[util.DriverRetryAttemptKey].GetStringValue() != attempt) {
		return nil, fmt.Errorf("driver task ownership changed during update; discard this stale driver attempt")
	}
	return updated, nil
}

func appendDriverPod(pods []*api.PipelineTask_TaskPod, pod *api.PipelineTask_TaskPod) []*api.PipelineTask_TaskPod {
	for _, existing := range pods {
		if existing.GetUid() == pod.GetUid() && existing.GetName() == pod.GetName() {
			return pods
		}
	}
	return append(pods, pod)
}

func driverOutputAllocationID(opts common.Options) string {
	if !opts.DriverRetryEnabled {
		return uuid.NewString()
	}
	// Parent identity separates identical inner-loop indexes in distinct outer
	// iterations. Manual RetryRun advances generation and gets new locations.
	identity, _ := json.Marshal([]string{
		opts.Run.GetRunId(), strconv.FormatInt(opts.DriverRetryGeneration, 10),
		opts.ParentTask.GetTaskId(), opts.TaskName, opts.ScopePath.DotNotation(),
		strconv.Itoa(opts.IterationIndex), opts.DriverType,
	})
	return uuid.NewSHA1(uuid.NameSpaceOID, identity).String()
}

func marshalDriverCheckpoint(execution *Execution) (string, error) {
	checkpoint := driverCheckpoint{Version: 1, TaskID: execution.TaskID,
		IterationCount: execution.IterationCount, Condition: execution.Condition,
		Cached: execution.Cached, PodSpecPatch: execution.PodSpecPatch}
	if execution.ExecutorInput != nil {
		encoded, err := protojson.Marshal(execution.ExecutorInput)
		if err != nil {
			return "", fmt.Errorf("failed to serialize driver executor input: %w", err)
		}
		checkpoint.ExecutorInput = encoded
	}
	encoded, err := json.Marshal(checkpoint)
	if err != nil {
		return "", fmt.Errorf("failed to serialize driver handoff: %w", err)
	}
	return string(encoded), nil
}

func restoreDriverCheckpoint(encoded, taskID string) (*Execution, error) {
	var checkpoint driverCheckpoint
	if err := json.Unmarshal([]byte(encoded), &checkpoint); err != nil {
		return nil, fmt.Errorf("invalid saved driver handoff; retry the run through the API: %w", err)
	}
	if checkpoint.Version != 1 || checkpoint.TaskID != taskID {
		return nil, fmt.Errorf("saved driver handoff has an unsupported version or task identity; retry the run through the API")
	}
	execution := &Execution{TaskID: taskID, IterationCount: checkpoint.IterationCount,
		Condition: checkpoint.Condition, Cached: checkpoint.Cached, PodSpecPatch: checkpoint.PodSpecPatch}
	if len(checkpoint.ExecutorInput) > 0 {
		execution.ExecutorInput = &pipelinespec.ExecutorInput{}
		if err := protojson.Unmarshal(checkpoint.ExecutorInput, execution.ExecutorInput); err != nil {
			return nil, fmt.Errorf("invalid executor input in saved driver handoff: %w", err)
		}
	}
	return execution, nil
}

func recoveredDriverCache(opts common.Options) (*api.PipelineTask, error) {
	encoded := opts.DriverRetryTask.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue()
	if encoded == "" {
		return nil, nil
	}
	outputs := &api.PipelineTask_InputOutputs{}
	if err := protojson.Unmarshal([]byte(encoded), outputs); err != nil {
		return nil, fmt.Errorf("invalid saved driver cache outputs; retry the run through the API: %w", err)
	}
	return &api.PipelineTask{Outputs: outputs}, nil
}

func saveDriverCacheDecision(ctx context.Context, opts common.Options, client kfpapi.API, outputs *api.PipelineTask_InputOutputs, fingerprint string) error {
	if opts.DriverRetryTask == nil {
		return nil
	}
	encoded, err := protojson.Marshal(outputs)
	if err != nil {
		return fmt.Errorf("failed to serialize driver cache outputs: %w", err)
	}
	task := proto.Clone(opts.DriverRetryTask).(*api.PipelineTask)
	task.CacheFingerprint = fingerprint
	driverRecoveryMetadata(task).CustomProperties[driverCachedOutputsKey] = structpb.NewStringValue(string(encoded))
	updated, err := updateDriverTask(ctx, client, task)
	if err != nil {
		return err
	}
	proto.Reset(opts.DriverRetryTask)
	proto.Merge(opts.DriverRetryTask, updated)
	return nil
}
