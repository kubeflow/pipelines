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
	"net/url"
	"os"
	"regexp"
	"strconv"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/cachekey"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// These are fixture readers, not a second implementation of the migrator.
type sourceFixture struct {
	SourceCommit string           `json:"source_commit"`
	Cases        map[string]int64 `json:"cases"`
	Executions   struct {
		Executions []struct {
			ID               string                     `json:"id"`
			Name             string                     `json:"name"`
			State            string                     `json:"lastKnownState"`
			CustomProperties map[string]json.RawMessage `json:"customProperties"`
		} `json:"executions"`
	} `json:"executions"`
	Cache struct {
		Inputs      json.RawMessage `json:"inputs"`
		Outputs     json.RawMessage `json:"outputs"`
		Key         json.RawMessage `json:"key"`
		Command     []string        `json:"command"`
		Image       string          `json:"image"`
		Fingerprint string          `json:"fingerprint"`
	} `json:"cache"`
}

type nativeFixture struct {
	Tasks             []model.Task                 `json:"tasks"`
	Artifacts         []model.Artifact             `json:"artifacts"`
	Links             []model.ArtifactTask         `json:"links"`
	LegacyProvenance  map[string]map[string]string `json:"legacy_provenance"`
	SkippedExecutions []int64                      `json:"skipped_executions"`
}

func readFixture(t *testing.T, path string, value any) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, value))
}

func TestFixtureURLsExcludePasswords(t *testing.T) {
	// Credentialed example URLs trigger the commit secret scanner even when the
	// password belongs only to a disposable database. Use runtime credentials.
	urls := regexp.MustCompile("[A-Za-z][A-Za-z0-9+.-]*://[^\\s\"'`]+")
	for _, path := range []string{"testdata/generate_legacy_schema.go", "testdata/README.md"} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			for _, raw := range urls.FindAllString(string(data), -1) {
				parsed, err := url.Parse(raw)
				require.NoError(t, err)
				if parsed.User != nil {
					_, hasPassword := parsed.User.Password()
					require.False(t, hasPassword, "fixture URLs must obtain passwords at runtime")
				}
			}
		})
	}
}

func TestFixtureProvenance(t *testing.T) {
	var manifest struct {
		SHA256 map[string]string `json:"sha256"`
	}
	readFixture(t, "manifest.json", &manifest)
	files := []string{}
	for path := range manifest.SHA256 {
		files = append(files, path)
	}
	require.ElementsMatch(t, []string{"legacy_source.json", "runtime_source.json", "expected_native.json", "merge_contract.json", "legacy_workflow.yaml", "generate_legacy.go", "generate_legacy_schema.go", "regenerate_catalogs.sh", "runtime_fixture_driver_test.go", "runtime_fixture_importer_test.go", "export_runtime.go", "legacy_schema_mysql.json", "legacy_schema_pgx.json", "legacy_partial_mysql.json", "legacy_partial_pgx.json", "identity_vectors.json"}, files)
	for path, expected := range manifest.SHA256 {
		data, err := os.ReadFile("testdata/" + path)
		require.NoError(t, err)
		digest := sha256.Sum256(data)
		require.Equal(t, expected, hex.EncodeToString(digest[:]), path)
	}
	var source sourceFixture
	readFixture(t, "legacy_source.json", &source)
	require.Equal(t, "a0733cb51b0484a6050cafee989ffe3312f808ad", source.SourceCommit)
}

func TestMigrationIdentityVectors(t *testing.T) {
	var vectors []struct {
		Kind        string `json:"kind"`
		SourceID    string `json:"source_id"`
		Metric      string `json:"metric"`
		ArtifactID  string `json:"artifact_id"`
		TaskID      string `json:"task_id"`
		IOType      int32  `json:"io_type"`
		Iteration   int64  `json:"iteration"`
		ArtifactKey string `json:"artifact_key"`
		Name        string `json:"name"`
		UUID        string `json:"uuid"`
	}
	readFixture(t, "identity_vectors.json", &vectors)
	require.Len(t, vectors, 15)
	identities := map[string]bool{}
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			var name string
			if vector.Kind == "link" {
				encoded, err := json.Marshal([]any{"artifact-task", vector.ArtifactID, vector.TaskID, vector.IOType, vector.Iteration, vector.ArtifactKey})
				require.NoError(t, err)
				name = string(encoded)
			} else {
				id, err := strconv.ParseInt(vector.SourceID, 10, 64)
				require.NoError(t, err)
				require.Positive(t, id)
				require.Equal(t, vector.SourceID, strconv.FormatInt(id, 10))
				switch vector.Kind {
				case "metric":
					name = "artifact:" + vector.SourceID + ":metric:" + vector.Metric
				case "metric_primary":
					name = "artifact:" + vector.SourceID
				default:
					name = vector.Kind + ":" + vector.SourceID
				}
			}
			require.Equal(t, vector.Name, name, "UUID name must match the recorded vector")
			id := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(name))
			require.Equal(t, vector.UUID, id.String())
			require.Equal(t, uuid.Version(5), id.Version())
			require.False(t, identities[id.String()], "different source entities/ports/iterations must not alias")
			identities[id.String()] = true
		})
	}
}

func TestLegacyWriterSkippedAndCachedStates(t *testing.T) {
	var source sourceFixture
	readFixture(t, "legacy_source.json", &source)
	require.Len(t, source.Executions.Executions, 14)
	states := map[string]string{}
	for _, execution := range source.Executions.Executions {
		states[execution.Name] = execution.State
		if execution.Name == "cached" {
			require.JSONEq(t, `{"stringValue":"5"}`, string(execution.CustomProperties["cached_execution_id"]))
		}
	}
	require.Equal(t, "CANCELED", states["skipped"], "the old writer uses CANCELED for NotTriggered")
	require.Equal(t, "CACHED", states["cached"])
	require.Equal(t, "FAILED", states["failed"])

	var native nativeFixture
	readFixture(t, "expected_native.json", &native)
	for _, task := range native.Tasks {
		if task.Name == "skipped" {
			require.Equal(t, model.TaskStatus(apiv2.PipelineTask_FAILED), task.State, "the stored source record does not explain its CANCELED state")
		}
		if task.Name == "cached" {
			require.Equal(t, model.TaskStatus(apiv2.PipelineTask_CACHED), task.State)
		}
	}
}

func TestLegacyRuntimeFixture(t *testing.T) {
	var fixture struct {
		SourceCommit string `json:"source_commit"`
		RunUUID      string `json:"run_uuid"`
		Executions   struct {
			Executions []struct {
				ID         string                     `json:"id"`
				Name       string                     `json:"name"`
				State      string                     `json:"lastKnownState"`
				Properties map[string]json.RawMessage `json:"customProperties"`
			} `json:"executions"`
		} `json:"executions"`
		Artifacts struct {
			Artifacts []struct {
				ID string `json:"id"`
			} `json:"artifacts"`
		} `json:"artifacts"`
		Events struct {
			Events []struct {
				ArtifactID  string `json:"artifactId"`
				ExecutionID string `json:"executionId"`
				Type        string `json:"type"`
				Path        struct {
					Steps []struct {
						Key string `json:"key"`
					} `json:"steps"`
				} `json:"path"`
			} `json:"events"`
		} `json:"events"`
	}
	readFixture(t, "runtime_source.json", &fixture)
	require.Equal(t, "a0733cb51b0484a6050cafee989ffe3312f808ad", fixture.SourceCommit)
	require.Len(t, fixture.Executions.Executions, 3)
	root, skipped, importer := fixture.Executions.Executions[0], fixture.Executions.Executions[1], fixture.Executions.Executions[2]
	require.Equal(t, "run/"+fixture.RunUUID, root.Name, "the pinned root driver chooses the execution name")
	require.Equal(t, "RUNNING", root.State)
	require.Empty(t, skipped.Name)
	require.Equal(t, "CANCELED", skipped.State)
	require.JSONEq(t, `{"stringValue":"skipped"}`, string(skipped.Properties["task_name"]))
	require.JSONEq(t, `{"intValue":"1"}`, string(skipped.Properties["parent_dag_id"]))
	require.Empty(t, importer.Name)
	require.Equal(t, "COMPLETE", importer.State)
	require.JSONEq(t, `{"stringValue":"importer"}`, string(importer.Properties["task_name"]))
	require.Len(t, fixture.Artifacts.Artifacts, 1)
	require.Len(t, fixture.Events.Events, 1)
	require.Equal(t, importer.ID, fixture.Events.Events[0].ExecutionID)
	require.Equal(t, fixture.Artifacts.Artifacts[0].ID, fixture.Events.Events[0].ArtifactID)
	require.Equal(t, "OUTPUT", fixture.Events.Events[0].Type)
	require.Len(t, fixture.Events.Events[0].Path.Steps, 1)
	require.Equal(t, "dataset", fixture.Events.Events[0].Path.Steps[0].Key)
}

func TestLegacyTaskMergeContract(t *testing.T) {
	var source sourceFixture
	readFixture(t, "legacy_source.json", &source)
	var fixture struct {
		SourceExecutionID int64 `json:"source_execution_id"`
		LegacyTask        struct {
			RunUUID           string `json:"run_uuid"`
			Namespace         string `json:"namespace"`
			PodName           string `json:"pod_name"`
			State             string `json:"state"`
			Fingerprint       string `json:"fingerprint"`
			CreatedTimestamp  int64  `json:"created_timestamp"`
			StartedTimestamp  int64  `json:"started_timestamp"`
			FinishedTimestamp int64  `json:"finished_timestamp"`
		} `json:"legacy_task"`
		Expected struct {
			TaskUUID               string `json:"task_uuid"`
			RunUUID                string `json:"run_uuid"`
			Namespace              string `json:"namespace"`
			PodName                string `json:"pod_name"`
			State                  string `json:"state"`
			Fingerprint            string `json:"fingerprint"`
			LegacyCacheFingerprint string `json:"legacy_cache_fingerprint"`
			CreatedAtInSec         int64  `json:"created_at_in_sec"`
			StartedInSec           int64  `json:"started_in_sec"`
			FinishedInSec          int64  `json:"finished_in_sec"`
		} `json:"expected_native"`
		StateEvidenceCases []struct {
			MLMDState           string `json:"mlmd_state"`
			WorkflowNodePhase   string `json:"workflow_node_phase"`
			ExpectedNativeState string `json:"expected_native_state"`
		} `json:"state_evidence_cases"`
	}
	readFixture(t, "merge_contract.json", &fixture)
	require.Equal(t, source.Cases["consumer"], fixture.SourceExecutionID)
	require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("execution:"+strconv.FormatInt(fixture.SourceExecutionID, 10))).String(), fixture.Expected.TaskUUID)
	require.Equal(t, fixture.LegacyTask.RunUUID, fixture.Expected.RunUUID)
	require.Equal(t, fixture.LegacyTask.Namespace, fixture.Expected.Namespace)
	require.Equal(t, fixture.LegacyTask.PodName, fixture.Expected.PodName)
	require.Equal(t, fixture.LegacyTask.CreatedTimestamp, fixture.Expected.CreatedAtInSec)
	require.Equal(t, fixture.LegacyTask.StartedTimestamp, fixture.Expected.StartedInSec)
	require.Equal(t, fixture.LegacyTask.FinishedTimestamp, fixture.Expected.FinishedInSec)
	require.Equal(t, fixture.LegacyTask.State, fixture.Expected.State)
	require.Equal(t, source.Cache.Fingerprint, fixture.LegacyTask.Fingerprint)
	require.Equal(t, fixture.LegacyTask.Fingerprint, fixture.Expected.LegacyCacheFingerprint)
	require.Empty(t, fixture.Expected.Fingerprint)
	require.Len(t, fixture.StateEvidenceCases, 2)
	require.Equal(t, "Omitted", fixture.StateEvidenceCases[0].WorkflowNodePhase)
	require.Equal(t, "SKIPPED", fixture.StateEvidenceCases[0].ExpectedNativeState)
	require.Empty(t, fixture.StateEvidenceCases[1].WorkflowNodePhase)
	require.Equal(t, "FAILED", fixture.StateEvidenceCases[1].ExpectedNativeState)
}

func TestExpectedNativeGraph(t *testing.T) {
	var source sourceFixture
	var native nativeFixture
	readFixture(t, "legacy_source.json", &source)
	readFixture(t, "expected_native.json", &native)
	require.Len(t, native.Tasks, 12)
	require.Len(t, native.Artifacts, 4)
	require.Len(t, native.Links, 11)
	require.Equal(t, []int64{source.Cases["gap"], source.Cases["orphan"]}, native.SkippedExecutions)
	require.Less(t, source.Cases["failed"], source.Cases["gap"])
	require.Equal(t, source.Cases["gap"]+1, source.Cases["late"], "an unowned execution separates retained IDs")
	tasks := map[string]model.Task{}
	tasksByName := map[string]model.Task{}
	for _, task := range native.Tasks {
		require.NotContains(t, tasks, task.UUID)
		id, ok := source.Cases[task.Name]
		require.True(t, ok, task.Name)
		require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("execution:"+strconv.FormatInt(id, 10))).String(), task.UUID)
		require.Contains(t, apiv2.PipelineTask_TaskType_name, int32(task.Type))
		require.Contains(t, apiv2.PipelineTask_TaskState_name, int32(task.State))
		require.Empty(t, task.Fingerprint, "historical hashes must not enter native cache lookup without an equivalent native key")
		tasks[task.UUID] = task
		tasksByName[task.Name] = task
	}
	loopAttrsChecked := 0
	for _, execution := range source.Executions.Executions {
		var sourceKey, nativeKey string
		switch execution.Name {
		case "loop":
			sourceKey, nativeKey = "iteration_count", "iterationCount"
		case "iteration-0", "iteration-1":
			sourceKey, nativeKey = "iteration_index", "iterationIndex"
		default:
			continue
		}
		var property struct {
			Value string `json:"intValue"`
		}
		raw, present := execution.CustomProperties[sourceKey]
		require.True(t, present, "%s must have source %s", execution.Name, sourceKey)
		require.NoError(t, json.Unmarshal(raw, &property))
		attribute, present := tasksByName[execution.Name].TypeAttrs[nativeKey]
		require.True(t, present, "%s must retain %s, including zero", execution.Name, nativeKey)
		require.Equal(t, property.Value, attribute, execution.Name)
		loopAttrsChecked++
	}
	require.Equal(t, 3, loopAttrsChecked)
	for _, task := range native.Tasks {
		seen := map[string]bool{task.UUID: true}
		for parent := task.ParentTaskUUID; parent != nil; parent = tasks[*parent].ParentTaskUUID {
			require.Contains(t, tasks, *parent)
			require.False(t, seen[*parent], "parent cycle")
			seen[*parent] = true
			require.Equal(t, task.RunUUID, tasks[*parent].RunUUID)
		}
	}
	artifacts := map[string]model.Artifact{}
	for _, artifact := range native.Artifacts {
		require.NotContains(t, artifacts, artifact.UUID)
		require.NotNil(t, artifact.URI)
		hash := sha256.Sum256([]byte(*artifact.URI))
		require.Equal(t, hex.EncodeToString(hash[:]), artifact.URIHash)
		artifacts[artifact.UUID] = artifact
	}
	require.Contains(t, artifacts, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:2")).String(), "the design assigns the primary metric the source artifact identity")
	require.Contains(t, artifacts, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:3")).String(), "an empty metrics artifact retains its metadata-only row")
	require.Equal(t, source.Cache.Fingerprint, native.LegacyProvenance[tasksByName["consumer"].UUID]["cache_fingerprint"])
	require.Equal(t, "CANCELED", native.LegacyProvenance[tasksByName["skipped"].UUID]["mlmd_state"])
	keys := map[string]bool{}
	metrics := map[string]map[string]bool{}
	for _, link := range native.Links {
		require.Contains(t, tasks, link.TaskID)
		require.Contains(t, artifacts, link.ArtifactID)
		task, artifact := tasks[link.TaskID], artifacts[link.ArtifactID]
		require.Equal(t, task.RunUUID, link.RunUUID)
		require.Equal(t, task.Namespace, artifact.Namespace)
		name, err := json.Marshal([]any{"artifact-task", link.ArtifactID, link.TaskID, int32(link.Type), link.Iteration, link.ArtifactKey})
		require.NoError(t, err)
		require.False(t, keys[string(name)], "duplicate native relationship")
		keys[string(name)] = true
		require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, name).String(), link.UUID)
		if link.ArtifactKey == "metrics" {
			if metrics[task.Name] == nil {
				metrics[task.Name] = map[string]bool{}
			}
			metrics[task.Name][artifact.Name] = true
		}
	}
	for _, task := range []string{"producer", "cached", "consumer"} {
		require.Equal(t, map[string]bool{"accuracy": true, "loss": true}, metrics[task], "all measurements must remain visible through task history")
	}
}

func TestLegacyCacheFingerprintCompatibility(t *testing.T) {
	var source sourceFixture
	readFixture(t, "legacy_source.json", &source)
	inputs := &pipelinespec.ExecutorInput_Inputs{}
	outputs := &pipelinespec.ExecutorInput_Outputs{}
	legacyKey := &cachekey.CacheKey{}
	require.NoError(t, protojson.Unmarshal(source.Cache.Inputs, inputs))
	require.NoError(t, protojson.Unmarshal(source.Cache.Outputs, outputs))
	require.NoError(t, protojson.Unmarshal(source.Cache.Key, legacyKey))
	for _, execution := range source.Executions.Executions {
		if execution.Name == "consumer" {
			stored, err := json.Marshal(map[string]string{"stringValue": source.Cache.Fingerprint})
			require.NoError(t, err)
			require.JSONEq(t, string(stored), string(execution.CustomProperties["cache_fingerprint"]))
			require.JSONEq(t, `{"structValue":{"epochs":3}}`, string(execution.CustomProperties["inputs"]))
		}
	}
	fingerprint, err := cacheutils.GenerateFingerPrint(legacyKey)
	require.NoError(t, err)
	require.Equal(t, source.Cache.Fingerprint, fingerprint)
	for _, scenario := range []string{"unchanged", "native artifact identity", "pipeline identity", "environment"} {
		t.Run(scenario, func(t *testing.T) {
			in := proto.Clone(inputs).(*pipelinespec.ExecutorInput_Inputs)
			command := append([]string{}, source.Cache.Command...)
			var env []*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_EnvVar
			switch scenario {
			case "native artifact identity":
				in.Artifacts["dataset"].Artifacts[0].ArtifactId = uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:1")).String()
			case "pipeline identity":
				command = append(command, "__kfp_pipeline_name=migration-contract")
			case "environment":
				env = []*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_EnvVar{{Name: "TRAIN_MODE", Value: "strict"}}
			}
			key, err := cacheutils.GenerateCacheKey(in, outputs, nil, command, source.Cache.Image, nil, env)
			require.NoError(t, err)
			got, err := cacheutils.GenerateFingerPrint(key)
			require.NoError(t, err)
			if scenario == "unchanged" {
				require.Equal(t, fingerprint, got)
			} else {
				require.NotEqual(t, fingerprint, got)
			}
		})
	}
}

func TestLegacyRetryRetainsRuntime(t *testing.T) {
	data, err := os.ReadFile("testdata/legacy_workflow.yaml")
	require.NoError(t, err)
	require.Contains(t, string(data), "--mlmd_server_address")
	require.Contains(t, string(data), "--dag_execution_id")
	workflow, err := util.NewWorkflowFromBytes(data)
	require.NoError(t, err)
	workflow.Name = "legacy-run"
	workflow.Labels = map[string]string{}
	workflow.Status.Phase = workflowapi.WorkflowFailed
	value := workflowapi.AnyString("5")
	workflow.Status.Nodes = workflowapi.Nodes{"driver": {ID: "driver", Name: "legacy-run.driver", Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeSucceeded, Outputs: &workflowapi.Outputs{Parameters: []workflowapi.Parameter{{Name: "execution-id", Value: &value}}}}}
	before := workflow.DeepCopy()
	retry, _, err := workflow.GenerateRetryExecution()
	require.NoError(t, err)
	retried, ok := retry.(*util.Workflow)
	require.True(t, ok)
	require.Equal(t, before.Spec, retried.Spec, "retry preserves legacy images, MLMD endpoint and execution arguments")
	require.Equal(t, before.Status.Nodes["driver"].Outputs, retried.Status.Nodes["driver"].Outputs)
	require.Equal(t, workflowapi.WorkflowRunning, retried.Status.Phase)
	require.Equal(t, workflowapi.WorkflowFailed, workflow.Status.Phase, "retry must not mutate the original")
}
