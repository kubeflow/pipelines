// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"database/sql"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Only historical-record tests seed the deprecated resource_references table.
func seedLegacyResourceReferences(t *testing.T, db *sql.DB, dbDialect dialect.DBDialect, refs ...*model.ResourceReference) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, NewResourceReferenceStore(db, nil, dbDialect).CreateResourceReferences(tx, refs))
	require.NoError(t, tx.Commit())
}

func TestCreateRun_DoesNotWriteLegacyResourceReferences(t *testing.T) {
	db, dbDialect, store := initializeRunStore()
	defer db.Close()

	var count int
	q := dbDialect.QuoteIdentifier
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+q("resource_references")).Scan(&count))
	assert.Zero(t, count)
	for _, id := range []string{"1", "2", "3"} {
		run, err := store.GetRun(id, false)
		require.NoError(t, err)
		assert.NotEmpty(t, run.ExperimentId)
		assert.NotEmpty(t, run.Namespace)
	}
}

func TestCreateJob_DoesNotWriteLegacyResourceReferences(t *testing.T) {
	db, dbDialect, store := initializeDBAndStore()
	defer db.Close()

	var count int
	q := dbDialect.QuoteIdentifier
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+q("resource_references")).Scan(&count))
	assert.Zero(t, count)
	for _, id := range []string{"1", "2"} {
		job, err := store.GetJob(id)
		require.NoError(t, err)
		assert.NotEmpty(t, job.ExperimentId)
		assert.NotEmpty(t, job.Namespace)
		assert.NotEmpty(t, job.PipelineId)
	}
}

func TestRunStore_ReadsHistoricalOwnership(t *testing.T) {
	db, dbDialect, store := initializeRunStore()
	defer db.Close()
	seedLegacyResourceReferences(t, db, dbDialect,
		&model.ResourceReference{ResourceUUID: "1", ResourceType: model.RunResourceType,
			ReferenceUUID: defaultFakeExpId, ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship},
		&model.ResourceReference{ResourceUUID: "1", ResourceType: model.RunResourceType,
			ReferenceUUID: "legacy-ns", ReferenceType: model.NamespaceResourceType, Relationship: model.OwnerRelationship},
	)
	q := dbDialect.QuoteIdentifier
	query, args, err := dbDialect.QueryBuilder().Update(q("run_details")).
		SetMap(sq.Eq{q("ExperimentUUID"): "", q("Namespace"): ""}).Where(sq.Eq{q("UUID"): "1"}).ToSql()
	require.NoError(t, err)
	_, err = db.Exec(query, args...)
	require.NoError(t, err)

	run, err := store.GetRun("1", false)
	require.NoError(t, err)
	assert.Equal(t, defaultFakeExpId, run.ExperimentId)
	assert.Equal(t, "legacy-ns", run.Namespace)
	opts, err := list.NewOptions(&model.Run{}, 10, "", nil)
	require.NoError(t, err)
	runs, _, _, err := store.ListRuns(&model.FilterContext{}, opts, false)
	require.NoError(t, err)
	for _, run := range runs {
		if run.UUID == "1" {
			assert.Equal(t, defaultFakeExpId, run.ExperimentId)
			assert.Equal(t, "legacy-ns", run.Namespace)
			return
		}
	}
	t.Fatal("historical run missing from list")
}

func TestJobStore_ReadsHistoricalOwnership(t *testing.T) {
	db, dbDialect, store := initializeDBAndStore()
	defer db.Close()
	seedLegacyResourceReferences(t, db, dbDialect,
		&model.ResourceReference{ResourceUUID: "1", ResourceType: model.JobResourceType,
			ReferenceUUID: defaultFakeExpId, ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship},
		&model.ResourceReference{ResourceUUID: "1", ResourceType: model.JobResourceType,
			ReferenceUUID: "legacy-ns", ReferenceType: model.NamespaceResourceType, Relationship: model.OwnerRelationship},
	)
	q := dbDialect.QuoteIdentifier
	query, args, err := dbDialect.QueryBuilder().Update(q("jobs")).
		SetMap(sq.Eq{q("ExperimentUUID"): "", q("Namespace"): ""}).Where(sq.Eq{q("UUID"): "1"}).ToSql()
	require.NoError(t, err)
	_, err = db.Exec(query, args...)
	require.NoError(t, err)

	job, err := store.GetJob("1")
	require.NoError(t, err)
	assert.Equal(t, defaultFakeExpId, job.ExperimentId)
	assert.Equal(t, "legacy-ns", job.Namespace)
	opts, err := list.NewOptions(&model.Job{}, 10, "", nil)
	require.NoError(t, err)
	jobs, _, _, err := store.ListJobs(&model.FilterContext{}, opts)
	require.NoError(t, err)
	for _, job := range jobs {
		if job.UUID == "1" {
			assert.Equal(t, defaultFakeExpId, job.ExperimentId)
			assert.Equal(t, "legacy-ns", job.Namespace)
			return
		}
	}
	t.Fatal("historical recurring run missing from list")
}
