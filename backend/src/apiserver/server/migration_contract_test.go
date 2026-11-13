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

package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type migrationProjection struct {
	Tasks            []model.Task                 `json:"tasks"`
	Artifacts        []model.Artifact             `json:"artifacts"`
	Links            []model.ArtifactTask         `json:"links"`
	LegacyProvenance map[string]map[string]string `json:"legacy_provenance"`
}

func seedMigrationProjection(t *testing.T, path string) (migrationProjection, *storage.TaskStore, *gorm.DB) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var fixture migrationProjection
	require.NoError(t, json.Unmarshal(data, &fixture))
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "native.db")+"?_foreign_keys=on"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(model.AllModels()...))
	create := func(value any) { t.Helper(); require.NoError(t, db.Omit(clause.Associations).Create(value).Error) }
	require.NotEmpty(t, fixture.Tasks)
	namespace := fixture.Tasks[0].Namespace
	create(&model.Experiment{UUID: "fixture-experiment", Name: "Default", Namespace: namespace})
	runs := map[string]bool{}
	for _, task := range fixture.Tasks {
		require.Equal(t, namespace, task.Namespace)
		if !runs[task.RunUUID] {
			create(&model.Run{UUID: task.RunUUID, ExperimentId: "fixture-experiment", Namespace: namespace})
			runs[task.RunUUID] = true
		}
	}
	jsonValue := func(value any) string {
		t.Helper()
		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		return string(encoded)
	}
	// Administrative seeding preserves IDs and serializes JSON explicitly, like
	// the history importer. Runtime create methods allocate new identities.
	for _, task := range fixture.Tasks {
		require.NoError(t, db.Model(&model.Task{}).Create(map[string]any{
			"UUID": task.UUID, "RunUUID": task.RunUUID, "Namespace": task.Namespace,
			"Name": task.Name, "DisplayName": task.DisplayName, "Type": task.Type,
			"State": task.State, "ParentTaskUUID": task.ParentTaskUUID,
			"Fingerprint": task.Fingerprint, "CreatedAtInSec": task.CreatedAtInSec,
			"StartedInSec": task.StartedInSec, "FinishedInSec": task.FinishedInSec,
			"pods": jsonValue(task.Pods), "TypeAttrs": jsonValue(task.TypeAttrs),
			"StatusMetadata":   jsonValue(task.StatusMetadata),
			"InputParameters":  jsonValue(task.InputParameters),
			"OutputParameters": jsonValue(task.OutputParameters),
		}).Error)
	}
	for _, artifact := range fixture.Artifacts {
		require.NoError(t, db.Model(&model.Artifact{}).Create(map[string]any{
			"UUID": artifact.UUID, "Namespace": artifact.Namespace, "Name": artifact.Name,
			"Type": artifact.Type, "URI": artifact.URI, "URIHash": artifact.URIHash,
			"NumberValue": artifact.NumberValue, "Metadata": jsonValue(artifact.Metadata),
			"CreatedAtInSec": artifact.CreatedAtInSec, "LastUpdateInSec": artifact.LastUpdateInSec,
		}).Error)
	}
	for _, link := range fixture.Links {
		require.NoError(t, db.Model(&model.ArtifactTask{}).Create(map[string]any{
			"UUID": link.UUID, "ArtifactID": link.ArtifactID,
			"TaskID": link.TaskID, "RunUUID": link.RunUUID,
			"Type": link.Type, "Iteration": link.Iteration,
			"ArtifactKey": link.ArtifactKey, "Producer": jsonValue(link.Producer),
		}).Error)
	}
	return fixture, storage.NewTaskStore(sqlDB, util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), dialect.NewDBDialect("sqlite")), db
}

// This checks the PR 1 native projection through current stores/API. It does
// not exercise a source transform or upgrade DDL.
func TestMigrationContractHistoryProjection(t *testing.T) {
	fixture, store, db := seedMigrationProjection(t, "../migration/mlmd/testdata/expected_native.json")
	sourceData, err := os.ReadFile("../migration/mlmd/testdata/legacy_source.json")
	require.NoError(t, err)
	var source struct {
		LegacyTask struct {
			PodName  string `json:"PodName"`
			State    string `json:"State"`
			Created  int64  `json:"CreatedTimestamp"`
			Finished int64  `json:"FinishedTimestamp"`
		} `json:"legacy_task"`
	}
	require.NoError(t, json.Unmarshal(sourceData, &source))
	tasks := map[string]*apiv2.PipelineTask{}
	for _, expected := range fixture.Tasks {
		task, err := store.GetTask(expected.UUID)
		require.NoError(t, err)
		apiTask, err := toAPITask(task, nil)
		require.NoError(t, err)
		tasks[expected.Name] = apiTask
		require.Equal(t, expected.UUID, apiTask.GetTaskId())
		require.Equal(t, apiv2.PipelineTask_TaskState(expected.State), apiTask.GetState())
		require.Equal(t, apiv2.PipelineTask_TaskType(expected.Type), apiTask.GetType())
		if expected.ParentTaskUUID != nil {
			require.Equal(t, *expected.ParentTaskUUID, apiTask.GetParentTaskId())
		}
	}
	require.Equal(t, apiv2.PipelineTask_FAILED, tasks["skipped"].GetState(), "the source record has no persisted NotTriggered evidence")
	require.Equal(t, apiv2.PipelineTask_CACHED, tasks["cached"].GetState())
	require.Equal(t, "SUCCEEDED", source.LegacyTask.State)
	require.Equal(t, apiv2.PipelineTask_SUCCEEDED, tasks["consumer"].GetState())
	require.Equal(t, source.LegacyTask.PodName, tasks["consumer"].GetPods()[0].GetName())
	require.Equal(t, source.LegacyTask.Created, tasks["consumer"].GetCreateTime().GetSeconds())
	require.Equal(t, source.LegacyTask.Finished, tasks["consumer"].GetEndTime().GetSeconds())
	require.Equal(t, tasks["producer"].GetTaskId(), tasks["cached"].GetStatusMetadata().GetCustomProperties()["cached_task_id"].GetStringValue())
	legacyHash := fixture.LegacyProvenance[tasks["consumer"].GetTaskId()]["cache_fingerprint"]
	require.NotEmpty(t, legacyHash)
	cacheHit, err := store.FindLatestCachedTask("migration-fixture", legacyHash)
	require.NoError(t, err)
	require.NotNil(t, cacheHit)
	require.Equal(t, tasks["consumer"].GetTaskId(), cacheHit.UUID)
	require.Len(t, tasks["producer"].GetPods(), 1)
	pod := tasks["producer"].GetPods()[0]
	require.Equal(t, "producer-pod", pod.GetName())
	require.Equal(t, "producer-uid", pod.GetUid())
	require.Equal(t, apiv2.PipelineTask_EXECUTOR, pod.GetType())
	require.Equal(t, int64(2), tasks["loop"].GetTypeAttributes().GetIterationCount())
	require.NotNil(t, tasks["iteration-0"].GetTypeAttributes().IterationIndex, "iteration zero must survive as a present value")
	require.Zero(t, tasks["iteration-0"].GetTypeAttributes().GetIterationIndex())
	require.Equal(t, int64(1), tasks["iteration-1"].GetTypeAttributes().GetIterationIndex())
	for _, name := range []string{"producer", "consumer"} {
		parameters := tasks[name].GetInputs().GetParameters()
		require.Len(t, parameters, 1)
		require.Equal(t, "epochs", parameters[0].GetParameterKey())
		require.Equal(t, float64(3), parameters[0].GetValue().GetNumberValue())
		require.Equal(t, apiv2.IOType_RUNTIME_VALUE_INPUT, parameters[0].GetType())
	}
	for name, task := range tasks {
		require.Equal(t, int64(1700000000), task.GetCreateTime().GetSeconds(), name)
		if name == "skipped" {
			continue
		}
		require.Len(t, task.GetOutputs().GetParameters(), 1, name)
		parameter := task.GetOutputs().GetParameters()[0]
		require.Equal(t, "result", parameter.GetParameterKey(), name)
		require.Equal(t, "done", parameter.GetValue().GetStringValue(), name)
	}
	for _, name := range []string{"producer", "cached", "consumer"} {
		io := tasks[name].GetOutputs()
		if name == "consumer" {
			io = tasks[name].GetInputs()
		}
		values := map[string]float64{}
		for _, group := range io.GetArtifacts() {
			if group.GetArtifactKey() != "metrics" {
				continue
			}
			for _, metric := range group.GetArtifacts() {
				values[metric.GetName()] = metric.GetNumberValue()
				require.Equal(t, "keep nonnumeric metadata", metric.GetMetadata()["note"].GetStringValue())
			}
		}
		require.Equal(t, map[string]float64{"accuracy": 0.875}, values, name)
	}
	var empty *apiv2.Artifact
	for _, group := range tasks["producer"].GetOutputs().GetArtifacts() {
		if group.GetArtifactKey() == "empty-metrics" {
			require.Len(t, group.GetArtifacts(), 1)
			empty = group.GetArtifacts()[0]
		}
	}
	require.NotNil(t, empty)
	require.Nil(t, empty.NumberValue, "an empty metrics artifact is not a zero-valued measurement")
	require.Equal(t, "no measurements", empty.GetMetadata()["note"].GetStringValue())

	var retained model.Artifact
	lossID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:2:metric:loss")).String()
	require.NoError(t, db.First(&retained, "UUID = ?", lossID).Error)
	require.Equal(t, "loss", retained.Name)
	require.NotNil(t, retained.NumberValue)
	require.Equal(t, 0.125, *retained.NumberValue)

	t.Run("populated legacy task merge projection", func(t *testing.T) {
		data, err := os.ReadFile("../migration/mlmd/testdata/merge_contract.json")
		require.NoError(t, err)
		var merge struct {
			Expected struct {
				TaskUUID       string `json:"task_uuid"`
				PodName        string `json:"pod_name"`
				StartedInSec   int64  `json:"started_in_sec"`
				FinishedInSec  int64  `json:"finished_in_sec"`
				CreatedAtInSec int64  `json:"created_at_in_sec"`
				Fingerprint    string `json:"fingerprint"`
			} `json:"expected_native"`
		}
		require.NoError(t, json.Unmarshal(data, &merge))
		expected := merge.Expected
		stored, err := store.GetTask(expected.TaskUUID)
		require.NoError(t, err)
		require.Equal(t, expected.StartedInSec, stored.StartedInSec)
		require.Equal(t, expected.FinishedInSec, stored.FinishedInSec)
		apiTask, err := toAPITask(stored, nil)
		require.NoError(t, err)
		require.Len(t, apiTask.GetPods(), 1)
		require.Equal(t, expected.PodName, apiTask.GetPods()[0].GetName())
		require.Equal(t, expected.CreatedAtInSec, apiTask.GetCreateTime().GetSeconds())
		require.Equal(t, expected.FinishedInSec, apiTask.GetEndTime().GetSeconds())
		require.Equal(t, expected.Fingerprint, apiTask.GetCacheFingerprint())
	})
}

// The released pipeline projection uses the same run IDs as its KFP/MLMD SQL
// snapshot. Verify that native stores and API conversion can read that graph.
func TestReleasedMigrationProjectionAPI(t *testing.T) {
	fixture, store, _ := seedMigrationProjection(t, "../migration/mlmd/testdata/release_218/expected_native.json")
	require.Len(t, fixture.Tasks, 10)
	wantRunCounts := map[string]int{
		"2ec7ad3b-d821-4b1d-8355-190393a06ddc": 4,
		"5dedb489-f0c5-496e-b2e3-eaf46a5fcc74": 4,
		"be2ae7cc-da75-4cbb-bdb6-25145b26ae52": 2,
	}
	for runID, want := range wantRunCounts {
		count, err := store.GetTaskCountForRun(runID)
		require.NoError(t, err)
		require.Equal(t, want, count, runID)
	}
	tasks := make(map[string]*apiv2.PipelineTask)
	for _, expected := range fixture.Tasks {
		stored, err := store.GetTask(expected.UUID)
		require.NoError(t, err)
		apiTask, err := toAPITask(stored, nil)
		require.NoError(t, err)
		tasks[expected.UUID] = apiTask
		require.Equal(t, expected.UUID, apiTask.GetTaskId())
		require.Equal(t, expected.RunUUID, apiTask.GetRunId())
		require.Equal(t, apiv2.PipelineTask_TaskType(expected.Type), apiTask.GetType())
		require.Equal(t, apiv2.PipelineTask_TaskState(expected.State), apiTask.GetState())
		require.Equal(t, expected.CreatedAtInSec, apiTask.GetCreateTime().GetSeconds())
		if expected.ParentTaskUUID != nil {
			require.Equal(t, *expected.ParentTaskUUID, apiTask.GetParentTaskId())
		}
	}
	loop := tasks["4edaa889-0b08-5797-8806-e00aa3e621df"]
	iteration := tasks["07d71166-b77e-50a0-880e-2ee731a8f6cf"]
	require.Equal(t, int64(3), loop.GetTypeAttributes().GetIterationCount())
	require.NotNil(t, iteration.GetTypeAttributes().IterationIndex)
	require.Zero(t, iteration.GetTypeAttributes().GetIterationIndex())
	producer := tasks["50c36f8f-7c03-50f1-bce1-c114870d2a8d"]
	cached := tasks["d7ea1d3f-78be-5e29-abee-d2b72a0e0b92"]
	require.Equal(t, apiv2.PipelineTask_CACHED, cached.GetState())
	require.Equal(t, producer.GetTaskId(), cached.GetStatusMetadata().GetCustomProperties()["cached_task_id"].GetStringValue())
	legacyFingerprint := fixture.LegacyProvenance[producer.GetTaskId()]["cache_fingerprint"]
	require.Equal(t, legacyFingerprint, producer.GetCacheFingerprint())
	require.Equal(t, legacyFingerprint, cached.GetCacheFingerprint())
	cacheHit, err := store.FindLatestCachedTask("kubeflow", legacyFingerprint)
	require.NoError(t, err)
	require.NotNil(t, cacheHit)
	require.Equal(t, producer.GetTaskId(), cacheHit.UUID)
	require.Len(t, producer.GetPods(), 1)
	require.Equal(t, "math-pipeline-k948h-system-container-impl-942327361", producer.GetPods()[0].GetName())
	require.Len(t, producer.GetOutputs().GetArtifacts(), 1)
	loopOutput := producer.GetOutputs().GetArtifacts()[0]
	require.Equal(t, "executor-logs", loopOutput.GetArtifactKey())
	require.NotNil(t, loopOutput.GetProducer().Iteration)
	require.Zero(t, loopOutput.GetProducer().GetIteration())
	require.Equal(t, "5c95098f-946b-5f11-83d1-ea1fc346ecd9", loopOutput.GetArtifacts()[0].GetArtifactId())
	metrics := tasks["f2c4a81b-0659-57db-a256-38f83003d939"]
	require.Equal(t, int64(1793737760), metrics.GetEndTime().GetSeconds())
	require.Len(t, metrics.GetPods(), 1)
	require.Equal(t, "output-metrics-59hc2-system-container-impl-2782546728", metrics.GetPods()[0].GetName())
	require.Len(t, metrics.GetOutputs().GetArtifacts(), 1)
	group := metrics.GetOutputs().GetArtifacts()[0]
	require.Equal(t, "metrics", group.GetArtifactKey())
	require.Len(t, group.GetArtifacts(), 1)
	require.Equal(t, "accuracy", group.GetArtifacts()[0].GetName())
	require.Equal(t, float64(4), group.GetArtifacts()[0].GetNumberValue())
}
