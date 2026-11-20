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
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type migrationProjection struct {
	Tasks     []model.Task
	Artifacts []model.Artifact
	Links     []model.ArtifactTask
}

func seedNativeMigrationProjection(t *testing.T, fixture migrationProjection) (*storage.TaskStore, *gorm.DB) {
	t.Helper()
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
		// Some captured MLMD executions have an empty namespace property.
		if task.Namespace != "" {
			require.Equal(t, namespace, task.Namespace)
		}
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
			"State":       task.State,
			"Fingerprint": task.Fingerprint, "CreatedAtInSec": task.CreatedAtInSec,
			"StartedInSec": task.StartedInSec, "FinishedInSec": task.FinishedInSec,
			"pods": jsonValue(task.Pods), "TypeAttrs": jsonValue(task.TypeAttrs),
			"StatusMetadata":   jsonValue(task.StatusMetadata),
			"InputParameters":  jsonValue(task.InputParameters),
			"OutputParameters": jsonValue(task.OutputParameters),
		}).Error)
	}
	// Resolve parent links after inserting all tasks, as in the migration.
	for _, task := range fixture.Tasks {
		if task.ParentTaskUUID != nil {
			require.NoError(t, db.Model(&model.Task{}).Where("UUID = ?", task.UUID).
				Update("ParentTaskUUID", *task.ParentTaskUUID).Error)
		}
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
	return storage.NewTaskStore(sqlDB, util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), dialect.NewDBDialect("sqlite")), db
}

// Verify the released fixture's native task inventory through the current store
// and API converter. The source workflow reports are accounted for separately.
func TestReleasedMigrationProjectionAPI(t *testing.T) {
	data, err := os.ReadFile("../migration/mlmd/testdata/release_218/migration_fixture.json")
	require.NoError(t, err)
	var released struct {
		Expected struct {
			Tasks []struct {
				Fields model.Task `json:"fields"`
			} `json:"tasks"`
			Artifacts []struct {
				Fields model.Artifact `json:"fields"`
			} `json:"artifacts"`
			ArtifactTasks []struct {
				Fields model.ArtifactTask `json:"fields"`
			} `json:"artifact_tasks"`
			API []struct {
				RunID   string   `json:"run_id"`
				TaskIDs []string `json:"task_ids"`
			} `json:"api"`
		} `json:"expected"`
	}
	require.NoError(t, json.Unmarshal(data, &released))
	var projection migrationProjection
	for _, task := range released.Expected.Tasks {
		projection.Tasks = append(projection.Tasks, task.Fields)
	}
	for _, artifact := range released.Expected.Artifacts {
		projection.Artifacts = append(projection.Artifacts, artifact.Fields)
	}
	for _, link := range released.Expected.ArtifactTasks {
		link.Fields.Iteration = model.ArtifactTaskNoIteration
		projection.Links = append(projection.Links, link.Fields)
	}
	require.Len(t, projection.Tasks, 34)
	store, db := seedNativeMigrationProjection(t, projection)
	apiTaskIDs := map[string][]string{}
	apiTasks := map[string]*apiv2.PipelineTask{}
	for _, expected := range projection.Tasks {
		stored, err := store.GetTask(expected.UUID)
		require.NoError(t, err)
		apiTask, err := toAPITask(stored, nil)
		require.NoError(t, err)
		require.Equal(t, expected.UUID, apiTask.GetTaskId())
		require.Equal(t, expected.RunUUID, apiTask.GetRunId())
		require.Equal(t, apiv2.PipelineTask_TaskType(expected.Type), apiTask.GetType())
		require.Equal(t, apiv2.PipelineTask_TaskState(expected.State), apiTask.GetState())
		require.Equal(t, expected.Fingerprint, apiTask.GetCacheFingerprint())
		require.Equal(t, expected.ParentTaskUUID, apiTask.ParentTaskId)
		apiTasks[apiTask.GetTaskId()] = apiTask
		apiTaskIDs[apiTask.GetRunId()] = append(apiTaskIDs[apiTask.GetRunId()], apiTask.GetTaskId())
	}
	findTask := func(mlmdID string) *apiv2.PipelineTask {
		t.Helper()
		task := apiTasks[uuid.NewSHA1(uuid.NameSpaceDNS, []byte("execution:"+mlmdID)).String()]
		require.NotNil(t, task)
		return task
	}
	outputs := findTask("2").GetOutputs().GetParameters()
	require.Len(t, outputs, 1)
	require.Equal(t, "Output", outputs[0].GetParameterKey())
	require.Equal(t, "fixture-produced", outputs[0].GetValue().GetStringValue())
	require.Equal(t, apiv2.IOType_OUTPUT, outputs[0].GetType())
	require.Equal(t, "produce", outputs[0].GetProducer().GetTaskName())

	// The captured cache hit reuses the original execution and its primary metric.
	cached := findTask("7")
	require.Equal(t, apiv2.PipelineTask_CACHED, cached.GetState())
	require.Equal(t, findTask("2").GetTaskId(), cached.GetStatusMetadata().GetCustomProperties()["cached_task_id"].GetStringValue())
	for _, task := range []*apiv2.PipelineTask{findTask("2"), cached} {
		var metrics []*apiv2.Artifact
		for _, group := range task.GetOutputs().GetArtifacts() {
			if group.GetArtifactKey() == "metrics" {
				metrics = append(metrics, group.GetArtifacts()...)
			}
		}
		require.Len(t, metrics, 1)
		require.Equal(t, "accuracy", metrics[0].GetName())
		require.Equal(t, 0.875, metrics[0].GetNumberValue())
	}
	var secondary model.Artifact
	lossID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:3:metric:loss")).String()
	require.NoError(t, db.First(&secondary, "UUID = ?", lossID).Error)
	require.Equal(t, "loss", secondary.Name)
	require.NotNil(t, secondary.NumberValue)
	require.Equal(t, 0.125, *secondary.NumberValue)

	inputs := findTask("4").GetInputs().GetParameters()
	require.Len(t, inputs, 1)
	require.Equal(t, "value", inputs[0].GetParameterKey())
	require.Equal(t, "fixture-produced", inputs[0].GetValue().GetStringValue())
	require.Equal(t, apiv2.IOType_TASK_OUTPUT_INPUT, inputs[0].GetType())
	require.Equal(t, "produce", inputs[0].GetProducer().GetTaskName())

	loop := findTask("23")
	require.Equal(t, int64(2), loop.GetTypeAttributes().GetIterationCount())
	rawLoopInputs := loop.GetInputs().GetParameters()
	require.Len(t, rawLoopInputs, 1)
	require.Equal(t, apiv2.IOType_UNSPECIFIED, rawLoopInputs[0].GetType())
	require.Nil(t, rawLoopInputs[0].GetProducer())
	require.Equal(t, []any{"one", "two"}, rawLoopInputs[0].GetValue().GetListValue().AsSlice())

	iteration := findTask("25")
	attrs := iteration.GetTypeAttributes()
	require.NotNil(t, attrs)
	require.NotNil(t, attrs.IterationIndex)
	require.Zero(t, attrs.GetIterationIndex())
	iterationInputs := iteration.GetInputs().GetParameters()
	require.Len(t, iterationInputs, 1)
	require.IsType(t, &structpb.Value_StringValue{}, iterationInputs[0].GetValue().GetKind())
	require.Equal(t, "one", iterationInputs[0].GetValue().GetStringValue())
	require.Equal(t, apiv2.IOType_ITERATOR_INPUT, iterationInputs[0].GetType())
	require.Equal(t, "for-loop-3", iterationInputs[0].GetProducer().GetTaskName())
	require.NotNil(t, iterationInputs[0].GetProducer().Iteration)
	require.Zero(t, iterationInputs[0].GetProducer().GetIteration())

	for _, run := range released.Expected.API {
		count, err := store.GetTaskCountForRun(run.RunID)
		require.NoError(t, err)
		require.Equal(t, len(run.TaskIDs), count, run.RunID)
		require.ElementsMatch(t, run.TaskIDs, apiTaskIDs[run.RunID], run.RunID)
		delete(apiTaskIDs, run.RunID)
	}
	require.Empty(t, apiTaskIDs)
}
