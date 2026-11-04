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
	"sort"
	"strconv"
	"unicode/utf8"

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
	Source string `json:"source,omitempty"`
}

// Plan is the native graph prepared from one stable source snapshot.
type Plan struct {
	Tasks         []model.Task
	Artifacts     []model.Artifact
	artifactSizes []int
	Links         []model.ArtifactTask
	Skipped       []Skip
	Warnings      []string
}

func runContext(contexts []*pb.Context, runs map[string]string) (string, error) {
	var run string
	for _, entry := range contexts {
		if entry.GetType() != "system.PipelineRun" {
			continue
		}
		name := entry.GetName()
		if _, exists := runs[name]; !exists || name == "" {
			continue
		}
		if run != "" && run != name {
			return "", fmt.Errorf("multiple PipelineRun contexts match KFP runs")
		}
		run = name
	}
	if run == "" {
		return "", fmt.Errorf("PipelineRun context has no KFP run_details row")
	}
	return run, nil
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
	if source.Namespace != "" && source.Namespace != model.NoNamespace && source.Namespace != task.Namespace {
		return fmt.Errorf("legacy cache row namespace %q differs from run namespace %q", source.Namespace, task.Namespace)
	}
	if source.PodName != "" {
		pod := map[string]any{"name": source.PodName}
		if uid := propertyString(execution.GetCustomProperties(), "pod_uid"); uid != "" && source.PodName == propertyString(execution.GetCustomProperties(), "pod_name") {
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

func (plan *Plan) skipEvent(event *pb.Event, reason string) {
	// The MLMD wire event has no numeric Event.id.
	source := fmt.Sprintf("execution=%d artifact=%d type=%s time=%d path=%s",
		event.GetExecutionId(), event.GetArtifactId(), event.GetType(), event.GetMillisecondsSinceEpoch(), event.GetPath())
	plan.Skipped = append(plan.Skipped, Skip{Kind: "event", ID: event.GetExecutionId(), Reason: reason, Source: source})
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
	legacyJoinErrors := map[int64]error{}
	for _, execution := range snapshot.Executions {
		id := execution.GetId()
		runUUID, err := runContext(snapshot.Contexts[id], runs)
		if err != nil {
			plan.Skipped = append(plan.Skipped, Skip{Kind: "execution", ID: id, Reason: err.Error()})
			continue
		}
		namespace := runs[runUUID]
		if sourceNamespace := propertyString(execution.GetCustomProperties(), "namespace"); sourceNamespace != "" &&
			sourceNamespace != model.NoNamespace && sourceNamespace != namespace {
			plan.Skipped = append(plan.Skipped, Skip{Kind: "execution", ID: id, Reason: "MLMD execution namespace differs from its KFP run"})
			continue
		}
		task, parent, err := TransformExecution(execution, runUUID, namespace)
		if err != nil {
			plan.Skipped = append(plan.Skipped, Skip{Kind: "execution", ID: id, Reason: err.Error()})
			continue
		}
		if source, ok := legacy[id]; ok {
			merged := task
			if err := mergeLegacy(&merged, source, execution); err != nil {
				legacyJoinErrors[id] = err
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("execution %d: ignored legacy cache row %s: %v", id, source.UUID, err))
			} else {
				task = merged
			}
		}
		if propertyString(execution.GetCustomProperties(), "display_name") == "" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("execution %d has no display_name; using an empty display name", id))
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
			if found && parentTask.RunUUID != task.RunUUID {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("execution %d: ignored cross-run parent %d", id, parent))
				delete(parents, id)
				continue
			}
			if !found {
				plan.Skipped = append(plan.Skipped, Skip{Kind: "execution", ID: id, Reason: "parent execution " + strconv.FormatInt(parent, 10) + " is absent from the same run"})
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
		parentTask := tasks[parent]
		if task.Type == model.TaskType(apiv2.PipelineTask_CONDITION) &&
			parentTask.Type != model.TaskType(apiv2.PipelineTask_CONDITION_BRANCH) &&
			(parentTask.Type != model.TaskType(apiv2.PipelineTask_ROOT) || !releasedConditionName.MatchString(task.Name)) {
			task.Type = model.TaskType(apiv2.PipelineTask_DAG)
		}
		parentUUID := parentTask.UUID
		task.ParentTaskUUID = &parentUUID
		tasks[id] = task
	}
	for _, execution := range snapshot.Executions {
		id := execution.GetId()
		if task, found := tasks[id]; found {
			if task.State == model.TaskStatus(apiv2.PipelineTask_CACHED) {
				producerID, _ := strconv.ParseInt(propertyString(execution.GetCustomProperties(), "cached_execution_id"), 10, 64)
				if producer, ok := tasks[producerID]; producerID != id && ok && producer.Namespace == task.Namespace {
					task.CachedTaskUUID = &producer.UUID
				} else if producerID != 0 {
					plan.Warnings = append(plan.Warnings, fmt.Sprintf("execution %d: cached producer %d could not be verified", id, producerID))
				}
			}
			plan.Tasks = append(plan.Tasks, task)
		}
	}
	for id, row := range legacy {
		reason := "no verified migrated execution"
		if err := legacyJoinErrors[id]; err != nil {
			reason = err.Error()
		} else if _, retained := tasks[id]; retained {
			continue
		}
		plan.Skipped = append(plan.Skipped, Skip{Kind: "legacy_cache_row", ID: id, Source: row.UUID, Reason: reason})
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
			plan.Skipped = append(plan.Skipped, Skip{Kind: "artifact", ID: event.GetArtifactId(), Reason: "source artifact spans KFP namespaces"})
			delete(artifactNamespace, event.GetArtifactId())
			conflictingArtifact[event.GetArtifactId()] = true
			continue
		}
		artifactNamespace[event.GetArtifactId()] = task.Namespace
	}
	for _, artifact := range snapshot.Artifacts {
		if namespace, found := artifactNamespace[artifact.GetId()]; found {
			rows := TransformArtifact(artifact, namespace)
			if rows[0].Name == "" {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("artifact %d has no source name", artifact.GetId()))
			}
			invalid := utf8.RuneCountInString(artifact.GetUri()) > 2048 || utf8.RuneCountInString(namespace) > 63
			for _, row := range rows {
				if utf8.RuneCountInString(row.Name) > 128 {
					invalid = true
				}
			}
			if invalid {
				plan.Skipped = append(plan.Skipped, Skip{Kind: "artifact", ID: artifact.GetId(), Reason: "artifact exceeds native URI, name, or namespace width"})
				delete(artifactNamespace, artifact.GetId())
				continue
			}
			plan.Artifacts = append(plan.Artifacts, rows...)
			plan.artifactSizes = append(plan.artifactSizes, len(rows))
		} else if !conflictingArtifact[artifact.GetId()] {
			plan.Skipped = append(plan.Skipped, Skip{Kind: "artifact", ID: artifact.GetId(), Reason: "no retained execution event owns the artifact"})
		}
	}
	for _, event := range snapshot.Events {
		task, hasTask := tasks[event.GetExecutionId()]
		_, hasArtifact := artifactSources[event.GetArtifactId()]
		_, hasNamespace := artifactNamespace[event.GetArtifactId()]
		if !hasTask {
			plan.skipEvent(event, "execution was skipped")
			continue
		}
		if !hasArtifact || !hasNamespace {
			plan.skipEvent(event, "artifact is missing or skipped")
			continue
		}
		key := ""
		steps := event.GetPath().GetSteps()
		if len(steps) == 1 {
			key = steps[0].GetKey()
		}
		if len(steps) != 1 || key == "" {
			plan.skipEvent(event, "indexed, empty, or composite path is unsupported")
			continue
		}
		if utf8.RuneCountInString(key) > 191 {
			plan.skipEvent(event, "artifact key exceeds native width")
			continue
		}
		var producer *apiv2.IOProducer
		if event.GetType() == pb.Event_OUTPUT {
			iteration, err := producerIteration(event.GetExecutionId(), tasks, parents)
			if err != nil {
				plan.skipEvent(event, err.Error())
				continue
			}
			if iteration != nil {
				producer = &apiv2.IOProducer{TaskName: task.Name, Iteration: iteration}
			}
		}
		link, err := TransformEvent(event, task.RunUUID, key, producer)
		if err != nil {
			plan.skipEvent(event, err.Error())
			continue
		}
		if seenLinks[link.UUID] {
			plan.skipEvent(event, "equivalent event already produced the same native link")
			continue
		}
		seenLinks[link.UUID] = true
		plan.Links = append(plan.Links, link)
	}
	// MLMD gRPC does not expose Event.id. The native relationship identity
	// supplies a stable source order even when gRPC returns links in a different order.
	sort.Slice(plan.Links, func(i, j int) bool { return plan.Links[i].UUID < plan.Links[j].UUID })
	return plan
}
