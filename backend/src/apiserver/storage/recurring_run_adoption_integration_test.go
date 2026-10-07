// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

// TestLegacyRecurringRunAdoptionProductionDatabases uses the same disposable
// MySQL databases and PostgreSQL schemas as the recurring-run concurrency suite.
func TestLegacyRecurringRunAdoptionProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, _ := recurringIntegrationDatabases(t, driver)
			testLegacyRecurringRunAdoptionProductionDatabase(t, dbs[0], driver)
		})
	}
}

func TestLegacyRecurringRunAdoptionReferenceBackedSnapshot(t *testing.T) {
	db := adoptionTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ResourceReference{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	testLegacyRecurringRunAdoptionProductionDatabase(t, sqlDB, "sqlite")
}

func testLegacyRecurringRunAdoptionProductionDatabase(t *testing.T, sqlDB *sql.DB, driver string) {
	t.Helper()
	db, err := OpenTransferDB(sqlDB, driver)
	require.NoError(t, err)
	require.True(t, db.Migrator().HasTable(&model.RecurringRunAdoption{}), "startup migration must create the adoption receipt table")

	a := adoptionTestCandidate(t, db, "adoption-a", true)
	b := adoptionTestCandidate(t, db, "adoption-b", false)
	current := adoptionTestCandidate(t, db, "adoption-current", true)
	require.NoError(t, db.Create(&current.State).Error)

	// Older rows can store their namespace only in resource references. The
	// adoption transaction compares the raw row, while resource validation uses
	// GetJob's resolved identity. Neither representation may rewrite the other.
	require.NoError(t, db.Model(&model.Job{}).Where(&model.Job{UUID: b.Job.UUID}).UpdateColumn("Namespace", "").Error)
	b.Job.Namespace = ""
	ref := model.ResourceReference{
		ResourceUUID: b.Job.UUID, ResourceType: model.JobResourceType,
		ReferenceUUID: "tenant", ReferenceType: model.NamespaceResourceType, Relationship: model.OwnerRelationship,
	}
	payload, err := json.Marshal(ref)
	require.NoError(t, err)
	ref.Payload = model.LargeText(payload)
	require.NoError(t, db.Create(&ref).Error)
	jobs := NewJobStore(sqlDB, util.NewFakeTimeForEpoch(), nil, dialect.NewDBDialect(driver))
	resolved, err := jobs.GetJob(b.Job.UUID)
	require.NoError(t, err)
	require.Equal(t, "tenant", resolved.Namespace)
	require.Empty(t, b.Job.Namespace)

	// The later sorted candidate fails after the first state insert. Both the
	// first insert and final receipt must remain absent on real SQL engines.
	changed := b
	changed.Job.ServiceAccount = "changed-after-inventory"
	_, err = ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{a, changed}, 1000)
	require.ErrorContains(t, err, "changed")
	var count int64
	require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	receipt, err := GetLegacyRecurringRunAdoption(db)
	require.NoError(t, err)
	require.Nil(t, receipt)

	receipt, err = ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{b, a}, 1000)
	require.NoError(t, err)
	require.Equal(t, &model.RecurringRunAdoption{
		ID: LegacyRecurringRunAdoptionID, AdoptedCount: 2, CompletedAt: 1000,
		JobIDs: `["adoption-a","adoption-b"]`, Ready: false,
	}, receipt)
	for _, candidate := range []RecurringRunAdoptionCandidate{a, b, current} {
		var storedJob model.Job
		require.NoError(t, db.Take(&storedJob, &model.Job{UUID: candidate.Job.UUID}).Error)
		require.Equal(t, candidate.Job, storedJob)
		state, err := jobs.GetRecurringRunState(candidate.Job.UUID)
		require.NoError(t, err)
		require.Equal(t, candidate.State, *state)
	}
	var storedRef model.ResourceReference
	require.NoError(t, db.Take(&storedRef, &model.ResourceReference{
		ResourceUUID: ref.ResourceUUID, ResourceType: ref.ResourceType, ReferenceType: ref.ReferenceType,
	}).Error)
	require.Equal(t, ref, storedRef)

	// A durable incomplete receipt and a ready receipt both ignore replacement
	// inputs, even after a scheduling state has been deleted.
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: a.Job.UUID}).Error)
	for _, ready := range []bool{false, true} {
		if ready {
			require.NoError(t, CompleteLegacyRecurringRunAdoption(db))
			require.NoError(t, CompleteLegacyRecurringRunAdoption(db))
			receipt.Ready = true
		}
		again, err := ApplyLegacyRecurringRunAdoption(db, []RecurringRunAdoptionCandidate{{}}, -1)
		require.NoError(t, err)
		require.Equal(t, receipt, again)
		require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&count).Error)
		require.EqualValues(t, 2, count)
	}
}
