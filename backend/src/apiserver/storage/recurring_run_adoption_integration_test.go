// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"gorm.io/gorm"

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

func TestOnlineRecurringRunAdoptionProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			connections := make([]*gorm.DB, len(dbs))
			for i, sqlDB := range dbs {
				var err error
				connections[i], err = OpenTransferDB(sqlDB, driver)
				require.NoError(t, err)
			}
			db := connections[0]
			candidate := adoptionTestCandidate(t, db, "online-a", true)
			errs := recurringConcurrent(len(connections), func(i int) error { return ApplyLegacyRecurringRunAdoptionForJob(connections[i], candidate, 1000) })
			for _, err := range errs {
				require.NoError(t, err)
			}
			var count int64
			require.NoError(t, db.Model(&model.RecurringRunState{}).Where(&model.RecurringRunState{JobUUID: candidate.Job.UUID}).Count(&count).Error)
			require.EqualValues(t, 1, count)
			pending, err := ListPendingLegacyRecurringRunAdoptions(db, "", 100)
			require.NoError(t, err)
			require.Len(t, pending, 1)
			store := NewJobStore(dbs[0], util.NewFakeTimeForEpoch(), nil, d)
			_, err = store.ClaimRecurringRun(candidate.Job.UUID, "next", 7, 580, 590, "version")
			require.ErrorContains(t, err, "synchronization is pending")
			require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, candidate.Job.UUID, func(*model.Job, *model.RecurringRunState) error { return nil }))
			_, err = store.ClaimRecurringRun(candidate.Job.UUID, "next", 7, 580, 590, "version")
			require.NoError(t, err)
			// Real driver upsert and pending query, with durable recovery after a failed
			// ready-receipt update following the simulated Kubernetes side effect.
			require.NoError(t, SetRecurringRunModeForReconciliation(db, candidate.Job, false, 1001))
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register("fail_online_receipt", func(tx *gorm.DB) {
				if tx.Statement.Table == "recurring_run_adoptions" {
					tx.AddError(fmt.Errorf("interrupted receipt acknowledgment"))
				}
			}))
			err = SynchronizeLegacyRecurringRunAdoption(db, candidate.Job.UUID, func(job *model.Job, state *model.RecurringRunState) error {
				require.False(t, job.Enabled)
				require.True(t, state.Pending)
				return nil
			})
			require.ErrorContains(t, err, "interrupted receipt acknowledgment")
			require.NoError(t, db.Callback().Update().Remove("fail_online_receipt"))
			receipt, err := GetLegacyRecurringRunAdoptionForJob(db, candidate.Job.UUID)
			require.NoError(t, err)
			require.False(t, receipt.Ready)
			pending, err = ListPendingLegacyRecurringRunAdoptions(db, "", 100)
			require.NoError(t, err)
			require.Len(t, pending, 1)
			require.NoError(t, SynchronizeLegacyRecurringRunAdoption(db, candidate.Job.UUID, func(job *model.Job, _ *model.RecurringRunState) error { require.False(t, job.Enabled); return nil }))
			pending, err = ListPendingLegacyRecurringRunAdoptions(db, "", 100)
			require.NoError(t, err)
			require.Empty(t, pending)
			// Native initialization seals provenance in its original SQL transaction.
			native, err := store.CreateJob(&model.Job{UUID: "online-native", K8SName: "native", Namespace: "test", Enabled: true})
			require.NoError(t, err)
			receipt, err = GetLegacyRecurringRunAdoptionForJob(db, native.UUID)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.True(t, receipt.Ready)
			require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: native.UUID}).Error)
			require.ErrorContains(t, ApplyLegacyRecurringRunAdoptionForJob(db, RecurringRunAdoptionCandidate{Job: *native, State: model.RecurringRunState{JobUUID: native.UUID}}, 1002), "refusing to reseed")
		})
	}
}
