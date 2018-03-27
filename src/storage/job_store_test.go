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
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/jinzhu/gorm"
	"github.com/kataras/iris/core/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

var ct, st, ft time.Time

func initializeJobDB() (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	gormDB, _ := gorm.Open("mysql", db)
	return gormDB, mock
}

// TODO: Switch to using the FakeWorkflowClient.
type FakeGoodWorkflowClient struct {
}

func (FakeGoodWorkflowClient) Create(*v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-abcd",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}, nil
}

func (FakeGoodWorkflowClient) Get(name string, options v1.GetOptions) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-xyz",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}, nil
}

func (FakeGoodWorkflowClient) List(opts v1.ListOptions) (*v1alpha1.WorkflowList, error) {
	panic("implement me")
}

func (FakeGoodWorkflowClient) Update(*v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (FakeGoodWorkflowClient) Delete(name string, options *v1.DeleteOptions) error {
	panic("implement me")
}

func (FakeGoodWorkflowClient) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	panic("implement me")
}

func (FakeGoodWorkflowClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (FakeGoodWorkflowClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Workflow, err error) {
	panic("implement me")
}

type FakeBadWorkflowClient struct {
	FakeGoodWorkflowClient
}

func (FakeBadWorkflowClient) Create(*v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return nil, errors.New("some error")
}

func (FakeBadWorkflowClient) Get(name string, options v1.GetOptions) (*v1alpha1.Workflow, error) {
	return nil, errors.New("some error")
}

func init() {
	ct, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
	st, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
	ft, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
}

func TestListJobs(t *testing.T) {
	db, mock := initializeJobDB()
	jobsRow := sqlmock.NewRows([]string{"name", "pipeline_id"}).
		AddRow("abcd", 1).
		AddRow("efgh", 1)

	mock.ExpectQuery("SELECT (.*) FROM `jobs` WHERE").WillReturnRows(jobsRow)
	store := &JobStore{db: db}
	jobs, err := store.ListJobs(1)

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	expectedJobs := []message.Job{
		{Metadata: &message.Metadata{}, Name: "abcd", PipelineID: 1},
		{Metadata: &message.Metadata{}, Name: "efgh", PipelineID: 1}}

	assert.Equal(t, jobs, expectedJobs, "Unexpected Job parsed. Expect %v. Got %v", expectedJobs, jobs)
}

func TestListJobsError(t *testing.T) {
	db, mock := initializeJobDB()
	mock.ExpectQuery("SELECT (.*) FROM `jobs`").WillReturnError(errors.New("something"))
	store := &JobStore{db: db}
	_, err := store.ListJobs(1)
	assert.IsType(t, new(util.InternalError), err, "expect to throw an internal error")
}

func TestCreateJob(t *testing.T) {
	db, mock := initializeJobDB()
	mock.ExpectExec("INSERT INTO `jobs`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "artifact-passing-abcd", 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	store := NewJobStore(db, &FakeGoodWorkflowClient{}, util.NewFakeTimeForEpoch())
	job, err := store.CreateJob(1, &v1alpha1.Workflow{})

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
		return
	}
	wfExpect := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-abcd",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}
	jobExpect := message.JobDetail{Workflow: wfExpect}
	assert.Equal(t, job, jobExpect, "Unexpected Job parsed. Expect %v. Got %v", job, jobExpect)
}

func TestCreateJob_StoreMetadataError(t *testing.T) {
	db, mock := initializeJobDB()
	mock.ExpectExec("INSERT INTO `jobs`").WillReturnError(errors.New("something"))
	store := NewJobStore(db, &FakeGoodWorkflowClient{}, util.NewFakeTimeForEpoch())
	_, err := store.CreateJob(1, &v1alpha1.Workflow{})
	assert.IsType(t, new(util.InternalError), err, "expect to throw an internal error")
	assert.Contains(t, err.Error(), "Failed to store job metadata", "Get unexpected error")
}

func TestCreateJob_CreateWorkflowError(t *testing.T) {
	store := &JobStore{wfClient: &FakeBadWorkflowClient{}}
	_, err := store.CreateJob(1, &v1alpha1.Workflow{})
	assert.IsType(t, new(util.InternalError), err, "expect to throw an internal error")
	assert.Contains(t, err.Error(), "Failed to create job", "Get unexpected error")
}

func TestGetJob(t *testing.T) {
	db, mock := initializeJobDB()
	jobRow := sqlmock.NewRows([]string{"name", "pipeline_id"}).AddRow("abcd", 1)
	mock.ExpectQuery("SELECT (.*) FROM `jobs` WHERE").WillReturnRows(jobRow)
	store := &JobStore{db: db, wfClient: &FakeGoodWorkflowClient{}}
	job, err := store.GetJob(1, "job1")

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	wfExpect := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-xyz",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}
	jobExpect := message.JobDetail{Workflow: wfExpect}
	assert.Equal(t, job, jobExpect, "Unexpected Job parsed. Expect %v. Got %v", job, jobExpect)
}

func TestGetJob_JobNotExistError(t *testing.T) {
	db, mock := initializeJobDB()
	mock.ExpectQuery("SELECT (.*) FROM `jobs`").WillReturnError(errors.New("something"))
	store := &JobStore{db: db}
	_, err := store.GetJob(1, "job1")
	assert.IsType(t, new(util.ResourceNotFoundError), err, "expect not to find the job")
}

func TestGetJob_GetWorkflowError(t *testing.T) {
	db, mock := initializeJobDB()
	jobRow := sqlmock.NewRows([]string{"name", "pipeline_id"}).AddRow("abcd", 1)
	mock.ExpectQuery("SELECT (.*) FROM `jobs` WHERE").WillReturnRows(jobRow)

	store := &JobStore{db: db, wfClient: &FakeBadWorkflowClient{}}
	_, err := store.GetJob(1, "job1")
	assert.IsType(t, new(util.InternalError), err, "expect to throw an internal error")
}
