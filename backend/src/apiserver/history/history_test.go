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

package history

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func database(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "history.db")+"?_foreign_keys=on"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(model.AllModels()...))
	return db
}

func create(t *testing.T, db *gorm.DB, value any) {
	t.Helper()
	require.NoError(t, createRecord(db, value))
}

func fixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	create(t, db, &model.Experiment{UUID: "experiment", Name: "Default", Namespace: "team", LastRunCreatedAtInSec: 100})
	create(t, db, &model.Pipeline{UUID: "pipeline", Name: "training", Namespace: "team", DefaultVersionId: "version"})
	create(t, db, &model.PipelineVersion{UUID: "version", Name: "v1", PipelineId: "pipeline", PipelineSpec: "{\"pipelineInfo\":{\"name\":\"training\"}}"})
	create(t, db, &model.PipelineTag{PipelineID: "pipeline", TagKey: "owner", TagValue: "source"})
	run := model.Run{UUID: "run", DisplayName: "completed training", K8SName: "old-workflow", Namespace: "team", ExperimentId: "experiment", RecurringRunId: "old-schedule", StorageState: model.StorageStateAvailable,
		PipelineSpec: model.PipelineSpec{PipelineId: "pipeline", PipelineVersionId: "version", PipelineSpecManifest: "{}"},
		RunDetails:   model.RunDetails{CreatedAtInSec: 100, FinishedAtInSec: 300, State: model.RuntimeStateSucceeded, Conditions: "Succeeded", WorkflowRuntimeManifest: "{\"metadata\":{\"name\":\"old-workflow\"}}"}}
	create(t, db, &run)
	parent := "z-parent"
	create(t, db, &model.Task{UUID: parent, Namespace: "team", RunUUID: "run", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}, State: model.TaskStatus(api.PipelineTask_SUCCEEDED), Fingerprint: "cache-key"})
	create(t, db, &model.Task{UUID: "a-child", Namespace: "team", RunUUID: "run", ParentTaskUUID: &parent, Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}, State: model.TaskStatus(api.PipelineTask_SUCCEEDED), Fingerprint: "child-key", OutputParameters: model.JSONSlice{map[string]any{"accuracy": 0.98}}})
	uri := "s3://retained-bucket/output"
	create(t, db, &model.Artifact{UUID: "artifact", Namespace: "team", URI: &uri, Metadata: model.JSONData{"score": 0.98, "large_id": json.Number("9007199254740993")}})
	create(t, db, &model.ArtifactTask{UUID: "link", ArtifactID: "artifact", TaskID: "a-child", RunUUID: "run", ArtifactKey: "model", Iteration: 0})
	create(t, db, &model.RunMetricV1{RunUUID: "run", NodeID: "a-child", Name: "accuracy", NumberValue: 0.98})
}

func exportFixture(t *testing.T, source *gorm.DB) *Bundle {
	t.Helper()
	bundle, err := Export(context.Background(), source, "retired-a", []string{"run"})
	require.NoError(t, err)
	// Exercise the real archive boundary, including JSON numeric values.
	raw, err := json.Marshal(bundle)
	require.NoError(t, err)
	var decoded Bundle
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&decoded))
	return &decoded
}

func count(t *testing.T, db *gorm.DB, value any) int64 {
	t.Helper()
	var result int64
	require.NoError(t, db.Model(value).Count(&result).Error)
	return result
}

func TestImportIntoPopulatedDatabaseAndRepeat(t *testing.T) {
	source, dest := database(t), database(t)
	fixture(t, source)
	create(t, dest, &model.Experiment{UUID: "native-experiment", Name: "Default", Namespace: "team"})
	create(t, dest, &model.Run{UUID: "native-run", Namespace: "team", ExperimentId: "native-experiment", RunDetails: model.RunDetails{State: model.RuntimeStateRunning}})
	bundle := exportFixture(t, source)
	opts := ImportOptions{NamePrefix: "retired-"}
	result, err := Import(context.Background(), dest, bundle, opts)
	require.NoError(t, err)
	require.Equal(t, Result{Imported: 1}, result)
	var run model.Run
	require.NoError(t, dest.First(&run, "UUID = ?", "run").Error)
	require.Equal(t, "retired-a", run.ImportedFrom)
	require.Len(t, run.ImportDigest, 64)
	require.Equal(t, int64(300), run.FinishedAtInSec)
	require.Empty(t, run.RecurringRunId)
	require.Equal(t, model.RuntimeStateSucceeded, run.State)
	require.Equal(t, int64(2), count(t, dest, &model.Run{}))
	require.Equal(t, int64(2), count(t, dest, &model.Task{}))
	require.Equal(t, int64(1), count(t, dest, &model.ArtifactTask{}))
	require.Equal(t, int64(1), count(t, dest, &model.RunMetricV1{}))
	require.Zero(t, count(t, dest, &model.Job{}))
	var native model.Run
	require.NoError(t, dest.First(&native, "UUID = ?", "native-run").Error)
	require.Equal(t, model.RuntimeStateRunning, native.State)
	require.Empty(t, native.ImportedFrom)
	var artifact model.Artifact
	require.NoError(t, dest.First(&artifact).Error)
	require.Equal(t, "s3://retained-bucket/output", *artifact.URI)
	var rawMetadata string
	require.NoError(t, dest.Raw("SELECT Metadata FROM artifacts WHERE UUID = ?", "artifact").Scan(&rawMetadata).Error)
	require.Contains(t, rawMetadata, "9007199254740993")
	var link model.ArtifactTask
	require.NoError(t, dest.First(&link).Error)
	require.Equal(t, int64(0), link.Iteration)
	// Source experiment activity changes while installations coexist. Re-export
	// must not overwrite the destination or spuriously invalidate the run.
	require.NoError(t, source.Model(&model.Experiment{}).Where("UUID = ?", "experiment").Update("LastRunCreatedAtInSec", 500).Error)
	result, err = Import(context.Background(), dest, exportFixture(t, source), opts)
	require.NoError(t, err)
	require.Equal(t, Result{Skipped: 1}, result)
	require.Equal(t, int64(2), count(t, dest, &model.Run{}))
}

func TestImportConflictRollsBackDependencies(t *testing.T) {
	source, dest := database(t), database(t)
	fixture(t, source)
	create(t, dest, &model.Run{UUID: "run", DisplayName: "destination must survive"})
	_, err := Import(context.Background(), dest, exportFixture(t, source), ImportOptions{})
	require.ErrorContains(t, err, "conflicts")
	require.Zero(t, count(t, dest, &model.Experiment{}))
	require.Zero(t, count(t, dest, &model.Pipeline{}))
	require.Zero(t, count(t, dest, &model.Task{}))
	var run model.Run
	require.NoError(t, dest.First(&run).Error)
	require.Equal(t, "destination must survive", run.DisplayName)
}

func TestImportDryRunAndExplicitExperimentMapping(t *testing.T) {
	source, dest := database(t), database(t)
	fixture(t, source)
	create(t, dest, &model.Experiment{UUID: "target", Name: "Default", Namespace: "team"})
	bundle := exportFixture(t, source)
	_, err := Import(context.Background(), dest, bundle, ImportOptions{})
	require.Error(t, err) // Same name does not authorize merging different UUIDs.
	opts := ImportOptions{ExperimentID: "target", DryRun: true}
	result, err := Import(context.Background(), dest, bundle, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Zero(t, count(t, dest, &model.Run{}))
	require.Zero(t, count(t, dest, &model.Pipeline{}))
	opts.DryRun = false
	_, err = Import(context.Background(), dest, bundle, opts)
	require.NoError(t, err)
	var run model.Run
	require.NoError(t, dest.First(&run).Error)
	require.Equal(t, "target", run.ExperimentId)
	result, err = Import(context.Background(), dest, bundle, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Skipped)
}

func TestExportRejectsIncompleteLegacyAndImportedRuns(t *testing.T) {
	for _, update := range []map[string]any{
		{"State": model.RuntimeStateRunning},
		{"FinishedAtInSec": 0},
		{"PipelineRunContextId": 12},
		{"ImportedFrom": "another-source"},
	} {
		source := database(t)
		fixture(t, source)
		require.NoError(t, source.Model(&model.Run{}).Where("UUID = ?", "run").Updates(update).Error)
		_, err := Export(context.Background(), source, "old", []string{"run"})
		require.Error(t, err)
	}
}

func TestRejectInvalidGraphAndSchemaWithoutWrites(t *testing.T) {
	source := database(t)
	fixture(t, source)
	tests := map[string]func(*Bundle){
		"format":           func(b *Bundle) { b.Format = "kfp-mlmd-history/v1" },
		"schema":           func(b *Bundle) { b.Schema = "other" },
		"task owner":       func(b *Bundle) { b.Entries[0].Tasks[0].RunUUID = "other" },
		"artifact link":    func(b *Bundle) { b.Entries[0].Links[0].TaskID = "missing" },
		"task cycle":       func(b *Bundle) { id := b.Entries[0].Tasks[0].UUID; b.Entries[0].Tasks[0].ParentTaskUUID = &id },
		"namespace":        func(b *Bundle) { b.Entries[0].Run.Namespace = "other-team" },
		"missing pipeline": func(b *Bundle) { b.Pipelines = nil },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			dest := database(t)
			bundle := exportFixture(t, source)
			mutate(bundle)
			_, err := Import(context.Background(), dest, bundle, ImportOptions{})
			require.Error(t, err)
			require.Zero(t, count(t, dest, &model.Run{}))
			require.Zero(t, count(t, dest, &model.Experiment{}))
		})
	}
	require.NoError(t, source.Exec("ALTER TABLE run_details ADD COLUMN FutureField TEXT").Error)
	_, err := Export(context.Background(), source, "old", []string{"run"})
	require.ErrorContains(t, err, "schema mismatch")
}

func TestChangedRepeatDoesNotOverwriteImportedRun(t *testing.T) {
	source, dest := database(t), database(t)
	fixture(t, source)
	bundle := exportFixture(t, source)
	_, err := Import(context.Background(), dest, bundle, ImportOptions{})
	require.NoError(t, err)
	bundle.Entries[0].Run.DisplayName = "changed source"
	_, err = Import(context.Background(), dest, bundle, ImportOptions{})
	require.ErrorContains(t, err, "conflicts")
	var run model.Run
	require.NoError(t, dest.First(&run).Error)
	require.Equal(t, "completed training", run.DisplayName)
}

func TestImportHistoryWithDeletedCatalogAndArtifactIdentityCollision(t *testing.T) {
	source, dest := database(t), database(t)
	fixture(t, source)
	key := "shared-external-artifact-identity"
	require.NoError(t, source.Model(&model.Artifact{}).Where("UUID = ?", "artifact").Update("IdentityKey", key).Error)
	create(t, dest, &model.Artifact{UUID: "native-artifact", Namespace: "team", IdentityKey: &key})
	// Deleting a catalog entry does not delete the embedded run specification.
	require.NoError(t, source.Where("UUID = ?", "pipeline").Delete(&model.Pipeline{}).Error)
	bundle := exportFixture(t, source)
	_, err := Import(context.Background(), dest, bundle, ImportOptions{})
	require.NoError(t, err)
	var run model.Run
	require.NoError(t, dest.First(&run).Error)
	require.Empty(t, run.PipelineId)
	require.Empty(t, run.PipelineVersionId)
	require.Equal(t, model.LargeText("{}"), run.PipelineSpecManifest)
	require.Equal(t, int64(2), count(t, dest, &model.Artifact{}))
	var imported model.Artifact
	require.NoError(t, dest.First(&imported, "UUID = ?", "artifact").Error)
	require.Nil(t, imported.IdentityKey)
	repeated, err := Import(context.Background(), dest, bundle, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, repeated.Skipped)
}

func TestExportTerminalRetriedRun(t *testing.T) {
	source := database(t)
	fixture(t, source)
	// The successful retry path retains the claim timestamp as historical data.
	require.NoError(t, source.Model(&model.Run{}).Where("UUID = ?", "run").Updates(map[string]any{"RetryGeneration": 2, "RetryClaimedAtInSec": 200}).Error)
	exportFixture(t, source)
}

func TestImportRejectsArtifactUUIDFromUnrelatedLineage(t *testing.T) {
	for _, ownership := range []struct {
		name    string
		sources []string
	}{
		{name: "unlinked"},
		{name: "native", sources: []string{""}},
		{name: "another source", sources: []string{"another-installation"}},
		{name: "mixed native", sources: []string{"retired-a", ""}},
		{name: "mixed sources", sources: []string{"retired-a", "another-installation"}},
	} {
		t.Run(ownership.name, func(t *testing.T) {
			source, dest := database(t), database(t)
			fixture(t, source)
			bundle := exportFixture(t, source)
			artifact := bundle.Entries[0].Artifacts[0]
			create(t, dest, &artifact) // Identical content must not authorize reuse.
			for i, owner := range ownership.sources {
				id := fmt.Sprintf("destination-%d", i)
				create(t, dest, &model.Run{UUID: id, ImportedFrom: owner, Namespace: "team"})
				create(t, dest, &model.Task{UUID: id, RunUUID: id, Namespace: "team", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
				create(t, dest, &model.ArtifactTask{UUID: id, RunUUID: id, TaskID: id, ArtifactID: artifact.UUID})
			}
			_, err := Import(context.Background(), dest, bundle, ImportOptions{})
			require.ErrorContains(t, err, "conflicts with destination lineage")
			require.Equal(t, int64(len(ownership.sources)), count(t, dest, &model.Run{}))
			require.Equal(t, int64(len(ownership.sources)), count(t, dest, &model.Task{}))
			require.Equal(t, int64(len(ownership.sources)), count(t, dest, &model.ArtifactTask{}))
			require.Equal(t, int64(1), count(t, dest, &model.Artifact{}))
			require.Zero(t, count(t, dest, &model.Experiment{}))
			require.Zero(t, count(t, dest, &model.Pipeline{}))
		})
	}
}

func TestImportSharedArtifactFromSameSource(t *testing.T) {
	for _, together := range []bool{false, true} {
		t.Run(fmt.Sprintf("same-bundle-%t", together), func(t *testing.T) {
			source, dest := database(t), database(t)
			fixture(t, source)
			create(t, source, &model.Run{UUID: "run-2", Namespace: "team", ExperimentId: "experiment", StorageState: model.StorageStateAvailable,
				RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded, FinishedAtInSec: 400}})
			create(t, source, &model.Task{UUID: "task-2", Namespace: "team", RunUUID: "run-2", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
			create(t, source, &model.ArtifactTask{UUID: "link-2", RunUUID: "run-2", TaskID: "task-2", ArtifactID: "artifact"})
			selections := [][]string{{"run"}, {"run-2"}}
			if together {
				selections = [][]string{{"run", "run-2"}}
			}
			for _, ids := range selections {
				bundle, err := Export(context.Background(), source, "retired-a", ids)
				require.NoError(t, err)
				result, err := Import(context.Background(), dest, bundle, ImportOptions{})
				require.NoError(t, err)
				require.Equal(t, len(ids), result.Imported)
				result, err = Import(context.Background(), dest, bundle, ImportOptions{})
				require.NoError(t, err)
				require.Equal(t, len(ids), result.Skipped)
			}
			require.Equal(t, int64(2), count(t, dest, &model.Run{}))
			require.Equal(t, int64(1), count(t, dest, &model.Artifact{}))
			require.Equal(t, int64(2), count(t, dest, &model.ArtifactTask{}))
		})
	}
}
