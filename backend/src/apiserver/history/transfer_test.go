// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func cloneTransfer(t *testing.T, b *NamespaceBundle) *NamespaceBundle {
	t.Helper()
	raw, err := json.Marshal(b)
	require.NoError(t, err)
	var clone NamespaceBundle
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	require.NoError(t, d.Decode(&clone))
	return &clone
}
func namespaceFixture(t *testing.T, source *gorm.DB) *NamespaceBundle {
	t.Helper()
	fixture(t, source)
	require.NoError(t, source.Model(&model.Task{}).Where(clause.Eq{Column: "UUID", Value: "a-child"}).Update("LogicalKey", "source-logical-key").Error)
	create(t, source, &model.Experiment{UUID: "empty", Name: "empty", Namespace: "team"})
	create(t, source, &model.PipelineVersion{UUID: "unused", Name: "unused", PipelineId: "pipeline", PipelineSpec: `{"pipelineInfo":{"name":"unused"}}`})
	create(t, source, &model.Job{UUID: "schedule", DisplayName: "nightly", Namespace: "team", ExperimentId: "experiment", Enabled: true, PipelineSpec: model.PipelineSpec{PipelineId: "pipeline", PipelineVersionId: "version"}})
	var pipelines []model.Pipeline
	var versions []model.PipelineVersion
	require.NoError(t, source.Find(&pipelines).Error)
	require.NoError(t, source.Find(&versions).Error)
	b, err := ExportNamespace(context.Background(), source, "team", "team", transfer.ExportOptions{}, pipelines, versions)
	require.NoError(t, err)
	require.Len(t, b.Experiments, 2)
	require.Len(t, b.Versions, 2)
	require.Len(t, b.Schedules, 1)
	return b
}
func exerciseNamespaceTransfer(t *testing.T, source, dest *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	b := namespaceFixture(t, source)
	create(t, dest, &model.Experiment{UUID: "native", Name: "Default", Namespace: "team"})
	opts := transfer.ImportOptions{NamePrefix: "old-", DryRun: true}
	plan, err := PrepareTransfer(ctx, dest, cloneTransfer(t, b), opts)
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, dest, plan, true))
	require.Zero(t, count(t, dest, &model.TransferReceipt{}))
	require.Zero(t, count(t, dest, &model.Run{}))
	require.Equal(t, int64(1), count(t, dest, &model.Experiment{}))
	require.Zero(t, count(t, dest, &model.TransferIdentity{}), "preview must not write installation metadata")
	opts.DryRun = false
	plan, err = PrepareTransfer(ctx, dest, cloneTransfer(t, b), opts)
	require.NoError(t, err)
	// Simulate Kubernetes assigning a new UID to the staged disabled schedule.
	old := plan.Bundle.Schedules[0].UUID
	plan.ReplaceID("schedule", old, "destination-schedule-uid")
	plan.Bundle.Schedules[0].K8SName = "transfer-staged"
	require.NoError(t, CommitTransfer(ctx, dest, plan, false))
	require.Equal(t, int64(3), count(t, dest, &model.Experiment{}))
	require.Equal(t, int64(2), count(t, dest, &model.PipelineVersion{}))
	require.Equal(t, int64(1), count(t, dest, &model.Job{}))
	var job model.Job
	require.NoError(t, dest.Take(&job).Error)
	require.False(t, job.Enabled)
	require.True(t, job.NoCatchup)
	require.Equal(t, "destination-schedule-uid", job.UUID)
	var state model.RecurringRunState
	require.NoError(t, dest.Take(&state).Error)
	require.Empty(t, state.RequestKey)
	require.False(t, state.Pending)
	require.Zero(t, state.LastRunIndex)
	var run model.Run
	require.NoError(t, dest.Take(&run).Error)
	require.NotEqual(t, "run", run.UUID)
	require.Equal(t, b.Source, run.ImportedFrom)
	var links []model.ArtifactTask
	require.NoError(t, dest.Find(&links).Error)
	require.Len(t, links, 1)
	require.Equal(t, run.UUID, links[0].RunUUID)
	require.Zero(t, links[0].Iteration)
	var tasks []model.Task
	require.NoError(t, dest.Find(&tasks).Error)
	for _, task := range tasks {
		require.Nil(t, task.LogicalKey)
	}
	var artifacts []model.Artifact
	require.NoError(t, readRows(dest, &artifacts))
	raw, err := json.Marshal(artifacts[0].Metadata)
	require.NoError(t, err)
	require.Contains(t, string(raw), "9007199254740993")
	repeat, err := PrepareTransfer(ctx, dest, cloneTransfer(t, b), opts)
	require.NoError(t, err)
	require.Zero(t, repeat.Summary.Imported)
	require.NoError(t, CommitTransfer(ctx, dest, repeat, false))
	require.Equal(t, int64(1), count(t, dest, &model.Run{}))
	changed := cloneTransfer(t, b)
	changed.Entries[0].Run.DisplayName = "changed"
	_, err = PrepareTransfer(ctx, dest, changed, opts)
	require.ErrorContains(t, err, "changed")
	require.Equal(t, int64(1), count(t, dest, &model.Run{}))
}
func TestNamespaceTransferSQLMerge(t *testing.T) {
	exerciseNamespaceTransfer(t, database(t), database(t))
}
func TestNamespaceTransferEmptyAndOwnership(t *testing.T) {
	ctx := context.Background()
	source, dest := database(t), database(t)
	create(t, source, &model.Experiment{UUID: "empty", Name: "empty", Namespace: "team"})
	b, err := ExportNamespace(ctx, source, "team", "team", transfer.ExportOptions{}, nil, nil)
	require.NoError(t, err)
	require.Empty(t, b.Entries)
	p, err := PrepareTransfer(ctx, dest, cloneTransfer(t, b), transfer.ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, dest, p, false))
	require.Equal(t, int64(1), count(t, dest, &model.Experiment{}))
	bad := cloneTransfer(t, b)
	bad.Experiments[0].Namespace = "other"
	require.ErrorContains(t, ValidateNamespace(bad, "team", "team"), "another namespace")
	require.Error(t, ValidateNamespace(b, "other", "other"))
}
func TestNamespaceTransferSingleUserOwnership(t *testing.T) {
	ctx := context.Background()
	source, dest := database(t), database(t)
	fixture(t, source)
	require.NoError(t, source.Model(&model.Experiment{}).Where(clause.Eq{Column: "UUID", Value: "experiment"}).Update("Namespace", "").Error)
	var ps []model.Pipeline
	var vs []model.PipelineVersion
	require.NoError(t, source.Find(&ps).Error)
	require.NoError(t, source.Find(&vs).Error)
	ps[0].Namespace = ""
	b, err := ExportNamespace(ctx, source, "", "team", transfer.ExportOptions{}, ps, vs)
	require.NoError(t, err)
	p, err := PrepareTransfer(ctx, dest, b, transfer.ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, dest, p, false))
	var e model.Experiment
	require.NoError(t, dest.Take(&e).Error)
	require.Empty(t, e.Namespace)
	var r model.Run
	require.NoError(t, dest.Take(&r).Error)
	require.Equal(t, "team", r.Namespace)
}
func TestNamespaceTransferDatabaseEngines(t *testing.T) {
	for _, tc := range []struct{ driver, env string }{{"mysql", "KFP_HISTORY_MYSQL_TEST_DSN"}, {"postgres", "KFP_HISTORY_POSTGRES_TEST_DSN"}} {
		t.Run(tc.driver, func(t *testing.T) {
			dsn := os.Getenv(tc.env)
			if dsn == "" {
				t.Skip("set " + tc.env + " to run database integration tests")
			}
			exerciseNamespaceTransfer(t, externalDatabase(t, tc.driver, dsn), externalDatabase(t, tc.driver, dsn))
		})
	}
}

func TestNamespaceTransferCompletionWindow(t *testing.T) {
	ctx := context.Background()
	db := database(t)
	b := namespaceFixture(t, db)
	at, err := ExportNamespace(ctx, db, "team", "team", transfer.ExportOptions{CompletedAfter: 300, CompletedBefore: 301}, b.Pipelines, b.Versions)
	require.NoError(t, err)
	require.Len(t, at.Entries, 1)
	before, err := ExportNamespace(ctx, db, "team", "team", transfer.ExportOptions{CompletedBefore: 300}, b.Pipelines, b.Versions)
	require.NoError(t, err)
	require.Empty(t, before.Entries)
}

func TestNamespaceTransferLaterBatchSharesArtifacts(t *testing.T) {
	ctx := context.Background()
	source, dest := database(t), database(t)
	all := namespaceFixture(t, source)
	first, err := ExportNamespace(ctx, source, "team", "team", transfer.ExportOptions{CompletedBefore: 350}, all.Pipelines, all.Versions)
	require.NoError(t, err)
	p, err := PrepareTransfer(ctx, dest, first, transfer.ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, dest, p, false))
	var run model.Run
	require.NoError(t, source.Where(clause.Eq{Column: "UUID", Value: "run"}).Take(&run).Error)
	run.UUID = "later-run"
	run.FinishedAtInSec = 400
	create(t, source, &run)
	create(t, source, &model.Task{UUID: "later-task", RunUUID: run.UUID, Namespace: "team", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
	create(t, source, &model.ArtifactTask{UUID: "later-link", RunUUID: run.UUID, TaskID: "later-task", ArtifactID: "artifact", ArtifactKey: "model"})
	require.NoError(t, source.Model(&model.Experiment{}).Where(clause.Eq{Column: "UUID", Value: "experiment"}).Update("LastRunCreatedAtInSec", 400).Error)
	require.NoError(t, source.Model(&model.Job{}).Where(clause.Eq{Column: "UUID", Value: "schedule"}).Updates(map[string]any{"UpdatedAtInSec": 450, "Conditions": "ENABLED"}).Error)
	second, err := ExportNamespace(ctx, source, "team", "team", transfer.ExportOptions{CompletedAfter: 350}, all.Pipelines, all.Versions)
	require.NoError(t, err)
	p, err = PrepareTransfer(ctx, dest, second, transfer.ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, dest, p, false))
	require.Equal(t, int64(2), count(t, dest, &model.Run{}))
	require.Equal(t, int64(1), count(t, dest, &model.Artifact{}))
	require.Equal(t, int64(2), count(t, dest, &model.ArtifactTask{}))
}

func TestNamespaceTransferRejectsMissingDefault(t *testing.T) {
	b := namespaceFixture(t, database(t))
	b.Pipelines[0].DefaultVersionId = "missing"
	require.ErrorContains(t, ValidateNamespace(b, "team", "team"), "Default pipeline version")
}

func TestNamespaceTransferNewVersionDoesNotConflictWithParentReceipt(t *testing.T) {
	ctx := context.Background()
	source, dest := database(t), database(t)
	b := namespaceFixture(t, source)
	first, err := PrepareTransfer(ctx, dest, cloneTransfer(t, b), transfer.ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, dest, first, false))
	b.Versions = append(b.Versions, model.PipelineVersion{UUID: "new-version", Name: "new-version", PipelineId: b.Pipelines[0].UUID, CreatedAtInSec: 500, PipelineSpec: b.Versions[0].PipelineSpec})
	b.Pipelines[0].DefaultVersionId = "new-version"
	require.NoError(t, ValidateNamespace(b, "team", "team"))
	second, err := PrepareTransfer(ctx, dest, b, transfer.ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, second.Summary.Imported)
	require.NoError(t, CommitTransfer(ctx, dest, second, false))
	require.Equal(t, int64(3), count(t, dest, &model.PipelineVersion{}))
	var parent model.Pipeline
	require.NoError(t, dest.Take(&parent).Error)
	require.Equal(t, first.Bundle.Pipelines[0].DefaultVersionId, parent.DefaultVersionId, "existing persisted catalog parent must not be overwritten")
}
