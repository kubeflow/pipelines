// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"errors"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func adoptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(&model.Job{}, &model.RecurringRunState{}, &model.RecurringRunAdoption{}))
	return db
}

func adoptionTestCandidate(t *testing.T, db *gorm.DB, id string, enabled bool) RecurringRunAdoptionCandidate {
	t.Helper()
	interval, start, end := int64(60), int64(100), int64(10000)
	plugins := model.LargeText(`{"plugin":{"value":"unchanged"}}`)
	job := model.Job{
		UUID: id, DisplayName: "legacy-" + id, K8SName: "schedule-" + id,
		Namespace: "tenant", ServiceAccount: "runner", Description: "legacy schedule",
		MaxConcurrency: 3, NoCatchup: true, CreatedAtInSec: 100, UpdatedAtInSec: 200,
		Enabled: enabled, ExperimentId: "experiment", Conditions: "ENABLED", PluginsInputString: &plugins,
		Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
			IntervalSecond: &interval, PeriodicScheduleStartTimeInSec: &start, PeriodicScheduleEndTimeInSec: &end,
		}},
		PipelineSpec: model.PipelineSpec{
			PipelineId: "pipeline", PipelineVersionId: "version", PipelineName: "pipeline-name",
			PipelineSpecManifest: "pipeline-spec", WorkflowSpecManifest: "workflow-spec", Parameters: `{"v1":"parameter"}`,
			RuntimeConfig: model.RuntimeConfig{Parameters: `{"v2":"parameter"}`, PipelineRoot: "s3://root"},
		},
	}
	require.NoError(t, db.Create(&job).Error)
	return RecurringRunAdoptionCandidate{Job: job, State: model.RecurringRunState{
		JobUUID: id, RequestKey: "legacy-request-" + id, PipelineVersionID: "version",
		LastRunUUID:  "historical-execution-" + id,
		LastRunIndex: 7, LastScheduledAtInSec: 520, LastCreatedAtInSec: 525,
	}}
}

func TestLegacyRecurringRunAdoptionPreservesJobsAndProgress(t *testing.T) {
	db := adoptionTestDB(t)
	a := adoptionTestCandidate(t, db, "a", true)
	b := adoptionTestCandidate(t, db, "b", false)
	receipt, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{b, a}, 1000)
	require.NoError(t, err)
	require.Equal(t, &model.RecurringRunAdoption{
		ID: LegacyRecurringRunAdoptionID, AdoptedCount: 2, CompletedAt: 1000, JobIDs: `["a","b"]`,
	}, receipt)
	for _, candidate := range []RecurringRunAdoptionCandidate{a, b} {
		var job model.Job
		require.NoError(t, db.Take(&job, &model.Job{UUID: candidate.Job.UUID}).Error)
		require.Equal(t, candidate.Job, job)
		var state model.RecurringRunState
		require.NoError(t, db.Take(&state, &model.RecurringRunState{JobUUID: candidate.Job.UUID}).Error)
		require.Equal(t, candidate.State, state)
	}
	require.NoError(t, CompleteLegacyRecurringRunAdoption(db))
	require.NoError(t, CompleteLegacyRecurringRunAdoption(db))
	recorded, err := GetLegacyRecurringRunAdoption(db)
	require.NoError(t, err)
	receipt.Ready = true
	require.Equal(t, receipt, recorded)
}

func TestLegacyRecurringRunAdoptionRetryDoesNotReimport(t *testing.T) {
	db := adoptionTestDB(t)
	candidate := adoptionTestCandidate(t, db, "a", true)
	receipt, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{candidate}, 1000)
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: "a"}).Error)
	adoptionTestCandidate(t, db, "new-missing", true)
	for _, ready := range []bool{false, true} {
		if ready {
			require.NoError(t, CompleteLegacyRecurringRunAdoption(db))
			receipt.Ready = true
		}
		// Invalid replacement input and completion time must not change a receipt.
		candidate.State.LastRunIndex = -1
		again, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{candidate}, -1)
		require.NoError(t, err)
		require.Equal(t, receipt, again)
		var count int64
		require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&count).Error)
		require.Zero(t, count)
	}
}

func TestLegacyRecurringRunAdoptionZeroProgressAndNoJobs(t *testing.T) {
	for _, withJob := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty inventory", true: "zero progress"}[withJob], func(t *testing.T) {
			db := adoptionTestDB(t)
			receipt, err := GetLegacyRecurringRunAdoption(db)
			require.NoError(t, err)
			require.Nil(t, receipt)
			require.ErrorContains(t, CompleteLegacyRecurringRunAdoption(db), "receipt is missing")
			var candidates []RecurringRunAdoptionCandidate
			if withJob {
				candidate := adoptionTestCandidate(t, db, "a", false)
				candidate.State = model.RecurringRunState{JobUUID: "a"}
				candidates = append(candidates, candidate)
			}
			receipt, err = ApplyLegacyRecurringRunAdoption(db, candidates, 1000)
			require.NoError(t, err)
			require.EqualValues(t, len(candidates), receipt.AdoptedCount)
			require.False(t, receipt.Ready)
			if !withJob {
				require.Equal(t, model.LargeText("[]"), receipt.JobIDs)
			}
		})
	}
}

func TestLegacyRecurringRunAdoptionRollsBack(t *testing.T) {
	for _, failure := range []string{"snapshot changed", "missing candidate", "extra candidate", "existing state", "receipt insert"} {
		t.Run(failure, func(t *testing.T) {
			db := adoptionTestDB(t)
			a := adoptionTestCandidate(t, db, "a", true)
			b := adoptionTestCandidate(t, db, "b", false)
			candidates := []RecurringRunAdoptionCandidate{a, b}
			var initialStateCount int64
			switch failure {
			case "snapshot changed":
				b.Job.ServiceAccount = "changed-after-review"
				candidates[1] = b
			case "missing candidate":
				candidates = candidates[:1]
			case "extra candidate":
				extra := a
				extra.Job.UUID, extra.State.JobUUID = "deleted", "deleted"
				candidates = append(candidates, extra)
			case "existing state":
				require.NoError(t, db.Create(&b.State).Error)
				initialStateCount = 1
			case "receipt insert":
				require.NoError(t, db.Callback().Create().Before("gorm:create").Register("reject_adoption_receipt", func(tx *gorm.DB) {
					if tx.Statement.Table == "recurring_run_adoptions" {
						tx.AddError(errors.New("receipt insertion failed"))
					}
				}))
			}
			receipt, err := ApplyLegacyRecurringRunAdoption(db, candidates, 1000)
			require.Error(t, err)
			require.Nil(t, receipt)
			var count int64
			require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&count).Error)
			require.Equal(t, initialStateCount, count)
			receipt, err = GetLegacyRecurringRunAdoption(db)
			require.NoError(t, err)
			require.Nil(t, receipt)
		})
	}
}

func TestLegacyRecurringRunAdoptionRejectsInvalidProgress(t *testing.T) {
	tests := map[string]func(*model.RecurringRunState){
		"mismatched identity":       func(s *model.RecurringRunState) { s.JobUUID = "other" },
		"pending claim":             func(s *model.RecurringRunState) { s.Pending = true },
		"negative index":            func(s *model.RecurringRunState) { s.LastRunIndex = -1 },
		"overflowing index":         func(s *model.RecurringRunState) { s.LastRunIndex = math.MaxInt64 },
		"negative scheduled":        func(s *model.RecurringRunState) { s.LastScheduledAtInSec = -1 },
		"negative created":          func(s *model.RecurringRunState) { s.LastCreatedAtInSec = -1 },
		"overflowing scheduled":     func(s *model.RecurringRunState) { s.LastScheduledAtInSec = math.MaxInt64 },
		"overflowing created":       func(s *model.RecurringRunState) { s.LastCreatedAtInSec = math.MaxInt64 },
		"created before scheduled":  func(s *model.RecurringRunState) { s.LastCreatedAtInSec = s.LastScheduledAtInSec - 1 },
		"missing scheduled":         func(s *model.RecurringRunState) { s.LastScheduledAtInSec = 0 },
		"missing request key":       func(s *model.RecurringRunState) { s.RequestKey = "" },
		"long request key":          func(s *model.RecurringRunState) { s.RequestKey = strings.Repeat("a", 256) },
		"invalid UTF-8 request key": func(s *model.RecurringRunState) { s.RequestKey = string([]byte{0xff}) },
		"zero index with progress":  func(s *model.RecurringRunState) { s.LastRunIndex = 0 },
		"zero index with historical identity": func(s *model.RecurringRunState) {
			*s = model.RecurringRunState{JobUUID: s.JobUUID, LastRunUUID: "historical-execution"}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			db := adoptionTestDB(t)
			candidate := adoptionTestCandidate(t, db, "a", true)
			mutate(&candidate.State)
			_, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{candidate}, 1000)
			require.Error(t, err)
			var count int64
			require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestLegacyRecurringRunAdoptionRejectsChangedExecutionFields(t *testing.T) {
	for _, column := range []string{"Enabled", "Namespace", "Name", "ExperimentUUID", "PipelineVersionId", "RuntimeParameters", "PipelineRoot", "IntervalSecond", "PluginsInput"} {
		t.Run(column, func(t *testing.T) {
			db := adoptionTestDB(t)
			candidate := adoptionTestCandidate(t, db, "a", true)
			var value any = "changed"
			switch column {
			case "Enabled":
				value = false
			case "IntervalSecond":
				value = int64(120)
			}
			require.NoError(t, db.Model(&model.Job{}).Where(&model.Job{UUID: "a"}).UpdateColumn(column, value).Error)
			_, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{candidate}, 1000)
			require.ErrorContains(t, err, "changed")
		})
	}
}

func TestLegacyRecurringRunAdoptionAllowsStatusReports(t *testing.T) {
	db := adoptionTestDB(t)
	candidate := adoptionTestCandidate(t, db, "a", true)
	require.NoError(t, db.Model(&model.Job{}).Where(&model.Job{UUID: "a"}).Updates(map[string]any{
		"Conditions": "DISABLED", "UpdatedAtInSec": int64(999),
	}).Error)
	_, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{candidate}, 1000)
	require.NoError(t, err)
}

func TestLegacyRecurringRunAdoptionConcurrentRetries(t *testing.T) {
	db := adoptionTestDB(t)
	candidate := adoptionTestCandidate(t, db, "a", true)
	const attempts = 4
	var workers sync.WaitGroup
	receipts := make(chan *model.RecurringRunAdoption, attempts)
	errors := make(chan error, attempts)
	for i := range attempts {
		workers.Add(1)
		go func() {
			defer workers.Done()
			receipt, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{candidate}, int64(1000+i))
			receipts <- receipt
			errors <- err
		}()
	}
	workers.Wait()
	close(receipts)
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	recorded, err := GetLegacyRecurringRunAdoption(db)
	require.NoError(t, err)
	for receipt := range receipts {
		require.Equal(t, recorded, receipt)
	}
	var count int64
	require.NoError(t, db.Model(&model.RecurringRunState{}).Where(clause.Eq{Column: clause.Column{Name: "JobUUID"}, Value: "a"}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
