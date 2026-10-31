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

// This checks the PR 1 native projection through current stores/API. It does
// not exercise a source transform or upgrade DDL.
func TestMigrationContractHistoryProjection(t *testing.T) {
	data, err := os.ReadFile("../migration/mlmd/testdata/expected_native.json")
	require.NoError(t, err)
	var fixture struct {
		Tasks            []model.Task                 `json:"tasks"`
		Artifacts        []model.Artifact             `json:"artifacts"`
		Links            []model.ArtifactTask         `json:"links"`
		LegacyProvenance map[string]map[string]string `json:"legacy_provenance"`
	}
	require.NoError(t, json.Unmarshal(data, &fixture))
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "native.db")+"?_foreign_keys=on"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(model.AllModels()...))
	create := func(value any) { t.Helper(); require.NoError(t, db.Omit(clause.Associations).Create(value).Error) }
	create(&model.Experiment{UUID: "fixture-experiment", Name: "Default", Namespace: "migration-fixture"})
	create(&model.Run{UUID: fixture.Tasks[0].RunUUID, ExperimentId: "fixture-experiment", Namespace: "migration-fixture"})
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
			"pods": jsonValue(task.Pods), "TypeAttrs": jsonValue(task.TypeAttrs),
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
		create(&link)
	}
	store := storage.NewTaskStore(sqlDB, util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), dialect.NewDBDialect("sqlite"))
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
	legacyHash := fixture.LegacyProvenance[tasks["consumer"].GetTaskId()]["cache_fingerprint"]
	require.NotEmpty(t, legacyHash)
	cacheHit, err := store.FindLatestCachedTask("migration-fixture", legacyHash)
	require.NoError(t, err)
	require.Nil(t, cacheHit, "legacy fingerprint provenance must not seed native cache lookup")
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
		require.Equal(t, map[string]float64{"accuracy": 0.875, "loss": 0.125}, values, name)
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
	require.NoError(t, db.First(&retained, "Name = ?", "loss").Error)
	require.Equal(t, "loss", retained.Name)

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
		require.NoError(t, db.Model(&model.Task{}).Where("UUID = ?", expected.TaskUUID).Updates(map[string]any{
			"pods":         jsonValue([]map[string]string{{"name": expected.PodName, "type": "EXECUTOR"}}),
			"StartedInSec": expected.StartedInSec, "FinishedInSec": expected.FinishedInSec,
		}).Error)
		stored, err := store.GetTask(expected.TaskUUID)
		require.NoError(t, err)
		require.Equal(t, expected.StartedInSec, stored.StartedInSec)
		require.Equal(t, expected.FinishedInSec, stored.FinishedInSec)
		apiTask, err := toAPITask(stored, nil)
		require.NoError(t, err)
		require.Equal(t, expected.PodName, apiTask.GetPods()[0].GetName())
		require.Equal(t, expected.CreatedAtInSec, apiTask.GetCreateTime().GetSeconds())
		require.Equal(t, expected.FinishedInSec, apiTask.GetEndTime().GetSeconds())
		require.Equal(t, expected.Fingerprint, apiTask.GetCacheFingerprint())
	})
}
