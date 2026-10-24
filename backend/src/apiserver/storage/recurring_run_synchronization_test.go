// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOnlineAdoptionPendingReceiptBlocksOnlyItsClaim(t *testing.T) {
	for _, recordID := range []string{legacyRecurringRunRecordPrefix + "1", LegacyRecurringRunAdoptionID} {
		t.Run(recordID, func(t *testing.T) {
			sqlDB, _, store := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
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

func TestModeSynchronizationRetainsIntentAndDoesNotReseed(t *testing.T) {
	sqlDB, _, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	var job model.Job
	require.NoError(t, db.Take(&job, &model.Job{UUID: "1"}).Error)
	require.NoError(t, SetRecurringRunModeForReconciliation(db, job, false, 100))
	receipt, err := GetLegacyRecurringRunAdoptionForJob(db, "1")
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	require.Error(t, SynchronizeLegacyRecurringRunAdoption(db, "1", func(*model.Job, *model.RecurringRunState) error { return fmt.Errorf("interrupted") }))
	require.NoError(t, db.Take(&job, &model.Job{UUID: "1"}).Error)
	require.False(t, job.Enabled)
	require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "1", func(current *model.Job, state *model.RecurringRunState) error {
		require.False(t, current.Enabled)
		require.Zero(t, state.LastRunIndex)
		return nil
	}))
	receipt, err = GetLegacyRecurringRunAdoptionForJob(db, "1")
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: "1"}).Error)
	require.Error(t, SynchronizeLegacyRecurringRunAdoption(db, "1", nil))
	require.Error(t, SetRecurringRunModeForReconciliation(db, job, true, 101))
	_, err = store.GetRecurringRunState("1")
	require.Error(t, err)
}

func TestPreReceiptNativeStateRemainsTrusted(t *testing.T) {
	sqlDB, _, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: "legacy-2.18:1"}).Error)
	_, err = store.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
	require.NoError(t, err)
	var job model.Job
	require.NoError(t, db.Take(&job, &model.Job{UUID: "1"}).Error)
	require.NoError(t, SetRecurringRunModeForReconciliation(db, job, false, 120))
	receipt, err := GetLegacyRecurringRunAdoptionForJob(db, "1")
	require.NoError(t, err)
	require.False(t, receipt.Ready)
}
