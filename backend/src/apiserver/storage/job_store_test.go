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
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultScheduledAtInSec = 10
	defaultCreatedAtInSec   = 20
)

func initializeJobDB() (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	gormDB, _ := gorm.Open("sqlite3", db)
	return gormDB, mock
}

func createWorkflow(name string) *v1alpha1.Workflow {
	return &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: name},
		Status:     v1alpha1.WorkflowStatus{Phase: "Pending"}}
}

func TestCreateJob(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())

	wf1 := createWorkflow("wf1")

	jobExpected := model.Job{
		CreatedAtInSec:   defaultCreatedAtInSec,
		Name:             wf1.Name,
		ScheduledAtInSec: defaultScheduledAtInSec,
		Status:           model.JobExecutionPending,
		UpdatedAtInSec:   defaultCreatedAtInSec,
		PipelineID:       1,
	}
	wfExpected := createWorkflow(wf1.Name)
	jobDetailExpect := model.JobDetail{
		Workflow: wfExpected,
		Job:      &jobExpected}
	jobDetail, err := jobStore.CreateJob(1, wf1, defaultScheduledAtInSec, defaultCreatedAtInSec)

	assert.Nil(t, err)
	assert.Equal(t, jobDetailExpect, *jobDetail, "Unexpected Job parsed.")

	job, err := getJobMetadata(db, 1, wf1.Name)
	assert.Equal(t, jobExpected, *job)
}

func TestCreateJob_CreateWorkflowFailed(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, &FakeBadWorkflowClient{}, util.NewFakeTimeForEpoch())

	wf1 := createWorkflow("wf1")

	jobDetail, err := jobStore.CreateJob(1, wf1, defaultScheduledAtInSec, defaultCreatedAtInSec)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "Internal Server Error")
	assert.Nil(t, jobDetail)

	job, err := getJobMetadata(db, 1, wf1.Name)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "Job wf1 not found.")
	assert.Nil(t, job)
}

func TestCreateJob_CreateMetadataError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())
	db.Close()

	_, err := jobStore.CreateJob(1, &v1alpha1.Workflow{},
		defaultScheduledAtInSec, defaultCreatedAtInSec)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "Internal Server Error")
	assert.Contains(t, err.(*util.UserError).Error(), "Failed to store job metadata")
}

func TestCreateJob_UpdateMetadataFailed(t *testing.T) {
	db, mock := initializeJobDB()
	mock.ExpectExec("INSERT INTO \"jobs\"").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE jobs").WillReturnError(errors.New("something"))

	store := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())
	wf1 := createWorkflow("wf1")
	jobExpected := model.Job{
		CreatedAtInSec:   defaultCreatedAtInSec,
		UpdatedAtInSec:   defaultCreatedAtInSec,
		Name:             wf1.Name,
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: defaultScheduledAtInSec,
		PipelineID:       1,
	}
	wfExpected := createWorkflow(wf1.Name)
	jobDetailExpect := model.JobDetail{
		Workflow: wfExpected,
		Job:      &jobExpected}

	jobDetail, err := store.CreateJob(1, wf1, defaultScheduledAtInSec, defaultCreatedAtInSec)

	assert.Nil(t, err)
	assert.Equal(t, jobDetailExpect, *jobDetail)
}

func TestListJobs_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())

	jobDetail, err := jobStore.CreateJob(1, createWorkflow("wf1"),
		defaultScheduledAtInSec, defaultCreatedAtInSec)
	jobDetail2, err := jobStore.CreateJob(1, createWorkflow("wf2"),
		defaultScheduledAtInSec, defaultCreatedAtInSec)
	jobStore.CreateJob(2, createWorkflow("wf3"),
		defaultScheduledAtInSec, defaultCreatedAtInSec)

	expectedFirstPageJobs := []model.Job{
		{
			CreatedAtInSec:   defaultCreatedAtInSec,
			UpdatedAtInSec:   defaultCreatedAtInSec,
			Name:             jobDetail.Job.Name,
			Status:           model.JobExecutionPending,
			ScheduledAtInSec: defaultScheduledAtInSec,
			PipelineID:       1,
		}}
	expectedSecondPageJobs := []model.Job{
		{
			CreatedAtInSec:   defaultCreatedAtInSec,
			UpdatedAtInSec:   defaultCreatedAtInSec,
			Name:             jobDetail2.Job.Name,
			Status:           model.JobExecutionPending,
			ScheduledAtInSec: defaultScheduledAtInSec,
			PipelineID:       1,
		}}
	jobs, nextPageToken, err := jobStore.ListJobs(1, "", 1, model.GetJobTablePrimaryKeyColumn())
	assert.Nil(t, err)
	assert.Equal(t, expectedFirstPageJobs, jobs, "Unexpected Job listed.")
	assert.NotEmpty(t, nextPageToken)

	jobs, nextPageToken, err = jobStore.ListJobs(1, nextPageToken, 1, model.GetJobTablePrimaryKeyColumn())
	assert.Nil(t, err)
	assert.Equal(t, expectedSecondPageJobs, jobs, "Unexpected Job listed.")
	assert.Empty(t, nextPageToken)
}

func TestListJobs_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())
	jobDetail, err := jobStore.CreateJob(1, createWorkflow("wf1"),
		defaultScheduledAtInSec, defaultCreatedAtInSec)
	jobDetail2, err := jobStore.CreateJob(1, createWorkflow("wf2"),
		defaultScheduledAtInSec, defaultCreatedAtInSec)
	expectedJobs := []model.Job{
		{
			CreatedAtInSec:   defaultCreatedAtInSec,
			UpdatedAtInSec:   defaultCreatedAtInSec,
			Name:             jobDetail.Job.Name,
			Status:           model.JobExecutionPending,
			ScheduledAtInSec: defaultScheduledAtInSec,
			PipelineID:       1,
		},
		{
			CreatedAtInSec:   defaultCreatedAtInSec,
			UpdatedAtInSec:   defaultCreatedAtInSec,
			Name:             jobDetail2.Job.Name,
			Status:           model.JobExecutionPending,
			ScheduledAtInSec: defaultScheduledAtInSec,
			PipelineID:       1,
		}}
	jobs, nextPageToken, err := jobStore.ListJobs(1, "", 10, model.GetJobTablePrimaryKeyColumn())
	assert.Nil(t, err)
	assert.Equal(t, expectedJobs, jobs, "Unexpected Job listed.")
	assert.Empty(t, nextPageToken)
}

func TestListJobsError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())
	db.Close()
	_, _, err := jobStore.ListJobs(1, "", 10, model.GetJobTablePrimaryKeyColumn())
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
}

func TestGetJob(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())
	wf1 := createWorkflow("wf1")
	createdJobDetail, err := jobStore.CreateJob(1, wf1,
		defaultScheduledAtInSec, defaultCreatedAtInSec)
	assert.Nil(t, err)
	jobDetailExpect := model.JobDetail{
		Workflow: wf1,
		Job: &model.Job{
			CreatedAtInSec:   defaultCreatedAtInSec,
			UpdatedAtInSec:   defaultCreatedAtInSec,
			Status:           model.JobExecutionPending,
			Name:             createdJobDetail.Job.Name,
			ScheduledAtInSec: defaultScheduledAtInSec,
			PipelineID:       1,
		}}

	jobDetail, err := jobStore.GetJob(1, wf1.Name)
	assert.Nil(t, err)
	assert.Equal(t, jobDetailExpect, *jobDetail)
}

func TestGetJob_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())

	_, err := jobStore.GetJob(1, "wf1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected not to find the job")
}

func TestGetJob_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())
	wf1 := createWorkflow("wf1")
	jobStore.CreateJob(1, wf1,
		defaultScheduledAtInSec, defaultCreatedAtInSec)
	db.Close()

	_, err := jobStore.GetJob(1, wf1.Name)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get job to return internal error")
}

func TestGetJob_GetWorkflowError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, NewWorkflowClientFake(), util.NewFakeTimeForEpoch())
	wf1 := createWorkflow("wf1")
	jobStore.CreateJob(1, wf1,
		defaultScheduledAtInSec, defaultCreatedAtInSec)

	jobStore = NewJobStore(db, &FakeBadWorkflowClient{}, util.NewFakeTimeForEpoch())
	_, err := jobStore.GetJob(1, wf1.Name)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
}
