// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

// LegacyTask holds KFP-owned fields from the pre-migration tasks table.
type LegacyTask struct {
	ExecutionID                                      int64
	RunUUID, Namespace, PodName, State, StateHistory string
	CreatedAtInSec, StartedInSec, FinishedInSec      int64
	UUID, Name, ParentTaskUUID, PipelineName         string
	Fingerprint, MLMDInputs, MLMDOutputs             string
	ChildrenPods, Payload                            string
}

// Skip records one source entity omitted from the native graph.
type Skip struct {
	Kind   string `json:"kind"`
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}

// Plan is the native graph prepared from one stable source snapshot.
type Plan struct {
	Tasks     []model.Task
	Artifacts []model.Artifact
	Links     []model.ArtifactTask
	Skipped   []Skip
	Warnings  []string
}

func runContext(contexts []*pb.Context) string {
	for _, context := range contexts {
		if context.GetType() == "system.PipelineRun" {
			return context.GetName()
		}
	}
	return ""
}

func legacyState(value string) (model.TaskStatus, error) {
	states := map[string]apiv2.PipelineTask_TaskState{
		"":                          apiv2.PipelineTask_RUNTIME_STATE_UNSPECIFIED,
		"RUNTIME_STATE_UNSPECIFIED": apiv2.PipelineTask_RUNTIME_STATE_UNSPECIFIED,
		"UNSPECIFIED":               apiv2.PipelineTask_RUNTIME_STATE_UNSPECIFIED,
		"RUNNING":                   apiv2.PipelineTask_RUNNING,
		"SUCCEEDED":                 apiv2.PipelineTask_SUCCEEDED,
		"SKIPPED":                   apiv2.PipelineTask_SKIPPED,
		"FAILED":                    apiv2.PipelineTask_FAILED,
		"CACHED":                    apiv2.PipelineTask_CACHED,
	}
	state, found := states[value]
	if !found {
		return 0, fmt.Errorf("unsupported KFP task state %q", value)
	}
	return model.TaskStatus(state), nil
}

func mergeLegacy(task *model.Task, source LegacyTask, execution *pb.Execution) error {
	if source.RunUUID != task.RunUUID {
		return fmt.Errorf("KFP task run %q differs from MLMD run %q", source.RunUUID, task.RunUUID)
	}
	if source.Namespace != "" {
		task.Namespace = source.Namespace
	}
	if source.PodName != "" {
		pod := map[string]any{"name": source.PodName, "type": "EXECUTOR"}
		if uid := propertyString(execution.GetCustomProperties(), "pod_uid"); uid != "" {
			pod["uid"] = uid
		}
		task.Pods = model.JSONSlice{pod}
	}
	if source.CreatedAtInSec != 0 {
		task.CreatedAtInSec = source.CreatedAtInSec
	}
	if source.StartedInSec != 0 {
		task.StartedInSec = source.StartedInSec
	}
	if source.FinishedInSec != 0 {
		task.FinishedInSec = source.FinishedInSec
	}
	if source.State != "" && source.State != "RUNTIME_STATE_UNSPECIFIED" && source.State != "UNSPECIFIED" {
		state, err := legacyState(source.State)
		if err != nil {
			return err
		}
		task.State = state
	}
	if source.StateHistory != "" {
		if err := json.Unmarshal([]byte(source.StateHistory), &task.StateHistory); err != nil {
			return fmt.Errorf("decode KFP state history: %w", err)
		}
	}
	if extra := legacyProperties(source, task); len(extra) != 0 {
		if task.StatusMetadata == nil {
			task.StatusMetadata = model.JSONData{}
		}
		custom, _ := task.StatusMetadata["customProperties"].(map[string]any)
		if custom == nil {
			custom = map[string]any{}
		}
		for key, value := range extra {
			custom["legacy_task."+key] = value
		}
		task.StatusMetadata["customProperties"] = custom
	}
	return nil
}

func legacyOnlyTask(source LegacyTask) (model.Task, error) {
	state, err := legacyState(source.State)
	if err != nil {
		return model.Task{}, err
	}
	if len(source.Name) > 128 || len(source.Namespace) > 63 {
		return model.Task{}, fmt.Errorf("KFP task %s exceeds native name or namespace width", source.UUID)
	}
	task := model.Task{
		UUID: source.UUID, RunUUID: source.RunUUID, Namespace: source.Namespace,
		Name: source.Name, State: state,
		Type:      model.TaskType(apiv2.PipelineTask_TASK_TYPE_UNSPECIFIED),
		TypeAttrs: model.JSONData{}, Pods: model.JSONSlice{},
		Fingerprint:    "",
		CreatedAtInSec: source.CreatedAtInSec, StartedInSec: source.StartedInSec,
		FinishedInSec: source.FinishedInSec,
	}
	if source.PodName != "" {
		task.Pods = model.JSONSlice{map[string]any{"name": source.PodName, "type": "EXECUTOR"}}
	}
	if source.StateHistory != "" {
		if err := json.Unmarshal([]byte(source.StateHistory), &task.StateHistory); err != nil {
			return model.Task{}, fmt.Errorf("KFP task %s has invalid state history: %w", source.UUID, err)
		}
	}
	preserveLegacyFingerprint(&task, source.Fingerprint)
	if extra := legacyProperties(source, &task); len(extra) != 0 {
		if task.StatusMetadata == nil {
			task.StatusMetadata = model.JSONData{}
		}
		custom, _ := task.StatusMetadata["customProperties"].(map[string]any)
		if custom == nil {
			custom = map[string]any{}
		}
		for key, value := range extra {
			custom["legacy_task."+key] = value
		}
		task.StatusMetadata["customProperties"] = custom
	}
	return task, nil
}

func appendUnjoinedLegacy(plan *Plan, rows []LegacyTask, joined map[int64]LegacyTask, runs map[string]string) error {
	redirect := map[string]string{}
	known := map[string]bool{}
	for id, source := range joined {
		redirect[source.UUID] = ExecutionUUID(id)
	}
	for _, task := range plan.Tasks {
		known[task.UUID] = true
	}
	for _, source := range rows {
		namespace, found := runs[source.RunUUID]
		if !found || source.Namespace != namespace {
			return fmt.Errorf("KFP task %s has no matching run and namespace", source.UUID)
		}
		if known[source.UUID] {
			return fmt.Errorf("KFP task %s collides with a migrated execution", source.UUID)
		}
		task, err := legacyOnlyTask(source)
		if err != nil {
			return err
		}
		plan.Tasks = append(plan.Tasks, task)
		known[task.UUID] = true
	}
	for index := range rows {
		parent := rows[index].ParentTaskUUID
		if parent == "" {
			continue
		}
		if mapped := redirect[parent]; mapped != "" {
			parent = mapped
		}
		if !known[parent] {
			return fmt.Errorf("KFP task %s references missing parent %s", rows[index].UUID, rows[index].ParentTaskUUID)
		}
		plan.Tasks[len(plan.Tasks)-len(rows)+index].ParentTaskUUID = &parent
	}
	return nil
}

func legacyProperties(source LegacyTask, task *model.Task) map[string]any {
	extra := map[string]any{}
	for key, value := range map[string]string{
		"pipelineName": source.PipelineName,
		"mlmdInputs":   source.MLMDInputs, "mlmdOutputs": source.MLMDOutputs,
		"childrenPods": source.ChildrenPods, "payload": source.Payload,
	} {
		if value != "" {
			extra[key] = value
		}
	}
	if source.UUID != "" && source.UUID != task.UUID {
		extra["sourceTaskUUID"] = source.UUID
	}
	if source.Name != "" && source.Name != task.Name {
		extra["sourceTaskName"] = source.Name
	}
	legacyFingerprint := ""
	if custom, ok := task.StatusMetadata["customProperties"].(map[string]any); ok {
		legacyFingerprint, _ = custom["legacy_cache_fingerprint"].(string)
	}
	if source.Fingerprint != "" && source.Fingerprint != legacyFingerprint {
		extra["sourceFingerprint"] = source.Fingerprint
	}
	if source.ParentTaskUUID != "" {
		extra["sourceParentTaskUUID"] = source.ParentTaskUUID
	}
	return extra
}

func producerIteration(executionID int64, tasks map[int64]model.Task, parents map[int64]int64) (*int64, error) {
	seen := map[int64]bool{}
	for id := executionID; id != 0 && !seen[id]; id = parents[id] {
		seen[id] = true
		value, found := tasks[id].TypeAttrs["iterationIndex"]
		if !found {
			continue
		}
		index, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("execution %d has invalid iteration index: %w", id, err)
		}
		return &index, nil
	}
	return nil, nil
}

// BuildPlan joins MLMD entities to KFP runs and legacy task rows. It only
// includes events whose task and artifact both survived source validation.
func BuildPlan(snapshot *Snapshot, runs map[string]string, legacy map[int64]LegacyTask) Plan {
	plan := Plan{}
	if snapshot == nil {
		return plan
	}
	tasks := map[int64]model.Task{}
	parents := map[int64]int64{}
	for _, execution := range snapshot.Executions {
		id := execution.GetId()
		runUUID := runContext(snapshot.Contexts[id])
		namespace, found := runs[runUUID]
		if !found || runUUID == "" {
			plan.Skipped = append(plan.Skipped, Skip{"execution", id, "PipelineRun context has no KFP run_details row"})
			continue
		}
		task, parent, err := TransformExecution(execution, runUUID, namespace)
		if err == nil {
			if source, ok := legacy[id]; ok {
				err = mergeLegacy(&task, source, execution)
			}
		}
		if err != nil {
			plan.Skipped = append(plan.Skipped, Skip{"execution", id, err.Error()})
			continue
		}
		if propertyString(execution.GetCustomProperties(), "display_name") == "" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("execution %d has no display_name; using the task name", id))
			if task.DisplayName == "" {
				task.DisplayName = task.Name
			}
		}
		tasks[id] = task
		if parent != 0 {
			parents[id] = parent
		}
	}
	for changed := true; changed; {
		changed = false
		for id, parent := range parents {
			task, childExists := tasks[id]
			if !childExists {
				continue
			}
			parentTask, found := tasks[parent]
			if !found || parentTask.RunUUID != task.RunUUID {
				plan.Skipped = append(plan.Skipped, Skip{"execution", id, "parent execution " + strconv.FormatInt(parent, 10) + " is absent from the same run"})
				delete(tasks, id)
				changed = true
			}
		}
	}
	for id, parent := range parents {
		task, found := tasks[id]
		if !found {
			continue
		}
		parentUUID := tasks[parent].UUID
		task.ParentTaskUUID = &parentUUID
		tasks[id] = task
	}
	for _, execution := range snapshot.Executions {
		if task, found := tasks[execution.GetId()]; found {
			plan.Tasks = append(plan.Tasks, task)
		}
	}
	artifactSources := map[int64]*pb.Artifact{}
	for _, artifact := range snapshot.Artifacts {
		artifactSources[artifact.GetId()] = artifact
	}
	artifactNamespace := map[int64]string{}
	conflictingArtifact := map[int64]bool{}
	seenLinks := map[string]bool{}
	for _, event := range snapshot.Events {
		task, found := tasks[event.GetExecutionId()]
		if !found {
			continue
		}
		if conflictingArtifact[event.GetArtifactId()] {
			continue
		}
		if previous, seen := artifactNamespace[event.GetArtifactId()]; seen && previous != task.Namespace {
			plan.Skipped = append(plan.Skipped, Skip{"artifact", event.GetArtifactId(), "source artifact spans KFP namespaces"})
			delete(artifactNamespace, event.GetArtifactId())
			conflictingArtifact[event.GetArtifactId()] = true
			continue
		}
		artifactNamespace[event.GetArtifactId()] = task.Namespace
	}
	for _, artifact := range snapshot.Artifacts {
		if namespace, found := artifactNamespace[artifact.GetId()]; found {
			if propertyString(artifact.GetCustomProperties(), "display_name") == "" {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("artifact %d has no display_name; using an empty name", artifact.GetId()))
			}
			rows := TransformArtifact(artifact, namespace)
			invalid := len(artifact.GetUri()) > 2048 || len(namespace) > 63
			for _, row := range rows {
				if len(row.Name) > 128 {
					invalid = true
				}
			}
			if invalid {
				plan.Skipped = append(plan.Skipped, Skip{"artifact", artifact.GetId(), "artifact exceeds native URI, name, or namespace width"})
				delete(artifactNamespace, artifact.GetId())
				continue
			}
			plan.Artifacts = append(plan.Artifacts, rows...)
		} else {
			plan.Skipped = append(plan.Skipped, Skip{"artifact", artifact.GetId(), "no retained execution event owns the artifact"})
		}
	}
	for _, event := range snapshot.Events {
		task, hasTask := tasks[event.GetExecutionId()]
		_, hasArtifact := artifactSources[event.GetArtifactId()]
		_, hasNamespace := artifactNamespace[event.GetArtifactId()]
		if !hasTask {
			plan.Skipped = append(plan.Skipped, Skip{"event", event.GetExecutionId(), fmt.Sprintf("artifact %d: execution was skipped", event.GetArtifactId())})
			continue
		}
		if !hasArtifact || !hasNamespace {
			plan.Skipped = append(plan.Skipped, Skip{"event", event.GetExecutionId(), fmt.Sprintf("artifact %d: artifact is missing or skipped", event.GetArtifactId())})
			continue
		}
		key := ""
		if steps := event.GetPath().GetSteps(); len(steps) > 0 {
			key = steps[0].GetKey()
		}
		if len(key) > 191 {
			plan.Skipped = append(plan.Skipped, Skip{"event", event.GetExecutionId(), fmt.Sprintf("artifact %d key %q exceeds native width", event.GetArtifactId(), key)})
			continue
		}
		var producer *apiv2.IOProducer
		if event.GetType() == pb.Event_OUTPUT {
			iteration, err := producerIteration(event.GetExecutionId(), tasks, parents)
			if err != nil {
				plan.Skipped = append(plan.Skipped, Skip{"event", event.GetExecutionId(), fmt.Sprintf("artifact %d: %v", event.GetArtifactId(), err)})
				continue
			}
			if iteration != nil {
				producer = &apiv2.IOProducer{TaskName: task.Name, Iteration: iteration}
			}
		}
		link, err := TransformEvent(event, task.RunUUID, key, producer)
		if err != nil {
			plan.Skipped = append(plan.Skipped, Skip{"event", event.GetExecutionId(), fmt.Sprintf("artifact %d key %q: %v", event.GetArtifactId(), key, err)})
			continue
		}
		if seenLinks[link.UUID] {
			plan.Skipped = append(plan.Skipped, Skip{"event", event.GetExecutionId(), fmt.Sprintf("artifact %d key %q duplicates another source event", event.GetArtifactId(), key)})
			continue
		}
		seenLinks[link.UUID] = true
		plan.Links = append(plan.Links, link)
	}
	return plan
}
