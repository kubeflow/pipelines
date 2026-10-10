// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package clientmanager

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type prerequisiteJob struct {
	UUID    string `gorm:"column:UUID;primaryKey;type:varchar(191)"`
	Enabled bool   `gorm:"column:Enabled"`
}

func (prerequisiteJob) TableName() string { return "jobs" }

func prerequisiteFixture(t *testing.T, db *gorm.DB, id string, enabled bool) {
	t.Helper()
	require.NoError(t, db.Create(&prerequisiteJob{UUID: id, Enabled: enabled}).Error)
	require.NoError(t, db.Create(&model.RecurringRunState{JobUUID: id, RequestKey: "", PipelineVersionID: "", Pending: true}).Error)
	inventory, err := json.Marshal([]string{id})
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: recurringRunAdoptionSourceID + ":" + id, AdoptedCount: 1, JobIDs: model.LargeText(inventory), Ready: true}).Error)
}

func prerequisiteSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&prerequisiteJob{}, &model.RecurringRunState{}, &model.RecurringRunAdoption{}))
}

func TestRecurringRunUpgradePrerequisiteFreshDatabase(t *testing.T) {
	db := getTestSQLite(t)
	require.NoError(t, requireCompletedRecurringRunAdoption(db))
	tables, err := db.Migrator().GetTables()
	require.NoError(t, err)
	require.Empty(t, tables)
	require.NoError(t, db.AutoMigrate(&prerequisiteJob{}))
	require.NoError(t, requireCompletedRecurringRunAdoption(db))
	require.False(t, db.Migrator().HasTable(&model.RecurringRunAdoption{}))
}

func TestRecurringRunUpgradePrerequisiteRejectsBeforeMigration(t *testing.T) {
	db := getTestSQLite(t)
	require.NoError(t, db.AutoMigrate(&prerequisiteJob{}))
	require.NoError(t, db.Create(&prerequisiteJob{UUID: "disabled", Enabled: false}).Error)
	tables, err := db.Migrator().GetTables()
	require.NoError(t, err)
	var schemaBefore []struct{ SQL string }
	require.NoError(t, db.Raw("SELECT sql FROM sqlite_master ORDER BY name").Scan(&schemaBefore).Error)
	err = requireCompletedRecurringRunAdoption(db)
	require.ErrorContains(t, err, "finish automatic SQL-state adoption")
	after, err := db.Migrator().GetTables()
	require.NoError(t, err)
	require.ElementsMatch(t, tables, after)
	var schemaAfter []struct{ SQL string }
	require.NoError(t, db.Raw("SELECT sql FROM sqlite_master ORDER BY name").Scan(&schemaAfter).Error)
	require.Equal(t, schemaBefore, schemaAfter)
	var jobs []prerequisiteJob
	require.NoError(t, db.Find(&jobs).Error)
	require.Equal(t, []prerequisiteJob{{UUID: "disabled", Enabled: false}}, jobs)
}

func TestRecurringRunUpgradePrerequisite(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*gorm.DB) error
		want   string
	}{
		{name: "ready native or adopted with pending claim"},
		{name: "disabled missing state", mutate: func(db *gorm.DB) error {
			return db.Delete(&model.RecurringRunState{}, map[string]any{"JobUUID": "schedule"}).Error
		}, want: "missing scheduling state"},
		{name: "native state predates receipt", mutate: func(db *gorm.DB) error {
			return db.Delete(&model.RecurringRunAdoption{}, map[string]any{"ID": recurringRunAdoptionSourceID + ":schedule"}).Error
		}},
		{name: "pending CR synchronization must not deadlock rollout", mutate: func(db *gorm.DB) error {
			return db.Model(&model.RecurringRunAdoption{}).Where(map[string]any{"ID": recurringRunAdoptionSourceID + ":schedule"}).Update("Ready", false).Error
		}},
		{name: "wrong inventory", mutate: func(db *gorm.DB) error {
			return db.Model(&model.RecurringRunAdoption{}).Where(map[string]any{"ID": recurringRunAdoptionSourceID + ":schedule"}).Update("JobIDs", `["other"]`).Error
		}, want: "invalid adoption receipt"},
		{name: "malformed inventory", mutate: func(db *gorm.DB) error {
			return db.Model(&model.RecurringRunAdoption{}).Where(map[string]any{"ID": recurringRunAdoptionSourceID + ":schedule"}).Update("JobIDs", `{`).Error
		}, want: "invalid adoption inventory"},
		{name: "wrong count", mutate: func(db *gorm.DB) error {
			return db.Model(&model.RecurringRunAdoption{}).Where(map[string]any{"ID": recurringRunAdoptionSourceID + ":schedule"}).Update("AdoptedCount", 2).Error
		}, want: "incomplete adoption inventory"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := getTestSQLite(t)
			prerequisiteSchema(t, db)
			prerequisiteFixture(t, db, "schedule", false)
			if tc.mutate != nil {
				require.NoError(t, tc.mutate(db))
			}
			err := requireCompletedRecurringRunAdoption(db)
			if tc.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
}

func TestRecurringRunUpgradePrerequisiteGlobalInventory(t *testing.T) {
	for _, tc := range []struct {
		name, inventory   string
		count             int64
		ready, individual bool
		want              string
	}{
		{"ready", `["schedule","deleted"]`, 2, true, false, ""},
		{"pending CR synchronization", `["schedule"]`, 1, false, false, ""},
		{"record overrides pending global", `["schedule"]`, 1, false, true, ""},
		{"record overrides malformed global", `{`, 1, false, true, ""},
		{"native not in global inventory", `["other"]`, 1, true, false, ""},
		{"duplicates", `["schedule","schedule"]`, 2, true, false, "invalid adoption inventory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := getTestSQLite(t)
			prerequisiteSchema(t, db)
			prerequisiteFixture(t, db, "schedule", false)
			if !tc.individual {
				require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, map[string]any{"ID": recurringRunAdoptionSourceID + ":schedule"}).Error)
			}
			require.NoError(t, db.Create(&model.RecurringRunAdoption{ID: recurringRunAdoptionSourceID, AdoptedCount: tc.count, JobIDs: model.LargeText(tc.inventory), Ready: tc.ready}).Error)
			err := requireCompletedRecurringRunAdoption(db)
			if tc.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
}

func TestRecurringRunUpgradePrerequisiteChecksLaterPages(t *testing.T) {
	db := getTestSQLite(t)
	prerequisiteSchema(t, db)
	for i := 0; i < 101; i++ {
		prerequisiteFixture(t, db, fmt.Sprintf("schedule-%03d", i), false)
	}
	require.NoError(t, requireCompletedRecurringRunAdoption(db))
	require.NoError(t, db.Delete(&model.RecurringRunState{}, map[string]any{"JobUUID": "schedule-100"}).Error)
	require.ErrorContains(t, requireCompletedRecurringRunAdoption(db), "schedule-100")
}

func TestAutoMigratePreservesRecurringRunAdoption(t *testing.T) {
	db := getTestSQLite(t)
	require.NoError(t, autoMigrate(db))
	require.True(t, db.Migrator().HasTable(&model.RecurringRunAdoption{}))
	receipt := model.RecurringRunAdoption{ID: recurringRunAdoptionSourceID + ":deleted", AdoptedCount: 1, JobIDs: `["deleted"]`, Ready: false}
	require.NoError(t, db.Create(&receipt).Error)
	require.NoError(t, autoMigrate(db))
	var after model.RecurringRunAdoption
	require.NoError(t, db.Take(&after).Error)
	require.Equal(t, receipt, after)
}

func TestRecurringRunUpgradePrerequisiteNativeWithoutReceiptSchema(t *testing.T) {
	db := getTestSQLite(t)
	require.NoError(t, db.AutoMigrate(&prerequisiteJob{}, &model.RecurringRunState{}))
	require.NoError(t, db.Create(&prerequisiteJob{UUID: "native", Enabled: false}).Error)
	require.NoError(t, db.Create(&model.RecurringRunState{JobUUID: "native", RequestKey: "", PipelineVersionID: ""}).Error)
	require.NoError(t, requireCompletedRecurringRunAdoption(db))
	require.False(t, db.Migrator().HasTable(&model.RecurringRunAdoption{}))
	// A healthy native schedule cannot hide a disabled legacy record.
	require.NoError(t, db.Create(&prerequisiteJob{UUID: "legacy", Enabled: false}).Error)
	require.ErrorContains(t, requireCompletedRecurringRunAdoption(db), `"legacy" is missing scheduling state`)
}
