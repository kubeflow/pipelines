// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"fmt"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestOnlineAdoptionConcurrentReplayAndInterruptedSync(t *testing.T) {
	db := adoptionTestDB(t)
	candidate := adoptionTestCandidate(t, db, "a", true)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- ApplyLegacyRecurringRunAdoptionForJob(db, candidate, 1000) }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	receipt, err := GetLegacyRecurringRunAdoptionForJob(db, "a")
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	err = SynchronizeLegacyRecurringRunAdoption(db, "a", func(*model.Job, *model.RecurringRunState) error { return fmt.Errorf("cr temporarily unavailable") })
	require.ErrorContains(t, err, "temporarily unavailable")
	receipt, err = GetLegacyRecurringRunAdoptionForJob(db, "a")
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	// Retry synchronizes current SQL progress, never the stale candidate.
	require.NoError(t, db.Model(&model.RecurringRunState{}).Where(&model.RecurringRunState{JobUUID: "a"}).UpdateColumn("LastRunIndex", 8).Error)
	require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "a", func(job *model.Job, state *model.RecurringRunState) error {
		require.EqualValues(t, 8, state.LastRunIndex)
		return nil
	}))
	require.NoError(t, ApplyLegacyRecurringRunAdoptionForJob(db, candidate, 2000))
	require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "a", func(*model.Job, *model.RecurringRunState) error {
		t.Fatal("completed record synchronized again")
		return nil
	}))
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: "a"}).Error)
	require.ErrorContains(t, ApplyLegacyRecurringRunAdoptionForJob(db, candidate, 2000), "refusing to reseed")
	require.ErrorContains(t, SynchronizeLegacyRecurringRunAdoption(db, "a", nil), "refusing to reseed")
}

func TestOnlineAdoptionGlobalReceiptDoesNotFreezeInventory(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(fmt.Sprint(ready), func(t *testing.T) {
			db := adoptionTestDB(t)
			a := adoptionTestCandidate(t, db, "a", true)
			b := adoptionTestCandidate(t, db, "b", false)
			_, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{a, b}, 1000)
			require.NoError(t, err)
			if ready {
				require.NoError(t, CompleteLegacyRecurringRunAdoption(db))
			}
			c := adoptionTestCandidate(t, db, "c", true)
			require.NoError(t, ApplyLegacyRecurringRunAdoptionForJob(db, c, 1001))
			require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "a", func(*model.Job, *model.RecurringRunState) error { return nil }))
			pending, err := ListPendingLegacyRecurringRunAdoptions(db, "", 10)
			require.NoError(t, err)
			var ids []string
			for _, job := range pending {
				ids = append(ids, job.ID)
			}
			if ready {
				require.Equal(t, []string{"c"}, ids)
			} else {
				require.Equal(t, []string{"b", "c"}, ids)
			}
			page, err := ListPendingLegacyRecurringRunAdoptions(db, "b", 1)
			require.NoError(t, err)
			require.Len(t, page, 1)
			require.Equal(t, "c", page[0].ID)
			require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: "b"}).Error)
			require.ErrorContains(t, ApplyLegacyRecurringRunAdoptionForJob(db, b, 1002), "refusing to reseed")
		})
	}
}

func TestOnlineAdoptionDefinitionMutationAndDeletedJob(t *testing.T) {
	db := adoptionTestDB(t)
	candidate := adoptionTestCandidate(t, db, "a", true)
	require.NoError(t, db.Model(&model.Job{}).Where(&model.Job{UUID: "a"}).UpdateColumn("Enabled", false).Error)
	require.ErrorContains(t, ApplyLegacyRecurringRunAdoptionForJob(db, candidate, 1000), "changed during adoption")
	candidate.Job.Enabled = false
	require.NoError(t, ApplyLegacyRecurringRunAdoptionForJob(db, candidate, 1000))
	require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "a", func(job *model.Job, _ *model.RecurringRunState) error { require.False(t, job.Enabled); return nil }))
	require.NoError(t, db.Delete(&model.Job{}, &model.Job{UUID: "a"}).Error)
	require.Error(t, SynchronizeLegacyRecurringRunAdoption(db, "a", func(*model.Job, *model.RecurringRunState) error { t.Fatal("deleted job updated"); return nil }))
}

func TestOnlineAdoptionModeChangeUsesSameSerialization(t *testing.T) {
	db := adoptionTestDB(t)
	candidate := adoptionTestCandidate(t, db, "a", true)
	require.NoError(t, ApplyLegacyRecurringRunAdoptionForJob(db, candidate, 1000))
	entered, release := make(chan struct{}), make(chan struct{})
	synced := make(chan error, 1)
	go func() {
		synced <- SynchronizeLegacyRecurringRunAdoption(db, "a", func(job *model.Job, _ *model.RecurringRunState) error {
			close(entered)
			<-release
			require.True(t, job.Enabled)
			return nil
		})
	}()
	<-entered
	changed := make(chan error, 1)
	go func() {
		changed <- SetRecurringRunModeForReconciliation(db, candidate.Job, false, 1001)
	}()
	close(release)
	require.NoError(t, <-synced)
	require.NoError(t, <-changed)
	var current model.Job
	require.NoError(t, db.Take(&current, &model.Job{UUID: "a"}).Error)
	require.False(t, current.Enabled)
	// Failed CR reconciliation retains desired SQL mode and a pending marker.
	require.NoError(t, SetRecurringRunModeForReconciliation(db, current, true, 1002))
	err := SynchronizeLegacyRecurringRunAdoption(db, "a", func(*model.Job, *model.RecurringRunState) error { return fmt.Errorf("failed CR update") })
	require.Error(t, err)
	require.NoError(t, db.Take(&current, &model.Job{UUID: "a"}).Error)
	require.True(t, current.Enabled)
	receipt, err := GetLegacyRecurringRunAdoptionForJob(db, "a")
	require.NoError(t, err)
	require.False(t, receipt.Ready)

}

func TestOnlineAdoptionPendingReceiptBlocksOnlyItsClaim(t *testing.T) {
	for _, recordID := range []string{legacyRecurringRunRecordPrefix + "1", LegacyRecurringRunAdoptionID} {
		t.Run(recordID, func(t *testing.T) {
			sqlDB, _, store := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			db, err := OpenTransferDB(sqlDB, "sqlite")
			require.NoError(t, err)
			require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + "1"}).Error)
			require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: recordID, AdoptedCount: 1, JobIDs: `["1"]`, CompletedAt: 100}).Error)
			_, err = store.ClaimRecurringRun("1", "next", 0, 100, 110, "v")
			require.ErrorContains(t, err, "synchronization is pending")
			if recordID == LegacyRecurringRunAdoptionID {
				require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + "1", AdoptedCount: 1, JobIDs: `["1"]`, Ready: true, CompletedAt: 100}).Error)
			} else {
				require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Where(&model.RecurringRunAdoption{ID: recordID}).UpdateColumn("Ready", true).Error)
			}
			_, err = store.ClaimRecurringRun("1", "next", 0, 100, 110, "v")
			require.NoError(t, err)
		})
	}
}

func TestNativeScheduleStateDeletionCannotBeAdopted(t *testing.T) {
	sqlDB, _, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	job, err := store.CreateJob(&model.Job{UUID: "native-sealed", K8SName: "native", Namespace: "test", Enabled: true})
	require.NoError(t, err)
	db, err := OpenTransferDB(sqlDB, "sqlite")
	require.NoError(t, err)
	receipt, err := GetLegacyRecurringRunAdoptionForJob(db, job.UUID)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.True(t, receipt.Ready)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: job.UUID}).Error)
	require.ErrorContains(t, ApplyLegacyRecurringRunAdoptionForJob(db, RecurringRunAdoptionCandidate{Job: *job, State: model.RecurringRunState{JobUUID: job.UUID}}, 100), "refusing to reseed")
	_, err = ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{{Job: *job, State: model.RecurringRunState{JobUUID: job.UUID}}}, 100)
	require.ErrorContains(t, err, "refusing to reseed")

}

func TestOnlineModeReconciliationSQLFailureRetainsPendingMarker(t *testing.T) {
	db := adoptionTestDB(t)
	candidate := adoptionTestCandidate(t, db, "a", true)
	require.NoError(t, ApplyLegacyRecurringRunAdoptionForJob(db, candidate, 1000))
	require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "a", func(*model.Job, *model.RecurringRunState) error { return nil }))
	require.NoError(t, SetRecurringRunModeForReconciliation(db, candidate.Job, false, 1001))
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("fail_receipt_ready", func(tx *gorm.DB) {
		if tx.Statement.Table == "recurring_run_adoptions" {
			tx.AddError(fmt.Errorf("interrupted receipt commit"))
		}
	}))
	applied := false
	err := SynchronizeLegacyRecurringRunAdoption(db, "a", func(job *model.Job, _ *model.RecurringRunState) error {
		require.False(t, job.Enabled)
		applied = true
		return nil
	})
	require.ErrorContains(t, err, "interrupted receipt commit")
	require.True(t, applied)
	receipt, err := GetLegacyRecurringRunAdoptionForJob(db, "a")
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	require.NoError(t, db.Callback().Update().Remove("fail_receipt_ready"))
	require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "a", func(job *model.Job, _ *model.RecurringRunState) error { require.False(t, job.Enabled); return nil }))
}
