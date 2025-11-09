// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package component provides component launcher functionality for KFP v2.
package component

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
)

// BatchUpdater collects API updates during execution and flushes them
// in batches to reduce database round-trips.
type BatchUpdater struct {
	// Map of task ID to the latest task update
	// Using a map automatically deduplicates multiple updates to the same task
	taskUpdates map[string]*apiV2beta1.PipelineTaskDetail

	// Artifact-task relationships to create
	// We can use the existing bulk API for these
	artifactTasks []*apiV2beta1.ArtifactTask

	// Artifacts to create
	// These need to be created before artifact-tasks that reference them
	artifacts []*createArtifactRequest

	// Metrics for tracking improvement
	queuedTaskUpdates       int
	queuedArtifactTasks     int
	queuedArtifacts         int
	dedupedTaskUpdates      int
	actualTaskUpdateCalls   int
	actualArtifactCalls     int
	actualArtifactTaskCalls int
}

// createArtifactRequest stores the full context needed to create an artifact
type createArtifactRequest struct {
	request *apiV2beta1.CreateArtifactRequest
}

// NewBatchUpdater creates a new BatchUpdater
func NewBatchUpdater() *BatchUpdater {
	return &BatchUpdater{
		taskUpdates:   make(map[string]*apiV2beta1.PipelineTaskDetail),
		artifactTasks: make([]*apiV2beta1.ArtifactTask, 0),
		artifacts:     make([]*createArtifactRequest, 0),
	}
}

// QueueTaskUpdate queues a task update. If the same task is updated multiple times,
// the updates are merged (parameters and artifacts are accumulated, status is taken from latest).
func (b *BatchUpdater) QueueTaskUpdate(task *apiV2beta1.PipelineTaskDetail) {
	if task == nil || task.TaskId == "" {
		glog.Warning("Attempted to queue nil task or task with empty ID")
		return
	}

	// Check if we already have an update for this task
	if existingTask, exists := b.taskUpdates[task.TaskId]; exists {
		b.dedupedTaskUpdates++
		glog.V(2).Infof("Merging task update for task %s", task.TaskId)

		// Merge the updates:
		// 1. Accumulate output parameters (append new ones)
		if task.Outputs != nil && len(task.Outputs.Parameters) > 0 {
			if existingTask.Outputs == nil {
				existingTask.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{}
			}
			existingTask.Outputs.Parameters = append(existingTask.Outputs.Parameters, task.Outputs.Parameters...)
		}

		// 2. Accumulate output artifacts (append new ones)
		if task.Outputs != nil && len(task.Outputs.Artifacts) > 0 {
			if existingTask.Outputs == nil {
				existingTask.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{}
			}
			existingTask.Outputs.Artifacts = append(existingTask.Outputs.Artifacts, task.Outputs.Artifacts...)
		}

		// 3. Accumulate input parameters (append new ones)
		if task.Inputs != nil && len(task.Inputs.Parameters) > 0 {
			if existingTask.Inputs == nil {
				existingTask.Inputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{}
			}
			existingTask.Inputs.Parameters = append(existingTask.Inputs.Parameters, task.Inputs.Parameters...)
		}

		// 4. Accumulate input artifacts (append new ones)
		if task.Inputs != nil && len(task.Inputs.Artifacts) > 0 {
			if existingTask.Inputs == nil {
				existingTask.Inputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{}
			}
			existingTask.Inputs.Artifacts = append(existingTask.Inputs.Artifacts, task.Inputs.Artifacts...)
		}

		// 5. Take the latest status and timestamps
		if task.State != apiV2beta1.PipelineTaskDetail_RUNTIME_STATE_UNSPECIFIED {
			existingTask.State = task.State
		}
		if task.EndTime != nil {
			existingTask.EndTime = task.EndTime
		}
		if task.StartTime != nil {
			existingTask.StartTime = task.StartTime
		}
	} else {
		// First update for this task
		b.taskUpdates[task.TaskId] = task
	}

	b.queuedTaskUpdates++
}

// QueueArtifactTask queues an artifact-task relationship to create
func (b *BatchUpdater) QueueArtifactTask(artifactTask *apiV2beta1.ArtifactTask) {
	if artifactTask == nil {
		glog.Warning("Attempted to queue nil artifact task")
		return
	}

	b.artifactTasks = append(b.artifactTasks, artifactTask)
	b.queuedArtifactTasks++
}

// QueueArtifact queues an artifact to create
func (b *BatchUpdater) QueueArtifact(request *apiV2beta1.CreateArtifactRequest) {
	if request == nil {
		glog.Warning("Attempted to queue nil artifact")
		return
	}

	b.artifacts = append(b.artifacts, &createArtifactRequest{request: request})
	b.queuedArtifacts++
}

// Flush executes all queued updates in batches
// Order of operations:
// 1. Create artifacts (they need to exist before artifact-tasks can reference them)
// 2. Update tasks (task updates can happen in parallel with artifact creation)
// 3. Create artifact-tasks (these depend on artifacts existing)
func (b *BatchUpdater) Flush(ctx context.Context, client kfpapi.API) error {
	if len(b.taskUpdates) == 0 && len(b.artifactTasks) == 0 && len(b.artifacts) == 0 {
		glog.V(2).Info("BatchUpdater: No updates to flush")
		return nil
	}

	glog.Infof("BatchUpdater: Flushing %d task updates (deduped from %d), %d artifacts, %d artifact-tasks",
		len(b.taskUpdates), b.queuedTaskUpdates, len(b.artifacts), len(b.artifactTasks))

	// Print details about what we're flushing
	for taskID, task := range b.taskUpdates {
		glog.V(1).Infof("  Task update: %s, status=%v, outputs: %d params, %d artifacts",
			taskID, task.State,
			len(task.GetOutputs().GetParameters()),
			len(task.GetOutputs().GetArtifacts()))
	}
	for i, artifact := range b.artifacts {
		glog.V(1).Infof("  Artifact #%d: name=%s, taskID=%s, key=%s",
			i, artifact.request.Artifact.Name, artifact.request.TaskId, artifact.request.ProducerKey)
	}
	for i, at := range b.artifactTasks {
		glog.V(1).Infof("  ArtifactTask #%d: artifactID=%s, taskID=%s, key=%s, type=%v",
			i, at.ArtifactId, at.TaskId, at.Key, at.Type)
	}

	// Step 1: Create artifacts using bulk API
	if len(b.artifacts) > 0 {
		bulkReq := &apiV2beta1.CreateArtifactsBulkRequest{
			Artifacts: make([]*apiV2beta1.CreateArtifactRequest, 0, len(b.artifacts)),
		}
		for _, artifactReq := range b.artifacts {
			bulkReq.Artifacts = append(bulkReq.Artifacts, artifactReq.request)
		}
		_, err := client.CreateArtifactsBulk(ctx, bulkReq)
		if err != nil {
			return fmt.Errorf("failed to create artifacts in bulk: %w", err)
		}
		b.actualArtifactCalls = 1 // Bulk call counts as 1
	}

	// Step 2: Update tasks using bulk API
	if len(b.taskUpdates) > 0 {
		_, err := client.UpdateTasksBulk(ctx, &apiV2beta1.UpdateTasksBulkRequest{
			Tasks: b.taskUpdates,
		})
		if err != nil {
			return fmt.Errorf("failed to update tasks in bulk: %w", err)
		}
		b.actualTaskUpdateCalls = 1 // Bulk call counts as 1
	}

	// Step 3: Create artifact-tasks using existing bulk API
	if len(b.artifactTasks) > 0 {
		_, err := client.CreateArtifactTasks(ctx, &apiV2beta1.CreateArtifactTasksBulkRequest{
			ArtifactTasks: b.artifactTasks,
		})
		if err != nil {
			return fmt.Errorf("failed to create artifact-tasks in bulk: %w", err)
		}
		b.actualArtifactTaskCalls = 1 // Bulk call counts as 1
	}

	// Log metrics
	glog.Infof("BatchUpdater metrics - Queued: %d task updates, %d artifacts, %d artifact-tasks | "+
		"Deduped: %d task updates | Actual API calls: %d task updates, %d artifacts, %d artifact-task calls",
		b.queuedTaskUpdates, b.queuedArtifacts, b.queuedArtifactTasks,
		b.dedupedTaskUpdates,
		b.actualTaskUpdateCalls, b.actualArtifactCalls, b.actualArtifactTaskCalls)

	// Reset for next batch
	b.reset()

	return nil
}

// reset clears all queued updates (called after flush)
func (b *BatchUpdater) reset() {
	b.taskUpdates = make(map[string]*apiV2beta1.PipelineTaskDetail)
	b.artifactTasks = make([]*apiV2beta1.ArtifactTask, 0)
	b.artifacts = make([]*createArtifactRequest, 0)

	// Keep metrics across resets for the lifetime of the BatchUpdater
}

// GetMetrics returns the current metrics
func (b *BatchUpdater) GetMetrics() map[string]int {
	return map[string]int{
		"queued_task_updates":        b.queuedTaskUpdates,
		"queued_artifacts":           b.queuedArtifacts,
		"queued_artifact_tasks":      b.queuedArtifactTasks,
		"deduped_task_updates":       b.dedupedTaskUpdates,
		"actual_task_update_calls":   b.actualTaskUpdateCalls,
		"actual_artifact_calls":      b.actualArtifactCalls,
		"actual_artifact_task_calls": b.actualArtifactTaskCalls,
	}
}
