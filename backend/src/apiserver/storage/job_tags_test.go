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

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestJobTagsLifecycle(t *testing.T) {
	db, dbDialect := NewFakeDBOrFatal()
	defer db.Close()
	testJobTagsLifecycle(t, db, dbDialect)
}

func testJobTagsLifecycle(t *testing.T, db *sql.DB, dbDialect dialect.DBDialect) {
	t.Helper()
	store := NewJobStore(db, util.NewFakeTimeForEpoch(), nil, dbDialect)
	initial := map[string]string{"team": "ml", "environment": "production", "empty": ""}
	job, err := store.CreateJob(&model.Job{UUID: "tagged", DisplayName: "nightly", Tags: initial})
	require.NoError(t, err)
	require.Equal(t, initial, job.Tags)
	job, err = store.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, initial, job.Tags)
	require.NoError(t, store.UpdateJobTags(job.UUID, nil))
	require.NoError(t, store.ChangeJobMode(job.UUID, true))
	job, err = store.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, initial, job.Tags)
	require.True(t, job.Enabled)
	workflow := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID(job.UUID), Name: "nightly"},
		Spec:       swfapi.ScheduledWorkflowSpec{Enabled: true},
	})
	require.NoError(t, store.UpdateJob(workflow))
	job, err = store.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, initial, job.Tags)

	replacement := map[string]string{"team": "platform"}
	require.NoError(t, store.UpdateJobTags(job.UUID, replacement))
	// Repeating the same update must work even within the same second on MySQL.
	require.NoError(t, store.UpdateJobTags(job.UUID, replacement))
	job, err = store.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, replacement, job.Tags)
	require.Equal(t, "nightly", job.DisplayName)
	require.True(t, job.Enabled)
	require.NoError(t, store.UpdateJobTags(job.UUID, map[string]string{}))
	job, err = store.GetJob(job.UUID)
	require.NoError(t, err)
	require.Empty(t, job.Tags)
	require.NoError(t, store.UpdateJobTags(job.UUID, replacement))
	require.NoError(t, store.DeleteJob(job.UUID))
	tags, err := queryTagsForEntities(db, dbDialect, "job_tags", "JobId", []string{job.UUID})
	require.NoError(t, err)
	require.Empty(t, tags)
	require.Error(t, store.UpdateJobTags(job.UUID, replacement))
}

func TestListJobsTagsPaginationAndIsolation(t *testing.T) {
	db, dbDialect := NewFakeDBOrFatal()
	defer db.Close()
	testListJobTagsPaginationAndIsolation(t, db, dbDialect)
}

func testListJobTagsPaginationAndIsolation(t *testing.T, db *sql.DB, dbDialect dialect.DBDialect) {
	t.Helper()
	store := NewJobStore(db, util.NewFakeTimeForEpoch(), nil, dbDialect)
	for _, job := range []*model.Job{
		{UUID: "1", DisplayName: "a", Namespace: "ns1", ExperimentId: "exp1", Tags: map[string]string{"team": "ml", "env": "prod"}},
		{UUID: "2", DisplayName: "b", Namespace: "ns1", ExperimentId: "exp1", Tags: map[string]string{"team": "ml", "env": "prod"}},
		{UUID: "3", DisplayName: "c", Namespace: "ns2", ExperimentId: "exp2", Tags: map[string]string{"team": "ml", "env": "prod"}},
		{UUID: "4", DisplayName: "d", Namespace: "ns1", ExperimentId: "exp1", Tags: map[string]string{"team": "ml", "env": "dev"}},
		{UUID: "5", DisplayName: "e", Namespace: "ns1", ExperimentId: "exp1"},
	} {
		_, err := store.CreateJob(job)
		require.NoError(t, err)
	}
	for _, ref := range []*model.ReferenceKey{
		{Type: model.NamespaceResourceType, ID: "ns1"},
		{Type: model.ExperimentResourceType, ID: "exp1"},
	} {
		opts, err := list.NewOptions(&model.Job{}, 1, "display_name", nil)
		require.NoError(t, err)
		context := &model.FilterContext{ReferenceKey: ref}
		tags := map[string]string{"team": "ml", "env": "prod"}
		jobs, total, token, err := store.ListJobs(context, opts, tags)
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Len(t, jobs, 1)
		require.Equal(t, "1", jobs[0].UUID)
		require.Equal(t, tags, jobs[0].Tags)
		require.NotEmpty(t, token)
		opts, err = list.NewOptionsFromToken(token, 1)
		require.NoError(t, err)
		jobs, total, token, err = store.ListJobs(context, opts, tags)
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Len(t, jobs, 1)
		require.Equal(t, "2", jobs[0].UUID)
		require.Empty(t, token)
	}
	ordinary, err := filter.New(&api.Filter{Predicates: []*api.Predicate{{Key: "display_name", Operation: api.Predicate_EQUALS, Value: &api.Predicate_StringValue{StringValue: "b"}}}})
	require.NoError(t, err)
	opts, err := list.NewOptions(&model.Job{}, 10, "", ordinary)
	require.NoError(t, err)
	jobs, total, _, err := store.ListJobs(&model.FilterContext{}, opts, map[string]string{"team": "ml"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, jobs, 1)
	require.Equal(t, "2", jobs[0].UUID)
	jobs, total, _, err = store.ListJobs(&model.FilterContext{}, opts, map[string]string{"missing": ""})
	require.NoError(t, err)
	require.Empty(t, jobs)
	require.Zero(t, total)
}

func TestJobTagsCreateRollback(t *testing.T) {
	db, dbDialect := NewFakeDBOrFatal()
	defer db.Close()
	store := NewJobStore(db, util.NewFakeTimeForEpoch(), nil, dbDialect)
	_, err := db.Exec(`CREATE TRIGGER reject_job_tag BEFORE INSERT ON job_tags BEGIN SELECT RAISE(ABORT, 'tag failure'); END`)
	require.NoError(t, err)
	_, err = store.CreateJob(&model.Job{UUID: "rollback", Tags: map[string]string{"team": "ml"}})
	require.Error(t, err)
	_, err = store.GetJob("rollback")
	require.ErrorContains(t, err, "not found")
}

func TestJobTagsUpdateRollback(t *testing.T) {
	db, dbDialect := NewFakeDBOrFatal()
	defer db.Close()
	store := NewJobStore(db, util.NewFakeTimeForEpoch(), nil, dbDialect)
	original := map[string]string{"team": "ml"}
	_, err := store.CreateJob(&model.Job{UUID: "rollback", Tags: original})
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER reject_tag_delete BEFORE DELETE ON job_tags BEGIN SELECT RAISE(ABORT, 'tag failure'); END`)
	require.NoError(t, err)
	require.Error(t, store.UpdateJobTags("rollback", map[string]string{"env": "prod"}))
	job, err := store.GetJob("rollback")
	require.NoError(t, err)
	require.Equal(t, original, job.Tags)
}
