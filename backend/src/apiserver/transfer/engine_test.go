// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package transfer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type emptyRPC struct{ writes int }

func (f *emptyRPC) Call(_ context.Context, method string, _ Node) (Node, error) {
	if len(method) > 3 && method[:3] == "Put" {
		f.writes++
	}
	return Node{}, nil
}

type fakeSchedules struct {
	writes int
	specs  map[string]string
	rows   map[string]*model.Job
}

func (f *fakeSchedules) Prepare(_ context.Context, source string, j *model.Job, _ string, spec []byte, dry bool) (*model.Job, error) {
	copy := *j
	if f.specs == nil {
		f.specs = map[string]string{}
	}
	f.specs[j.UUID] = string(spec)
	if dry {
		return &copy, nil
	}
	if f.rows == nil {
		f.rows = map[string]*model.Job{}
	}
	if old := f.rows[source+j.UUID]; old != nil {
		return old, nil
	}
	copy.UUID = "local-" + j.UUID
	copy.K8SName = "local-name-" + j.UUID
	copy.Enabled = false
	copy.NoCatchup = true
	f.rows[source+j.UUID] = &copy
	f.writes++
	return &copy, nil
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	return db
}
func fixture(t *testing.T, db *gorm.DB) *Engine {
	t.Helper()
	for _, row := range []any{
		&model.Experiment{UUID: "experiment", Name: "Default", Namespace: "team", StorageState: model.StorageStateAvailable},
		&model.Experiment{UUID: "empty", Name: "Empty", Namespace: "team", StorageState: model.StorageStateAvailable},
		&model.Experiment{UUID: "other", Name: "Other", Namespace: "private"},
		&model.Pipeline{UUID: "pipeline", Name: "catalog", Namespace: "team", Status: model.PipelineReady, Tags: map[string]string{"team": "science"}},
		&model.PipelineVersion{UUID: "version", Name: "v1", PipelineId: "pipeline", Status: model.PipelineVersionReady, PipelineSpec: "spec"},
		&model.PipelineVersion{UUID: "unused", Name: "v2", PipelineId: "pipeline", Status: model.PipelineVersionReady, PipelineSpec: "unused spec"},
		&model.PipelineTag{PipelineID: "pipeline", TagKey: "team", TagValue: "science"},
		&model.PipelineVersionTag{PipelineVersionID: "unused", TagKey: "stage", TagValue: "draft"},
		&model.Job{UUID: "schedule", K8SName: "original", Namespace: "team", DisplayName: "Every hour", ExperimentId: "experiment", Enabled: true, NoCatchup: false, PipelineSpec: model.PipelineSpec{PipelineId: "pipeline", PipelineVersionId: "version"}},
		&model.Run{UUID: "run", Namespace: "team", DisplayName: "Finished", ExperimentId: "experiment", RecurringRunId: "schedule", StorageState: model.StorageStateAvailable, RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded, CreatedAtInSec: 10, FinishedAtInSec: 20}},
		&model.Run{UUID: "active", Namespace: "team", ExperimentId: "experiment", RunDetails: model.RunDetails{State: model.RuntimeStateRunning, CreatedAtInSec: 30}},
		&model.Task{UUID: "task", RunID: "run", Namespace: "team", Fingerprint: "source-cache", State: model.RuntimeStateSucceeded},
		&model.RunMetric{RunUUID: "run", NodeID: "node", Name: "accuracy", NumberValue: 0.95},
		&model.ResourceReference{ResourceUUID: "run", ResourceType: model.RunResourceType, ReferenceUUID: "schedule", ReferenceType: model.JobResourceType, Relationship: model.CreatorRelationship},
	} {
		require.NoError(t, create(db, row))
	}
	return &Engine{DB: db, RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
}

func TestNamespaceExportIncludesEmptyExperimentsAndFullCatalog(t *testing.T) {
	src := fixture(t, testDB(t))
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	var b Bundle
	require.NoError(t, json.Unmarshal(data, &b))
	require.Len(t, b.Experiments, 2)
	require.Len(t, b.Pipelines, 1)
	require.Len(t, b.Versions, 2)
	require.Len(t, b.Runs, 1)
	require.Len(t, b.Schedules, 1)
	require.Equal(t, "science", b.Pipelines[0].Tags["team"])
	require.True(t, b.Schedules[0].Enabled)
	boundary, err := src.Export(context.Background(), "team", ExportOptions{CompletedAfter: 20, CompletedBefore: 21})
	require.NoError(t, err)
	var exact Bundle
	require.NoError(t, json.Unmarshal(boundary, &exact))
	require.Len(t, exact.Runs, 1)
	data, err = src.Export(context.Background(), "team", ExportOptions{CompletedAfter: 21})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &b))
	require.Empty(t, b.Runs)
	require.Len(t, b.Experiments, 2)
	require.Len(t, b.Versions, 2)
}

func TestNamespaceImportPreviewApplyRepeat(t *testing.T) {
	src := fixture(t, testDB(t))
	require.NoError(t, create(src.DB, &model.RecurringRunState{JobUUID: "schedule", RequestKey: "source-tick", PipelineVersionID: "version", LastRunIndex: 50, Pending: true}))
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	rpc := &emptyRPC{}
	schedules := &fakeSchedules{}
	dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: rpc}, Schedules: schedules}
	preview, err := dst.Import(context.Background(), "team", data, ImportOptions{DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 7, preview.Imported)
	require.Zero(t, schedules.writes)
	require.Zero(t, rpc.writes)
	var count int64
	require.NoError(t, dst.DB.Model(&model.Experiment{}).Count(&count).Error)
	require.Zero(t, count)
	result, err := dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, preview.Counts, result.Counts)
	require.Equal(t, 1, schedules.writes)
	var run model.Run
	require.NoError(t, dst.DB.First(&run, "UUID = ?", "run").Error)
	require.NotNil(t, run.ImportedFrom)
	require.Equal(t, "local-schedule", run.RecurringRunId)
	var job model.Job
	require.NoError(t, dst.DB.First(&job, "UUID = ?", run.RecurringRunId).Error)
	require.False(t, job.Enabled)
	require.True(t, job.NoCatchup)
	var state model.RecurringRunState
	require.NoError(t, dst.DB.Where(equal("JobUUID", job.UUID)).First(&state).Error)
	require.Equal(t, model.RecurringRunState{JobUUID: job.UUID}, state)
	require.NoError(t, dst.DB.Model(&model.RecurringRunState{}).Where(equal("JobUUID", job.UUID)).Updates(map[string]any{"LastRunIndex": 2, "Pending": true}).Error)
	var task model.Task
	require.NoError(t, dst.DB.First(&task, "UUID = ?", "task").Error)
	require.Empty(t, task.Fingerprint)
	repeat, err := dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Zero(t, repeat.Imported)
	require.Equal(t, 7, repeat.Skipped)
	require.Equal(t, 1, schedules.writes)
	require.NoError(t, dst.DB.Where(equal("JobUUID", job.UUID)).First(&state).Error)
	require.EqualValues(t, 2, state.LastRunIndex)
	require.True(t, state.Pending)
}

func TestNamespaceImportLaterBatchReusesSchedule(t *testing.T) {
	src := fixture(t, testDB(t))
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	schedules := &fakeSchedules{}
	dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: schedules}
	_, err = dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, create(src.DB, &model.Run{UUID: "later", Namespace: "team", ExperimentId: "experiment", RecurringRunId: "schedule", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded, CreatedAtInSec: 40, FinishedAtInSec: 50}}))
	require.NoError(t, src.DB.Model(&model.Job{}).Where(equal("UUID", "schedule")).Update("UpdatedAtInSec", 100).Error)
	data, err = src.Export(context.Background(), "team", ExportOptions{CompletedAfter: 21})
	require.NoError(t, err)
	result, err := dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 1, schedules.writes)
	var count int64
	require.NoError(t, dst.DB.Model(&model.Job{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	var run model.Run
	require.NoError(t, dst.DB.Where(equal("UUID", "later")).First(&run).Error)
	require.Equal(t, "local-schedule", run.RecurringRunId)
}

func rewriteArchive(t *testing.T, data []byte, change func(*Bundle)) []byte {
	t.Helper()
	var b Bundle
	require.NoError(t, json.Unmarshal(data, &b))
	change(&b)
	b.Digest = ""
	b.Digest = hash(b)
	out, err := json.Marshal(b)
	require.NoError(t, err)
	return out
}
func TestNamespaceImportRejectsConflictsBeforeStaging(t *testing.T) {
	src := fixture(t, testDB(t))
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	for _, test := range []struct {
		name   string
		change func(*Bundle)
	}{
		{"malicious runtime namespace", func(b *Bundle) {
			b.RuntimeNamespace = "private"
			b.Schedules[0].Namespace = "private"
			b.Runs[0].Run.Namespace = "private"
			b.Runs[0].Tasks[0].Namespace = "private"
			b.References = append(b.References, model.ResourceReference{ResourceType: model.RunResourceType, ResourceUUID: "run", ReferenceType: model.NamespaceResourceType, ReferenceUUID: "private", Relationship: model.OwnerRelationship})
		}},
		{"other namespace", func(b *Bundle) { b.Runs[0].Run.Namespace = "private" }},
		{"unfinished", func(b *Bundle) { b.Runs[0].Run.State = model.RuntimeStateRunning }},
		{"missing experiment", func(b *Bundle) { b.Runs[0].Run.ExperimentId = "absent" }},
		{"missing task parent", func(b *Bundle) { b.Runs[0].Tasks[0].ParentTaskId = "absent" }},
		{"cycle", func(b *Bundle) { b.Runs[0].Tasks[0].ParentTaskId = b.Runs[0].Tasks[0].UUID }},
		{"missing pinned version", func(b *Bundle) { b.CatalogDefaults = map[string]string{"pipeline": "missing"} }},
		{"missing pinned pipeline", func(b *Bundle) { b.CatalogDefaults = map[string]string{"missing": "v1"} }},
		{"SQL cannot honor Kubernetes pin", func(b *Bundle) { b.CatalogDefaults = map[string]string{"pipeline": "v1"} }},
		{"source URI", func(b *Bundle) { b.Versions[0].PipelineSpecURI = "s3://source/catalog" }},
		{"missing inline definition", func(b *Bundle) { b.Versions[0].PipelineSpec = "" }},
		{"cross generation", func(b *Bundle) { b.Format = "kfp-native/v1" }},
		{"missing execution", func(b *Bundle) { b.Runs[0].Tasks[0].MLMDExecutionID = "42" }},
		{"missing artifact", func(b *Bundle) { b.Runs[0].Tasks[0].MLMDInputs = `{"value":{"artifact_ids":[42]}}` }},
	} {
		t.Run(test.name, func(t *testing.T) {
			schedules := &fakeSchedules{}
			dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: schedules}
			_, err := dst.Import(context.Background(), "team", rewriteArchive(t, data, test.change), ImportOptions{})
			require.Error(t, err)
			require.Zero(t, schedules.writes)
		})
	}
	t.Run("existing default", func(t *testing.T) {
		dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
		require.NoError(t, create(dst.DB, &model.Experiment{UUID: "native", Namespace: "team", Name: "Default"}))
		_, err := dst.Import(context.Background(), "team", data, ImportOptions{})
		require.ErrorContains(t, err, "prefix")
		_, err = dst.Import(context.Background(), "team", data, ImportOptions{NamePrefix: "old-"})
		require.NoError(t, err)
	})
	t.Run("native ID never adopted", func(t *testing.T) {
		dst := &Engine{DB: testDB(t), RuntimeNamespace: "team"}
		require.NoError(t, create(dst.DB, &model.Experiment{UUID: "experiment", Namespace: "team", Name: "Default"}))
		_, err := dst.Import(context.Background(), "team", data, ImportOptions{})
		require.ErrorContains(t, err, "already exists")
	})
}

func TestNamespaceImportSQLRollbackAfterScheduleStaging(t *testing.T) {
	src := fixture(t, testDB(t))
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	schedules := &fakeSchedules{}
	dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: schedules}
	require.NoError(t, dst.DB.Callback().Create().Before("gorm:create").Register("fail-run", func(tx *gorm.DB) {
		if tx.Statement.Table == "run_details" {
			tx.AddError(fmt.Errorf("injected write failure"))
		}
	}))
	_, err = dst.Import(context.Background(), "team", data, ImportOptions{})
	require.ErrorContains(t, err, "injected")
	var count int64
	require.NoError(t, dst.DB.Model(&model.Experiment{}).Count(&count).Error)
	require.Zero(t, count)
	require.Equal(t, 1, schedules.writes)
	require.NoError(t, dst.DB.Callback().Create().Remove("fail-run"))
	_, err = dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, schedules.writes)
}

func TestSingleUserExperimentOwnership(t *testing.T) {
	src := fixture(t, testDB(t))
	require.NoError(t, src.DB.Model(&model.Experiment{}).Where(equal("Namespace", "team")).Update("Namespace", "").Error)
	require.NoError(t, src.DB.Model(&model.Pipeline{}).Where(equal("Namespace", "team")).Update("Namespace", "").Error)
	data, err := src.Export(context.Background(), "", ExportOptions{})
	require.NoError(t, err)
	dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = dst.Import(context.Background(), "", data, ImportOptions{})
	require.NoError(t, err)
	var run model.Run
	require.NoError(t, dst.DB.Where(equal("UUID", "run")).First(&run).Error)
	require.Equal(t, "team", run.Namespace)
}

func TestExportHistoryWithDeletedCatalogKeepsEmbeddedSpec(t *testing.T) {
	src := fixture(t, testDB(t))
	require.NoError(t, src.DB.Model(&model.Run{}).Where(equal("UUID", "run")).Updates(map[string]any{"PipelineId": "deleted-pipeline", "PipelineVersionId": "deleted-version", "PipelineSpecManifest": "embedded-history-spec"}).Error)
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	var archive Bundle
	require.NoError(t, json.Unmarshal(data, &archive))
	require.Empty(t, archive.Runs[0].Run.PipelineId)
	require.Empty(t, archive.Runs[0].Run.PipelineVersionId)
	require.Equal(t, model.LargeText("embedded-history-spec"), archive.Runs[0].Run.PipelineSpecManifest)
}

// CI points this test at an empty disposable MySQL database.
func TestTransferMySQLIntegration(t *testing.T) {
	dsn := os.Getenv("KFP_TRANSFER_MYSQL_DSN")
	if dsn == "" {
		if os.Getenv("KFP_TRANSFER_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("KFP_TRANSFER_MYSQL_DSN is required")
		}
		t.Skip("MySQL fixture not configured")
	}
	config, err := mysqldriver.ParseDSN(dsn)
	require.NoError(t, err)
	config.DBName = ""
	admin, err := sql.Open("mysql", config.FormatDSN())
	require.NoError(t, err)
	defer admin.Close()
	open := func() *gorm.DB {
		name := "transfer_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		_, err := admin.Exec("CREATE DATABASE `" + name + "`")
		require.NoError(t, err)
		copy := *config
		copy.DBName = name
		db, err := gorm.Open(mysql.Open(copy.FormatDSN()), &gorm.Config{})
		require.NoError(t, err)
		require.NoError(t, db.AutoMigrate(model.AllModels()...))
		t.Cleanup(func() {
			pool, err := db.DB()
			require.NoError(t, err)
			require.NoError(t, pool.Close())
			cleanup, err := sql.Open("mysql", config.FormatDSN())
			require.NoError(t, err)
			defer cleanup.Close()
			_, err = cleanup.Exec("DROP DATABASE `" + name + "`")
			require.NoError(t, err)
		})
		return db
	}
	source := fixture(t, open())
	data, err := source.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	target := &Engine{DB: open(), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	preview, err := target.Import(context.Background(), "team", data, ImportOptions{DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 7, preview.Imported)
	result, err := target.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 7, result.Imported)
	result, err = target.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 7, result.Skipped)
	var task model.Task
	require.NoError(t, target.DB.Where(equal("UUID", "task")).First(&task).Error)
	require.Empty(t, task.Fingerprint)
	var job model.Job
	require.NoError(t, target.DB.Where(equal("UUID", "local-schedule")).First(&job).Error)
	require.False(t, job.Enabled)
	require.True(t, job.NoCatchup)
}

func TestReexportImportedCompletedHistory(t *testing.T) {
	source := fixture(t, testDB(t))
	data, err := source.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	middle := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = middle.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	again, err := middle.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	var archive Bundle
	require.NoError(t, json.Unmarshal(again, &archive))
	require.Len(t, archive.Runs, 1)
	require.Nil(t, archive.Runs[0].Run.ImportedFrom)
	require.Nil(t, archive.Runs[0].Run.ImportDigest)
	destination := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = destination.Import(context.Background(), "team", again, ImportOptions{})
	require.NoError(t, err)
}

func TestLaterCatalogVersionKeepsDestinationDefault(t *testing.T) {
	src := fixture(t, testDB(t))
	require.NoError(t, src.DB.Model(&model.Pipeline{}).Where(equal("UUID", "pipeline")).Update("DefaultVersionId", "version").Error)
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, create(src.DB, &model.PipelineVersion{UUID: "new-version", Name: "v3", PipelineId: "pipeline", Status: model.PipelineVersionReady, PipelineSpec: "new spec"}))
	require.NoError(t, src.DB.Model(&model.Pipeline{}).Where(equal("UUID", "pipeline")).Update("DefaultVersionId", "new-version").Error)
	data, err = src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	result, err := dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	var pipeline model.Pipeline
	require.NoError(t, dst.DB.Where(equal("UUID", "pipeline")).First(&pipeline).Error)
	require.Equal(t, "version", pipeline.DefaultVersionId)
}

func TestNamespaceExportHydratesAndClearsSourceCatalogURI(t *testing.T) {
	src := fixture(t, testDB(t))
	require.NoError(t, src.DB.Model(&model.PipelineVersion{}).Where(equal("UUID", "version")).Updates(map[string]any{"PipelineSpec": "", "PipelineSpecURI": "s3://source/catalog/version.yaml"}).Error)
	calls := 0
	src.LoadPipelineSpec = func(_ context.Context, version *model.PipelineVersion) ([]byte, error) {
		calls++
		require.Equal(t, model.LargeText("s3://source/catalog/version.yaml"), version.PipelineSpecURI)
		return []byte("hydrated definition"), nil
	}
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	var archive Bundle
	require.NoError(t, json.Unmarshal(data, &archive))
	for _, version := range archive.Versions {
		require.Empty(t, version.PipelineSpecURI)
		require.NotEmpty(t, version.PipelineSpec)
	}
	dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = dst.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	var version model.PipelineVersion
	require.NoError(t, dst.DB.Where(equal("UUID", "version")).First(&version).Error)
	require.Empty(t, version.PipelineSpecURI)
	require.Equal(t, model.LargeText("hydrated definition"), version.PipelineSpec)
}

type retainedDefaultCatalog struct{ name string }

func (retainedDefaultCatalog) Export(context.Context, string) ([]model.Pipeline, []model.PipelineVersion, map[string]string, error) {
	return nil, nil, nil, nil
}
func (c retainedDefaultCatalog) Prepare(_ context.Context, _, _ string, pipelines []model.Pipeline, versions []model.PipelineVersion, _ map[string]string, _ bool) (map[string]string, map[string]string, map[string]string, error) {
	pmap, vmap := map[string]string{}, map[string]string{}
	for _, p := range pipelines {
		pmap[p.UUID] = p.UUID
	}
	for _, v := range versions {
		vmap[v.UUID] = v.UUID
	}
	return pmap, vmap, map[string]string{"pipeline": c.name}, nil
}

func TestTransferScheduleValidatesRetainedDestinationDefault(t *testing.T) {
	src := fixture(t, testDB(t))
	require.NoError(t, src.DB.Model(&model.Job{}).Where(equal("UUID", "schedule")).Update("PipelineVersionId", "").Error)
	data, err := src.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	data = rewriteArchive(t, data, func(b *Bundle) { b.CatalogDefaults = map[string]string{"pipeline": "v2"} })
	schedules := &fakeSchedules{}
	dst := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: schedules, Catalog: retainedDefaultCatalog{name: "v1"}}
	_, err = dst.Import(context.Background(), "team", data, ImportOptions{DryRun: true})
	require.NoError(t, err)
	require.Equal(t, "spec", schedules.specs["schedule"])
	dst.Catalog = retainedDefaultCatalog{name: "destination-only"}
	_, err = dst.Import(context.Background(), "team", data, ImportOptions{DryRun: true})
	require.ErrorContains(t, err, "destination catalog default is absent")
	require.Zero(t, schedules.writes)
}
