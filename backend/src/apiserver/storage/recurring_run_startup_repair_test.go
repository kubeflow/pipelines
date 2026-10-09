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

func TestStartupRepairPagesPersistWorkWithoutReseeding(t *testing.T) {
	sqlDB, _, jobs := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	state, err := jobs.ClaimRecurringRun("1", "pending", 0, 100, 110, "")
	require.NoError(t, err)
	_, err = jobs.CreateJob(&model.Job{UUID: "z-disabled", Namespace: "n1", Enabled: false})
	require.NoError(t, err)
	_, err = jobs.CreateJob(&model.Job{UUID: "z-missing", Namespace: "n1", Enabled: false})
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: "z-missing"}).Error)
	// A page locks only its bounded job inventory, and records repair durably.
	page, err := PrepareRecurringRunStartupRepairs(db, "", 1, 200)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "1", page[0].ID)
	receipt, err := GetLegacyRecurringRunAdoptionForJob(db, "1")
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	actual, err := jobs.GetRecurringRunState("1")
	require.NoError(t, err)
	require.Equal(t, state, actual)
	pending, err := ListPendingLegacyRecurringRunAdoptions(db, "", 100)
	require.NoError(t, err)
	require.Contains(t, []string{pending[0].ID, pending[len(pending)-1].ID}, "1")
	page, err = PrepareRecurringRunStartupRepairs(db, "z-", 100, 200)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.False(t, page[0].Enabled)
	receipt, err = GetLegacyRecurringRunAdoptionForJob(db, "z-disabled")
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	_, err = jobs.GetRecurringRunState("z-missing")
	require.Error(t, err)
	_, err = PrepareRecurringRunStartupRepairs(db, "", 0, 200)
	require.Error(t, err)
}

func TestStartupRepairSkipsMalformedGlobalReceiptWithoutStarvingSealedJobs(t *testing.T) {
	sqlDB, _, jobs := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + "1"}).Error)
	require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID, AdoptedCount: 1, JobIDs: `["1","1"]`, Ready: true}).Error)
	_, err = jobs.CreateJob(&model.Job{UUID: "z-sealed", Namespace: "n1", Enabled: false})
	require.NoError(t, err)
	page, err := PrepareRecurringRunStartupRepairs(db, "", 100, 200)
	require.NoError(t, err)
	require.Equal(t, "1", page[0].ID)
	require.Equal(t, "z-sealed", page[len(page)-1].ID)
	_, err = GetLegacyRecurringRunAdoptionForJob(db, "1")
	require.ErrorContains(t, err, "invalid legacy adoption inventory")
	sealed, err := GetLegacyRecurringRunAdoptionForJob(db, "z-sealed")
	require.NoError(t, err)
	require.False(t, sealed.Ready)
	var count int64
	require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Where(&model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + "1"}).Count(&count).Error)
	require.Zero(t, count)
}

func TestPendingRepairScanIsolatesMalformedGlobalInventory(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(fmt.Sprint(ready), func(t *testing.T) {
			sqlDB, _, jobs := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + "1"}).Error)
			require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID, AdoptedCount: 1, JobIDs: `["1","1"]`, Ready: ready}).Error)
			_, err = jobs.CreateJob(&model.Job{UUID: "z-sealed", Namespace: "n1", Enabled: false})
			require.NoError(t, err)
			_, err = PrepareRecurringRunStartupRepairs(db, "", 100, 200)
			require.NoError(t, err)
			page, err := ListPendingLegacyRecurringRunAdoptions(db, "", 100)
			require.NoError(t, err)
			require.Equal(t, "z-sealed", page[len(page)-1].ID)
			_, err = GetLegacyRecurringRunAdoptionForJob(db, "1")
			require.ErrorContains(t, err, "invalid legacy adoption inventory")
			if !ready {
				require.Equal(t, "1", page[0].ID)
			}
			require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, "z-sealed", func(*model.Job, *model.RecurringRunState) error { return nil }))
			receipt, err := GetLegacyRecurringRunAdoptionForJob(db, "z-sealed")
			require.NoError(t, err)
			require.True(t, receipt.Ready)
		})
	}
}
