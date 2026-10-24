// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPendingScanDoesNotExpandGlobalInventory(t *testing.T) {
	sqlDB, _, _ := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + "1"}).Error)
	ids := make([]string, 70000)
	ids[0] = "1"
	for i := 1; i < len(ids); i++ {
		ids[i] = fmt.Sprintf("historical-%d", i)
	}
	inventory, err := json.Marshal(ids)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID, AdoptedCount: int64(len(ids)), JobIDs: model.LargeText(inventory)}).Error)
	queries := 0
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:bounded-scan", func(tx *gorm.DB) {
		if !tx.DryRun {
			queries++
		}
		require.LessOrEqual(t, len(tx.Statement.Vars), 8, "page parameters must not grow with the global inventory")
		require.NotContains(t, tx.Statement.SQL.String(), "JobIDs", "discovery must not read the global inventory")
	}))
	page, err := ListPendingLegacyRecurringRunAdoptions(db, "", 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "1", page[0].ID)
	require.Equal(t, 1, queries)
}

func TestPendingScanAdvancesPastGlobalNonmembers(t *testing.T) {
	sqlDB, _, jobs := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	for _, id := range []string{"a-nonmember", "b-nonmember", "z-member", "z-sealed"} {
		_, err := jobs.CreateJob(&model.Job{UUID: id, Namespace: "n1", Enabled: false})
		require.NoError(t, err)
		if id != "z-sealed" {
			require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + id}).Error)
		}
	}
	require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID, AdoptedCount: 2, JobIDs: `["z-member","z-sealed"]`}).Error)
	cursor := ""
	synchronized := []string{}
	for _, want := range []string{"a-nonmember", "b-nonmember", "z-member"} {
		page, err := ListPendingLegacyRecurringRunAdoptions(db, cursor, 1)
		require.NoError(t, err)
		require.Len(t, page, 1, "nonmember pages must not signal end of scan")
		require.Equal(t, want, page[0].ID)
		require.False(t, page[0].Enabled)
		cursor = page[0].ID
		require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, cursor, func(job *model.Job, _ *model.RecurringRunState) error {
			synchronized = append(synchronized, job.UUID)
			return nil
		}))
	}
	require.Equal(t, []string{"z-member"}, synchronized)
	page, err := ListPendingLegacyRecurringRunAdoptions(db, cursor, 1)
	require.NoError(t, err)
	require.Empty(t, page, "ready individual receipt must override pending global receipt")
	require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Where(&model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID}).Update("Ready", true).Error)
	page, err = ListPendingLegacyRecurringRunAdoptions(db, "", 100)
	require.NoError(t, err)
	require.Empty(t, page, "ready global receipt must not queue healthy unsealed jobs")
}
