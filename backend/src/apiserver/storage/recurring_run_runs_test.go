// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestRecurringRunInventoryAndCapacityResolveLegacyReferences(t *testing.T) {
	for _, tc := range []struct {
		name, rawJob, referenceJob  string
		resourceType, referenceType model.ResourceType
		matches                     bool
	}{
		{"reference fallback", "", "1", model.RunResourceType, model.JobResourceType, true},
		{"explicit matching job", "1", "2", model.RunResourceType, model.JobResourceType, true},
		{"explicit conflicting job", "2", "1", model.RunResourceType, model.JobResourceType, false},
		{"foreign reference", "", "2", model.RunResourceType, model.JobResourceType, false},
		{"wrong resource type", "", "1", model.JobResourceType, model.JobResourceType, false},
		{"wrong reference type", "", "1", model.RunResourceType, model.ExperimentResourceType, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, d, jobs := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			orm, err := OpenTransferDB(db, "sqlite")
			require.NoError(t, err)
			run := &model.Run{UUID: "legacy-run", DisplayName: "legacy-request", RecurringRunId: tc.rawJob,
				RunDetails: model.RunDetails{Conditions: "Running"}}
			require.NoError(t, orm.Create(run).Error)
			require.NoError(t, orm.Create(&model.ResourceReference{
				ResourceUUID: run.UUID, ResourceType: tc.resourceType,
				ReferenceUUID: tc.referenceJob, ReferenceType: tc.referenceType,
			}).Error)
			runs := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
			ids, err := runs.ListRunIDsForRecurringRun("1")
			require.NoError(t, err)
			replayID, err := runs.GetRunByRecurringRunIDAndDisplayName("1", run.DisplayName)
			require.NoError(t, err)
			_, err = jobs.ClaimRecurringRun("1", "next-tick", 0, 100, 110, "")
			if tc.matches {
				require.Equal(t, []string{run.UUID}, ids)
				require.Equal(t, run.UUID, replayID)
				require.ErrorContains(t, err, "maximum concurrency")
			} else {
				require.Empty(t, ids)
				require.Empty(t, replayID)
				require.NoError(t, err)
			}
		})
	}
}

func TestRecurringRunAssociationQueryDialects(t *testing.T) {
	for _, name := range []string{"mysql", "pgx", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			d := dialect.NewDBDialect(name)
			q := d.QuoteIdentifier
			query, args, err := d.FinalizeSelect(sq.Select("COUNT(*)").From(q("run_details")).
				Where(recurringRunAssociation(q, "job")))
			require.NoError(t, err)
			require.Contains(t, query, q("run_details")+"."+q("JobUUID"))
			require.Contains(t, query, q("resource_references")+"."+q("ResourceUUID"))
			require.Equal(t, []any{"job", "", model.JobResourceType, "job", model.RunResourceType}, args)
			if name == "pgx" {
				require.NotContains(t, query, "?")
				require.Contains(t, query, "$5")
			} else {
				require.NotContains(t, query, "$1")
			}
		})
	}
}
