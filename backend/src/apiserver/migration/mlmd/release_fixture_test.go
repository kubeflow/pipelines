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

package mlmd_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

type releaseGraph struct {
	Runs []struct {
		ID        string `json:"id"`
		State     string `json:"state"`
		Namespace string `json:"namespace"`
	} `json:"runs"`
	Tasks []struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		RunID             string `json:"run_id"`
		Namespace         string `json:"namespace"`
		State             string `json:"state"`
		ExecutionID       string `json:"execution_id"`
		PodName           string `json:"pod_name"`
		Fingerprint       string `json:"fingerprint"`
		CreatedTimestamp  int64  `json:"created_timestamp"`
		StartedTimestamp  int64  `json:"started_timestamp"`
		FinishedTimestamp int64  `json:"finished_timestamp"`
		StateHistory      string `json:"state_history"`
	} `json:"tasks"`
	Types []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"types"`
	Contexts []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"contexts"`
	Executions []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		TypeID    int    `json:"type_id"`
		State     int    `json:"state"`
		CreatedMS int64  `json:"created_ms"`
		UpdatedMS int64  `json:"updated_ms"`
	} `json:"executions"`
	Associations []struct {
		ContextID   int `json:"context_id"`
		ExecutionID int `json:"execution_id"`
	} `json:"associations"`
	Properties []struct {
		ExecutionID int    `json:"execution_id"`
		Name        string `json:"name"`
		IntValue    *int64 `json:"int_value"`
		StringValue string `json:"string_value"`
	} `json:"execution_properties"`
	Artifacts []struct {
		ID        int    `json:"id"`
		TypeID    int    `json:"type_id"`
		URI       string `json:"uri"`
		CreatedMS int64  `json:"created_ms"`
		UpdatedMS int64  `json:"updated_ms"`
	} `json:"artifacts"`
	ArtifactProperties []struct {
		ArtifactID  int      `json:"artifact_id"`
		Name        string   `json:"name"`
		DoubleValue *float64 `json:"double_value"`
		StringValue string   `json:"string_value"`
	} `json:"artifact_properties"`
	Events []struct {
		ID          int `json:"id"`
		ExecutionID int `json:"execution_id"`
		ArtifactID  int `json:"artifact_id"`
		Type        int `json:"type"`
	} `json:"events"`
	EventPaths []struct {
		EventID int    `json:"event_id"`
		StepKey string `json:"step_key"`
	} `json:"event_paths"`
}

func TestReleasedWriterRoutesAndDrainTimestamps(t *testing.T) {
	var graph releaseGraph
	readFixture(t, "release_218/source_graph.json", &graph)
	contexts := map[string]int{}
	for _, context := range graph.Contexts {
		contexts[context.Name] = context.ID
	}
	for runID, workflowFile := range map[string]string{
		"2ec7ad3b-d821-4b1d-8355-190393a06ddc": "parameters_workflow.json",
		"5dedb489-f0c5-496e-b2e3-eaf46a5fcc74": "parameters_cached_workflow.json",
		"be2ae7cc-da75-4cbb-bdb6-25145b26ae52": "metrics_workflow.json",
	} {
		t.Run(workflowFile, func(t *testing.T) {
			var workflow struct {
				Spec struct {
					Templates []struct {
						Container *struct {
							Image string   `json:"image"`
							Args  []string `json:"args"`
						} `json:"container"`
						InitContainers []struct {
							Image string `json:"image"`
						} `json:"initContainers"`
					} `json:"templates"`
				} `json:"spec"`
				Status struct {
					Phase      string    `json:"phase"`
					FinishedAt time.Time `json:"finishedAt"`
				} `json:"status"`
			}
			readFixture(t, "release_218/"+workflowFile, &workflow)
			require.Equal(t, "Succeeded", workflow.Status.Phase)
			require.False(t, workflow.Status.FinishedAt.IsZero())
			driverFound, launcherFound := false, false
			for _, template := range workflow.Spec.Templates {
				if template.Container != nil && template.Container.Image == "ghcr.io/kubeflow/kfp-driver:2.18.0" {
					driverFound = true
					require.Contains(t, template.Container.Args, "--mlmd_server_address")
					require.Contains(t, template.Container.Args, "--ml_pipeline_server_address")
				}
				for _, init := range template.InitContainers {
					launcherFound = launcherFound || init.Image == "ghcr.io/kubeflow/kfp-launcher:2.18.0"
				}
			}
			require.True(t, driverFound, "missing legacy MLMD and API reporting route")
			require.True(t, launcherFound, "missing legacy launcher writer")
			contextID, found := contexts[runID]
			require.True(t, found)
			executionIDs := map[int]bool{}
			for _, association := range graph.Associations {
				if association.ContextID == contextID {
					executionIDs[association.ExecutionID] = true
				}
			}
			require.NotEmpty(t, executionIDs)
			for _, execution := range graph.Executions {
				if executionIDs[execution.ID] {
					require.Positive(t, execution.UpdatedMS)
					require.LessOrEqual(t, execution.UpdatedMS, workflow.Status.FinishedAt.UnixMilli(), "MLMD execution %d updated after Workflow completion", execution.ID)
				}
			}
			for _, task := range graph.Tasks {
				if task.RunID == runID {
					require.Positive(t, task.FinishedTimestamp)
					require.LessOrEqual(t, task.FinishedTimestamp, workflow.Status.FinishedAt.Unix(), "KFP task %s updated after Workflow completion", task.ID)
				}
			}
		})
	}
}

func TestReleasedPipelineFixtureJoins(t *testing.T) {
	const prefix = "testdata/release_218/"
	var manifest struct {
		Release      string            `json:"release"`
		SourceCommit string            `json:"source_commit"`
		SHA256       map[string]string `json:"sha256"`
	}
	readFixture(t, "release_218/manifest.json", &manifest)
	require.Equal(t, "2.18.0", manifest.Release)
	require.Equal(t, "555bfa60a70da88c594960bdacfd78fa7c061043", manifest.SourceCommit)
	for name, expected := range manifest.SHA256 {
		data, err := os.ReadFile(prefix + name)
		require.NoError(t, err)
		digest := sha256.Sum256(data)
		require.Equal(t, expected, hex.EncodeToString(digest[:]), name)
	}
	var images []struct {
		Image   string `json:"image"`
		ImageID string `json:"imageID"`
	}
	readFixture(t, "release_218/image_provenance.json", &images)
	for _, requiredImage := range []string{
		"ghcr.io/kubeflow/kfp-api-server:2.18.0",
		"ghcr.io/kubeflow/kfp-driver:2.18.0",
		"ghcr.io/kubeflow/kfp-launcher:2.18.0",
		"gcr.io/tfx-oss-public/ml_metadata_store_server:1.14.0",
	} {
		found := false
		for _, image := range images {
			if image.Image == requiredImage && strings.Contains(image.ImageID, "@sha256:") {
				found = true
			}
		}
		require.True(t, found, "missing runtime digest for %s", requiredImage)
	}

	var graph releaseGraph
	readFixture(t, "release_218/source_graph.json", &graph)
	sql, err := os.ReadFile(prefix + "source.sql")
	require.NoError(t, err)

	const firstRun = "2ec7ad3b-d821-4b1d-8355-190393a06ddc"
	const cachedRun = "5dedb489-f0c5-496e-b2e3-eaf46a5fcc74"
	const metricsRun = "be2ae7cc-da75-4cbb-bdb6-25145b26ae52"
	workflows := map[string]string{
		firstRun:   "parameters_workflow.json",
		cachedRun:  "parameters_cached_workflow.json",
		metricsRun: "metrics_workflow.json",
	}
	contexts := map[string]int{}
	for _, context := range graph.Contexts {
		contexts[context.Name] = context.ID
	}
	executions := map[int]struct {
		Name   string
		TypeID int
		State  int
	}{}
	for _, execution := range graph.Executions {
		executions[execution.ID] = struct {
			Name   string
			TypeID int
			State  int
		}{execution.Name, execution.TypeID, execution.State}
	}
	associated := map[int]map[int]bool{}
	for _, association := range graph.Associations {
		if associated[association.ContextID] == nil {
			associated[association.ContextID] = map[int]bool{}
		}
		associated[association.ContextID][association.ExecutionID] = true
	}

	for _, run := range graph.Runs {
		workflowFile, selected := workflows[run.ID]
		if !selected {
			continue // The SQL snapshot also retains one failed exploratory run.
		}
		require.Equal(t, "SUCCEEDED", run.State)
		require.Contains(t, string(sql), run.ID)
		contextID, found := contexts[run.ID]
		require.True(t, found, "missing MLMD run context for %s", run.ID)
		rootFound := false
		for executionID := range associated[contextID] {
			if executions[executionID].Name == "run/"+run.ID {
				rootFound = true
			}
		}
		require.True(t, rootFound, "missing MLMD root execution for %s", run.ID)

		var workflow struct {
			Metadata struct {
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		}
		readFixture(t, "release_218/"+workflowFile, &workflow)
		require.Equal(t, run.ID, workflow.Metadata.Labels["pipeline/runid"])
		require.Equal(t, "Succeeded", workflow.Status.Phase)
	}

	joinedTasks := 0
	for _, task := range graph.Tasks {
		if _, selected := workflows[task.RunID]; !selected || task.ExecutionID == "" {
			continue
		}
		executionID, err := strconv.Atoi(task.ExecutionID)
		require.NoError(t, err)
		_, found := executions[executionID]
		require.True(t, found, "KFP task references missing MLMD execution %d", executionID)
		require.True(t, associated[contexts[task.RunID]][executionID], "task execution belongs to another run")
		joinedTasks++
	}
	require.Equal(t, 6, joinedTasks) // Five parameter tasks and one metrics task.

	cachedExecutions := map[int]bool{}
	for executionID := range associated[contexts[cachedRun]] {
		if executions[executionID].TypeID == 13 {
			require.Equal(t, 5, executions[executionID].State) // MLMD CACHED.
			cachedExecutions[executionID] = true
		}
	}
	require.Len(t, cachedExecutions, 5)
	cacheReferences := 0
	for _, property := range graph.Properties {
		if !cachedExecutions[property.ExecutionID] || property.Name != "cached_execution_id" {
			continue
		}
		originalID, err := strconv.Atoi(property.StringValue)
		require.NoError(t, err)
		require.True(t, associated[contexts[firstRun]][originalID], "cached execution points outside the first run")
		cacheReferences++
	}
	require.Equal(t, 5, cacheReferences)

	metricArtifacts := map[int]bool{}
	for _, artifact := range graph.Artifacts {
		if artifact.TypeID == 15 { // system.Metrics in the released MLMD type table.
			metricArtifacts[artifact.ID] = true
		}
	}
	require.NotEmpty(t, metricArtifacts)
	metricEvent := false
	for _, event := range graph.Events {
		if associated[contexts[metricsRun]][event.ExecutionID] && metricArtifacts[event.ArtifactID] {
			metricEvent = true
		}
	}
	require.True(t, metricEvent, "metrics artifact has no event in the metrics run")
	require.True(t, strings.Contains(string(sql), "INSERT INTO `Execution`"))
}

func TestReleasedNativeProjectionMatchesSource(t *testing.T) {
	var graph releaseGraph
	readFixture(t, "release_218/source_graph.json", &graph)
	var expected struct {
		Tasks []struct {
			SourceExecutionID int `json:"source_execution_id"`
			model.Task
		} `json:"tasks"`
		Artifacts []struct {
			SourceArtifactID int `json:"source_artifact_id"`
			model.Artifact
		} `json:"artifacts"`
		Links []struct {
			SourceEventID int `json:"source_event_id"`
			model.ArtifactTask
		} `json:"links"`
		LegacyProvenance map[string]map[string]string `json:"legacy_provenance"`
	}
	readFixture(t, "release_218/expected_native.json", &expected)

	runs := map[string]string{}
	for _, run := range graph.Runs {
		runs[run.ID] = run.Namespace
	}
	types := map[int]string{}
	for _, kind := range graph.Types {
		types[kind.ID] = kind.Name
	}
	executions := map[int]int{}
	for index, execution := range graph.Executions {
		executions[execution.ID] = index
	}
	properties := map[int]map[string]struct {
		IntValue    *int64
		StringValue string
	}{}
	for _, property := range graph.Properties {
		if properties[property.ExecutionID] == nil {
			properties[property.ExecutionID] = map[string]struct {
				IntValue    *int64
				StringValue string
			}{}
		}
		properties[property.ExecutionID][property.Name] = struct {
			IntValue    *int64
			StringValue string
		}{property.IntValue, property.StringValue}
	}
	legacyTasks := map[int]int{}
	for index, task := range graph.Tasks {
		if task.ExecutionID == "" {
			continue
		}
		id, err := strconv.Atoi(task.ExecutionID)
		require.NoError(t, err)
		require.NotContains(t, legacyTasks, id, "MLMD execution must join at most one KFP row")
		legacyTasks[id] = index
	}

	selected := map[int]model.Task{}
	for _, projection := range expected.Tasks {
		id, found := executions[projection.SourceExecutionID]
		require.True(t, found)
		source := graph.Executions[id]
		task := projection.Task
		require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("execution:"+strconv.Itoa(source.ID))).String(), task.UUID)
		require.Equal(t, runs[task.RunUUID], task.Namespace)
		require.NotEmpty(t, task.Namespace)
		var fingerprint string
		if source.State == 3 || source.State == 5 {
			fingerprint = properties[source.ID]["cache_fingerprint"].StringValue
		}
		require.Equal(t, fingerprint, task.Fingerprint,
			"copy source fingerprints only for COMPLETE/CACHED executions")
		require.Equal(t, map[int]model.TaskStatus{2: model.TaskStatus(apiv2.PipelineTask_RUNNING), 3: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), 5: model.TaskStatus(apiv2.PipelineTask_CACHED)}[source.State], task.State)
		if parent := properties[source.ID]["parent_dag_id"].IntValue; parent != nil {
			parentID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("execution:"+strconv.FormatInt(*parent, 10))).String()
			require.Equal(t, &parentID, task.ParentTaskUUID)
		} else {
			require.Nil(t, task.ParentTaskUUID)
		}
		switch types[source.TypeID] {
		case "system.ContainerExecution":
			require.Equal(t, model.TaskType(apiv2.PipelineTask_RUNTIME), task.Type)
		case "system.DAGExecution":
			want := apiv2.PipelineTask_DAG
			if task.ParentTaskUUID == nil {
				want = apiv2.PipelineTask_ROOT
			} else if properties[source.ID]["iteration_count"].IntValue != nil {
				want = apiv2.PipelineTask_LOOP
			}
			require.Equal(t, model.TaskType(want), task.Type)
		default:
			t.Fatalf("unexpected execution type %q", types[source.TypeID])
		}
		if name := properties[source.ID]["task_name"].StringValue; name != "" {
			require.Equal(t, name, task.Name)
		}
		for sourceKey, nativeKey := range map[string]string{"iteration_count": "iterationCount", "iteration_index": "iterationIndex"} {
			if value := properties[source.ID][sourceKey].IntValue; value != nil {
				require.Equal(t, strconv.FormatInt(*value, 10), task.TypeAttrs[nativeKey])
			}
		}
		if podName := properties[source.ID]["pod_name"].StringValue; podName != "" {
			require.Len(t, task.Pods, 1)
			require.Equal(t, podName, task.Pods[0].(map[string]interface{})["name"], "MLMD owns the execution pod")
		}
		if index, joined := legacyTasks[source.ID]; joined {
			legacy := graph.Tasks[index]
			require.Equal(t, legacy.RunID, task.RunUUID)
			require.Equal(t, legacy.Namespace, task.Namespace)
			require.Equal(t, legacy.CreatedTimestamp, task.CreatedAtInSec, "KFP owns a recorded task creation time")
			require.Equal(t, legacy.StartedTimestamp, task.StartedInSec)
			require.Equal(t, legacy.FinishedTimestamp, task.FinishedInSec)
			require.Empty(t, legacy.PodName, "this released join has no KFP pod value")
			require.Equal(t, "null", legacy.StateHistory)
		} else {
			require.Equal(t, source.CreatedMS/1000, task.CreatedAtInSec)
		}
		selected[source.ID] = task
	}
	require.Len(t, selected, 10)
	producer := selected[19]
	cached := selected[28]
	require.Equal(t, "19", properties[28]["cached_execution_id"].StringValue)
	require.Equal(t, producer.UUID, cached.StatusMetadata["customProperties"].(map[string]interface{})["cached_task_id"])
	require.Equal(t, graph.Tasks[legacyTasks[19]].Fingerprint, expected.LegacyProvenance[producer.UUID]["cache_fingerprint"])
	require.Equal(t, "19", expected.LegacyProvenance[cached.UUID]["cached_execution_id"])
	skippedKFPTasks := 0
	for _, task := range graph.Tasks {
		if task.RunID == cached.RunUUID && task.State == "SKIPPED" && task.ExecutionID == "" {
			require.Equal(t, "executor", task.Name)
			require.Contains(t, task.StateHistory, `"State":"SKIPPED"`)
			skippedKFPTasks++
		}
	}
	require.Equal(t, 5, skippedKFPTasks, "the cached run's KFP task rows must be in the source graph")
	var cachedWorkflow struct {
		Metadata struct {
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec struct {
			Templates []struct {
				DAG *struct {
					Tasks []struct {
						Name string `json:"name"`
						When string `json:"when"`
					} `json:"tasks"`
				} `json:"dag"`
			} `json:"templates"`
		} `json:"spec"`
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	}
	readFixture(t, "release_218/parameters_cached_workflow.json", &cachedWorkflow)
	require.Equal(t, "Succeeded", cachedWorkflow.Status.Phase)
	require.Equal(t, cached.RunUUID, cachedWorkflow.Metadata.Labels["pipeline/runid"])
	var guardedExecutor bool
	for _, template := range cachedWorkflow.Spec.Templates {
		if template.DAG == nil {
			continue
		}
		for _, step := range template.DAG.Tasks {
			if step.Name == "executor" && step.When == "{{inputs.parameters.cached-decision}} != true" {
				guardedExecutor = true
			}
		}
	}
	require.True(t, guardedExecutor, "the submitted Workflow must contain the skip guard for the KFP SKIPPED rows")

	require.Len(t, expected.Artifacts, 2)
	metric := expected.Artifacts[0]
	require.Equal(t, 4, metric.SourceArtifactID)
	require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:4")).String(), metric.UUID)
	require.Equal(t, model.ArtifactType(apiv2.Artifact_Metric), metric.Type)
	require.Equal(t, "accuracy", metric.Name)
	metricValueFound := false
	for _, property := range graph.ArtifactProperties {
		if property.ArtifactID == 4 && property.Name == metric.Name {
			require.Equal(t, property.DoubleValue, metric.NumberValue)
			metricValueFound = true
		}
	}
	require.True(t, metricValueFound, "missing source metric value")
	artifactFound := false
	for _, source := range graph.Artifacts {
		if source.ID == metric.SourceArtifactID {
			artifactFound = true
			require.Equal(t, source.URI, *metric.URI)
			uriHash := sha256.Sum256([]byte(source.URI))
			require.Equal(t, hex.EncodeToString(uriHash[:]), metric.URIHash)
			require.Equal(t, source.CreatedMS/1000, metric.CreatedAtInSec)
			require.Equal(t, source.UpdatedMS/1000, metric.LastUpdateInSec)
		}
	}
	require.True(t, artifactFound, "missing source metric artifact")

	require.Len(t, expected.Links, 2)
	link := expected.Links[0]
	require.Equal(t, 4, link.SourceEventID)
	require.Equal(t, selected[14].UUID, link.TaskID)
	require.Equal(t, metric.UUID, link.ArtifactID)
	require.Equal(t, selected[14].RunUUID, link.RunUUID)
	require.Equal(t, apiv2.IOType_OUTPUT, apiv2.IOType(link.Type))
	eventFound := false
	for _, event := range graph.Events {
		if event.ID == link.SourceEventID {
			eventFound = true
			require.Equal(t, 14, event.ExecutionID)
			require.Equal(t, metric.SourceArtifactID, event.ArtifactID)
			require.Equal(t, 4, event.Type)
		}
	}
	require.True(t, eventFound, "missing source metric output event")
	eventPathFound := false
	for _, path := range graph.EventPaths {
		if path.EventID == link.SourceEventID {
			eventPathFound = true
			require.Equal(t, path.StepKey, link.ArtifactKey)
		}
	}
	require.True(t, eventPathFound, "missing source metric event path")
	name, err := json.Marshal([]any{"artifact-task", link.ArtifactID, link.TaskID, int32(link.Type), link.Iteration, link.ArtifactKey})
	require.NoError(t, err)
	require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, name).String(), link.UUID)

	// The released loop produced an artifact in iteration zero. Its source
	// event and parent DAG property establish the native link iteration.
	loopArtifact := expected.Artifacts[1]
	loopLink := expected.Links[1]
	require.Equal(t, 6, loopArtifact.SourceArtifactID)
	require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:6")).String(), loopArtifact.UUID)
	require.Equal(t, model.ArtifactType(apiv2.Artifact_Artifact), loopArtifact.Type)
	var loopArtifactSeen bool
	for _, source := range graph.Artifacts {
		if source.ID == loopArtifact.SourceArtifactID {
			loopArtifactSeen = true
			require.Equal(t, source.URI, *loopArtifact.URI)
			uriHash := sha256.Sum256([]byte(source.URI))
			require.Equal(t, hex.EncodeToString(uriHash[:]), loopArtifact.URIHash)
			require.Equal(t, source.CreatedMS/1000, loopArtifact.CreatedAtInSec)
		}
	}
	require.True(t, loopArtifactSeen)
	require.Equal(t, 6, loopLink.SourceEventID)
	require.Equal(t, selected[19].UUID, loopLink.TaskID)
	require.Equal(t, loopArtifact.UUID, loopLink.ArtifactID)
	require.Equal(t, int64(0), loopLink.Iteration)
	var parentID int
	var iterationIndex *int64
	for _, property := range graph.Properties {
		if property.ExecutionID == 19 && property.Name == "parent_dag_id" {
			require.NotNil(t, property.IntValue)
			parentID = int(*property.IntValue)
		}
	}
	for _, property := range graph.Properties {
		if property.ExecutionID == parentID && property.Name == "iteration_index" {
			iterationIndex = property.IntValue
		}
	}
	require.Equal(t, 15, parentID)
	require.NotNil(t, iterationIndex)
	require.Equal(t, *iterationIndex, loopLink.Iteration)
	var eventSeen, pathSeen bool
	for _, event := range graph.Events {
		if event.ID == loopLink.SourceEventID {
			eventSeen = true
			require.Equal(t, 19, event.ExecutionID)
			require.Equal(t, loopArtifact.SourceArtifactID, event.ArtifactID)
			require.Equal(t, 4, event.Type)
		}
	}
	for _, path := range graph.EventPaths {
		if path.EventID == loopLink.SourceEventID {
			pathSeen = true
			require.Equal(t, path.StepKey, loopLink.ArtifactKey)
		}
	}
	require.True(t, eventSeen)
	require.True(t, pathSeen)
	name, err = json.Marshal([]any{"artifact-task", loopLink.ArtifactID, loopLink.TaskID, int32(loopLink.Type), loopLink.Iteration, loopLink.ArtifactKey})
	require.NoError(t, err)
	require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, name).String(), loopLink.UUID)
}
