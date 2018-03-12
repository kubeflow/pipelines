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
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func initializePipelineDB() (PipelineStoreInterface, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	gormDB, _ := gorm.Open("mysql", db)
	return &PipelineStore{db: gormDB}, mock
}

func TestListPipelines(t *testing.T) {
	expectedPipelines := []pipelinemanager.Pipeline{
		{Metadata: &pipelinemanager.Metadata{ID: 1}, Name: "Pipeline123", PackageId: 1, Parameters: []pipelinemanager.Parameter{}},
		{Metadata: &pipelinemanager.Metadata{ID: 2}, Name: "Pipeline456", PackageId: 2, Parameters: []pipelinemanager.Parameter{}}}
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
	expectedPipeline := pipelinemanager.Pipeline{
		Metadata: &pipelinemanager.Metadata{ID: 1}, Name: "Pipeline123", PackageId: 1, Parameters: []pipelinemanager.Parameter{}}
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
	pipeline := pipelinemanager.Pipeline{Name: "Pipeline123"}
	ps, mock := initializePipelineDB()
	mock.ExpectExec("INSERT INTO `pipelines`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), pipeline.Name, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pipeline, err := ps.CreatePipeline(pipeline)
	assert.Nil(t, err, "Unexpected error creating pipeline")
	assert.Equal(t, uint(1), pipeline.ID, "ID should be assigned")
}

func TestCreatePipelineError(t *testing.T) {
	pipeline := pipelinemanager.Pipeline{Name: "Pipeline123"}
	ps, mock := initializePipelineDB()
	mock.ExpectExec("INSERT INTO `pipelines`").WillReturnError(errors.New("something"))

	_, err := ps.CreatePipeline(pipeline)
	assert.IsType(t, new(util.InternalError), err, "Expect create pipeline to return error")
}
