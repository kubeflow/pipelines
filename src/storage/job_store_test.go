// Copyright 2018 Google LLC
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
	"ml/src/message"
	"ml/src/util"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultScheduledTimeInSec = 10
)

func createWorkflow(name string) *v1alpha1.Workflow {
	return &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: name},
		Status:     v1alpha1.WorkflowStatus{Phase: "Pending"}}
}

func TestCreateJob(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	wf1 := createWorkflow("wf1")
	jobDetail, err := store.JobStore.CreateJob(1, wf1, defaultScheduledTimeInSec)

	jobExpected := message.Job{
		Metadata:         &message.Metadata{ID: 1},
		Name:             jobDetail.Job.Name,
		ScheduledAtInSec: defaultScheduledTimeInSec,
		PipelineID:       1,
	}

	wfExpected := createWorkflow(jobDetail.Job.Name)

	jobDetailExpect := message.JobDetail{
		Workflow: wfExpected,
		Job:      &jobExpected}

	assert.Nil(t, err)
	assert.Equal(t, jobDetailExpect, *jobDetail, "Unexpected Job parsed.")

	var job message.Job
	queryJob(store.DB, 1, wf1.Name, &job)
	assert.Equal(t, jobExpected, job)
}

func TestCreateJob_CreateWorkflowError(t *testing.T) {
	store := &JobStore{wfClient: &FakeBadWorkflowClient{}}
	_, err := store.CreateJob(1, createWorkflow("wf1"), defaultScheduledTimeInSec)
	assert.IsType(t, new(util.InternalError), err, "Expected to throw an internal error")
	assert.Contains(t, err.Error(), "Failed to create job", "Got unexpected error")
}

func TestCreateJob_StoreMetadataError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB.Close()

	_, err := store.JobStore.CreateJob(1, &v1alpha1.Workflow{}, defaultScheduledTimeInSec)
	assert.IsType(t, new(util.InternalError), err, "Expected to throw an internal error")
	assert.Contains(t, err.Error(), "Failed to store job metadata", "Get unexpected error")
}

func TestListJobs(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	jobDetail, err := store.JobStore.CreateJob(1, createWorkflow("wf1"), defaultScheduledTimeInSec)
	store.JobStore.CreateJob(2, createWorkflow("wf2"), defaultScheduledTimeInSec)

	jobsExpected := []message.Job{
		{Metadata: &message.Metadata{ID: 1},
			Name:             jobDetail.Job.Name,
			ScheduledAtInSec: defaultScheduledTimeInSec,
			PipelineID:       1,
		}}
	jobs, err := store.JobStore.ListJobs(1)
	assert.Nil(t, err)
	assert.Equal(t, jobsExpected, jobs, "Unexpected Job listed.")

	jobs, err = store.JobStore.ListJobs(3)
	assert.Empty(t, jobs)
}

func TestListJobsError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB.Close()
	_, err := store.JobStore.ListJobs(1)
	assert.IsType(t, new(util.InternalError), err, "Expected to throw an internal error")
}

func TestGetJob(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	wf1 := createWorkflow("wf1")
	createdJobDetail, err := store.JobStore.CreateJob(1, wf1, defaultScheduledTimeInSec)
	assert.Nil(t, err)
	jobDetailExpect := message.JobDetail{
		Workflow: wf1,
		Job: &message.Job{
			Metadata:         &message.Metadata{ID: 1},
			Name:             createdJobDetail.Job.Name,
			ScheduledAtInSec: defaultScheduledTimeInSec,
			PipelineID:       1,
		}}

	jobDetail, err := store.JobStore.GetJob(1, wf1.Name)
	jobDetail.Job.Metadata = &message.Metadata{ID: jobDetail.Job.ID}
	assert.Nil(t, err)
	assert.Equal(t, jobDetailExpect, *jobDetail)
}

func TestGetJob_NotFoundError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	_, err := store.JobStore.GetJob(1, "wf1")
	assert.IsType(t, new(util.ResourceNotFoundError), err, "Expected not to find the job")
}

func TestGetJob_InternalError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	wf1 := createWorkflow("wf1")
	store.JobStore.CreateJob(1, wf1, defaultScheduledTimeInSec)
	store.DB.Close()

	_, err := store.JobStore.GetJob(1, wf1.Name)
	assert.IsType(t, new(util.InternalError), err, "Expected get job to return internal error")
}

func TestGetJob_GetWorkflowError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	wf1 := createWorkflow("wf1")
	store.JobStore.CreateJob(1, wf1, defaultScheduledTimeInSec)

	jobStore := NewJobStore(store.DB, &FakeBadWorkflowClient{}, util.NewFakeTimeForEpoch())
	_, err := jobStore.GetJob(1, wf1.Name)
	assert.IsType(t, new(util.InternalError), err, "Expected to throw an internal error")
}
