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
	"errors"
	"ml/src/message"
	"ml/src/util"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func initializePipelineDB() (PipelineStoreInterface, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	gormDB, _ := gorm.Open("mysql", db)
	return NewPipelineStore(gormDB, util.NewFakeTimeForEpoch()), mock
}

func TestListPipelines(t *testing.T) {
	expectedPipelines := []message.Pipeline{
		{Metadata: &message.Metadata{ID: 1}, Name: "Pipeline123", PackageId: 1, Parameters: []message.Parameter{}},
		{Metadata: &message.Metadata{ID: 2}, Name: "Pipeline456", PackageId: 2, Parameters: []message.Parameter{}}}
	ps, mock := initializePipelineDB()
	pipelinesRow := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "package_id"}).
		AddRow(1, nil, nil, nil, "Pipeline123", "", 1).
		AddRow(2, nil, nil, nil, "Pipeline456", "", 2)
	mock.ExpectQuery("SELECT (.*) FROM `pipelines`").WillReturnRows(pipelinesRow)
	parametersRow := sqlmock.NewRows([]string{"name", "value", "owner_id", "owner_type"})
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnRows(parametersRow)
	pipelines, _ := ps.ListPipelines()

	assert.Equal(t, expectedPipelines, pipelines, "Got unexpected pipelines")
}

func TestListPipelinesError(t *testing.T) {
	ps, mock := initializePipelineDB()
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnError(errors.New("something"))
	_, err := ps.ListPipelines()

	assert.IsType(t, new(util.InternalError), err, "Expect to list pipeline to return error")
}

func TestGetPipeline(t *testing.T) {
	expectedPipeline := message.Pipeline{
		Metadata: &message.Metadata{ID: 1}, Name: "Pipeline123", PackageId: 1, Parameters: []message.Parameter{}}
	ps, mock := initializePipelineDB()
	pipelinesRow := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "package_id"}).
		AddRow(1, nil, nil, nil, "Pipeline123", "", 1)
	mock.ExpectQuery("SELECT (.*) FROM `pipelines`").WillReturnRows(pipelinesRow)
	parametersRow := sqlmock.NewRows([]string{"name", "value", "owner_id", "owner_type"})
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnRows(parametersRow)

	pipeline, _ := ps.GetPipeline(123)

	assert.Equal(t, expectedPipeline, pipeline, "Got unexpected pipeline")
}

func TestGetPipelineError(t *testing.T) {
	ps, mock := initializePipelineDB()
	mock.ExpectQuery("SELECT (.*) FROM `parameters`").WillReturnError(errors.New("something"))
	_, err := ps.GetPipeline(123)
	assert.IsType(t, new(util.ResourceNotFoundError), err, "Expect get pipeline to return error")
}

func TestCreatePipeline(t *testing.T) {
	pipeline := message.Pipeline{Name: "Pipeline123"}
	ps, mock := initializePipelineDB()
	mock.ExpectExec("INSERT INTO `pipelines`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), pipeline.Name, sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), true, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pipeline, err := ps.CreatePipeline(pipeline)
	assert.Nil(t, err, "Unexpected error creating pipeline")
	assert.Equal(t, uint(1), pipeline.ID, "ID should be assigned")
}

func TestCreatePipelineError(t *testing.T) {
	pipeline := message.Pipeline{Name: "Pipeline123"}
	ps, mock := initializePipelineDB()
	mock.ExpectExec("INSERT INTO `pipelines`").WillReturnError(errors.New("something"))

	_, err := ps.CreatePipeline(pipeline)
	assert.IsType(t, new(util.InternalError), err, "Expect create pipeline to return error")
}

func TestGetPipelineAndLatestJobIteratorPipelineWithoutJob(t *testing.T) {
	store, err := NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	pipeline1 := message.Pipeline{
		Name:      "MY_PIPELINE_1",
		PackageId: 123,
		Schedule:  "1 0 * * *"}

	pipeline2 := message.Pipeline{
		Name:      "MY_PIPELINE_2",
		PackageId: 123,
		Schedule:  "1 0 * * 1"}

	workflow1 := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_1",
		},
	}

	workflow2 := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_2",
		},
	}

	store.PipelineStore.CreatePipeline(pipeline1)
	store.PipelineStore.CreatePipeline(pipeline2)
	store.JobStore.CreateJob(1, workflow1)
	store.JobStore.CreateJob(1, workflow2)

	// Checking the first row, which does not have a job.
	iterator, err := store.PipelineStore.GetPipelineAndLatestJobIterator()

	assert.Nil(t, err)
	assert.True(t, iterator.Next())

	result, err := iterator.Get()
	assert.Nil(t, err)

	pipelineID := "2"
	pipelineEnabled := true
	enabledSec := int64(2)

	expected := &PipelineAndLatestJob{
		PipelineID:             &pipelineID,
		PipelineName:           &pipeline2.Name,
		PipelineSchedule:       &pipeline2.Schedule,
		JobName:                nil,
		JobScheduledAtInSec:    nil,
		PipelineEnabled:        &pipelineEnabled,
		PipelineEnabledAtInSec: &enabledSec,
	}

	assert.Equal(t, expected, result)

	// Checking the second row, which has a job.
	assert.True(t, iterator.Next())

	result, err = iterator.Get()
	assert.Nil(t, err)

	pipelineID = "1"
	pipelineEnabled = true
	enabledSec = int64(1)
	scheduledSec := int64(4)

	expected = &PipelineAndLatestJob{
		PipelineID:             &pipelineID,
		PipelineName:           &pipeline1.Name,
		PipelineSchedule:       &pipeline1.Schedule,
		JobName:                &workflow2.Name,
		JobScheduledAtInSec:    &scheduledSec,
		PipelineEnabled:        &pipelineEnabled,
		PipelineEnabledAtInSec: &enabledSec,
	}

	assert.Equal(t, expected, result)

	// Checking that there are no rows left.
	assert.False(t, iterator.Next())

}

func TestGetPipelineAndLatestJobIteratorPipelineWithoutSchedule(t *testing.T) {
	store, err := NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	pipeline1 := message.Pipeline{
		Name:      "MY_PIPELINE_1",
		PackageId: 123,
		Schedule:  "1 0 * * *"}

	pipeline2 := message.Pipeline{
		Name:      "MY_PIPELINE_2",
		PackageId: 123,
		Schedule:  ""}

	workflow1 := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_1",
		},
	}

	workflow2 := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_2",
		},
	}

	store.PipelineStore.CreatePipeline(pipeline1)
	store.PipelineStore.CreatePipeline(pipeline2)
	store.JobStore.CreateJob(1, workflow1)
	store.JobStore.CreateJob(1, workflow2)

	// Checking the first row, which does not have a job.
	iterator, err := store.PipelineStore.GetPipelineAndLatestJobIterator()

	assert.Nil(t, err)
	assert.True(t, iterator.Next())

	result, err := iterator.Get()
	assert.Nil(t, err)

	pipelineID := "1"
	pipelineEnabled := true
	enabledSec := int64(1)
	scheduledSec := int64(4)

	expected := &PipelineAndLatestJob{
		PipelineID:             &pipelineID,
		PipelineName:           &pipeline1.Name,
		PipelineSchedule:       &pipeline1.Schedule,
		JobName:                &workflow2.Name,
		JobScheduledAtInSec:    &scheduledSec,
		PipelineEnabled:        &pipelineEnabled,
		PipelineEnabledAtInSec: &enabledSec,
	}

	assert.Equal(t, expected, result)

	// Checking that there are no rows left.
	assert.False(t, iterator.Next())

}
