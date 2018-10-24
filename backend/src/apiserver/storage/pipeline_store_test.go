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

	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	fakeUUID      = "123e4567-e89b-12d3-a456-426655440000"
	fakeUUIDTwo   = "123e4567-e89b-12d3-a456-426655440001"
	fakeUUIDThree = "123e4567-e89b-12d3-a456-426655440002"
	fakeUUIDFour  = "123e4567-e89b-12d3-a456-426655440003"
)

func createPipeline(name string) *model.Pipeline {
	return &model.Pipeline{Name: name, Parameters: `[{"Name": "param1"}]`, Status: model.PipelineReady}
}

func TestListPipelines_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipeline(&model.Pipeline{Name: "pipeline3", Status: model.PipelineCreating})
	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline2 := model.Pipeline{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pipelinesExpected := []model.Pipeline{expectedPipeline1, expectedPipeline2}

	pipelines, nextPageToken, err := pipelineStore.ListPipelines(&common.PaginationContext{
		PageSize:        10,
		KeyFieldName:    model.GetPipelineTablePrimaryKeyColumn(),
		SortByFieldName: model.GetPipelineTablePrimaryKeyColumn(),
		IsDesc:          false,
	})
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelines_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline2"))
	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline4 := model.Pipeline{
		UUID:           fakeUUIDFour,
		CreatedAtInSec: 4,
		Name:           "pipeline2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pipelinesExpected := []model.Pipeline{expectedPipeline1, expectedPipeline4}
	pipelines, nextPageToken, err := pipelineStore.ListPipelines(&common.PaginationContext{
		PageSize:        2,
		KeyFieldName:    model.GetPipelineTablePrimaryKeyColumn(),
		SortByFieldName: "Name",
		IsDesc:          false,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline2 := model.Pipeline{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline3",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline3 := model.Pipeline{
		UUID:           fakeUUIDThree,
		CreatedAtInSec: 3,
		Name:           "pipeline4",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pipelinesExpected2 := []model.Pipeline{expectedPipeline2, expectedPipeline3}

	pipelines, nextPageToken, err = pipelineStore.ListPipelines(
		&common.PaginationContext{
			Token: &common.Token{
				SortByFieldValue: "pipeline3",
				// The value of the key field of the next row to be returned.
				KeyFieldValue: fakeUUIDTwo},
			PageSize:        2,
			KeyFieldName:    model.GetPipelineTablePrimaryKeyColumn(),
			SortByFieldName: "Name",
			IsDesc:          false,
		})
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline2"))

	expectedPipeline2 := model.Pipeline{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline3",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline3 := model.Pipeline{
		UUID:           fakeUUIDThree,
		CreatedAtInSec: 3,
		Name:           "pipeline4",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pipelinesExpected := []model.Pipeline{expectedPipeline3, expectedPipeline2}
	pipelines, nextPageToken, err := pipelineStore.ListPipelines(&common.PaginationContext{
		PageSize:        2,
		KeyFieldName:    model.GetPipelineTablePrimaryKeyColumn(),
		SortByFieldName: "Name",
		IsDesc:          true,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline4 := model.Pipeline{
		UUID:           fakeUUIDFour,
		CreatedAtInSec: 4,
		Name:           "pipeline2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pipelinesExpected2 := []model.Pipeline{expectedPipeline4, expectedPipeline1}
	pipelines, nextPageToken, err = pipelineStore.ListPipelines(
		&common.PaginationContext{
			Token: &common.Token{
				SortByFieldValue: "pipeline2",
				// The value of the key field of the next row to be returned.
				KeyFieldValue: fakeUUIDFour},
			PageSize:        2,
			KeyFieldName:    model.GetPipelineTablePrimaryKeyColumn(),
			SortByFieldName: "Name",
			IsDesc:          true,
		})
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pipelinesExpected := []model.Pipeline{expectedPipeline1}

	pipelines, nextPageToken, err := pipelineStore.ListPipelines(&common.PaginationContext{
		PageSize:        2,
		KeyFieldName:    model.GetPipelineTablePrimaryKeyColumn(),
		SortByFieldName: model.GetPipelineTablePrimaryKeyColumn(),
		IsDesc:          false,
	})
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelinesError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	_, _, err := pipelineStore.ListPipelines(&common.PaginationContext{
		PageSize:     2,
		KeyFieldName: model.GetPipelineTablePrimaryKeyColumn(),
		IsDesc:       true,
	})
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineExpected := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady,
	}

	pipeline, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline.")
}

func TestGetPipeline_NotFound_Creating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(&model.Pipeline{Name: "pipeline3", Status: model.PipelineCreating})

	_, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	_, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	_, err := pipelineStore.GetPipeline("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestCreatePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineExpected := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}

	pipeline := createPipeline("pipeline1")
	pipeline, err := pipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline.")
}

func TestCreatePipeline_DuplicateKey(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	pipeline := createPipeline("pipeline1")
	_, err := pipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	_, err = pipelineStore.CreatePipeline(pipeline)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name pipeline1 already exist")
}

func TestCreatePipeline_InternalServerError(t *testing.T) {
	pipeline := &model.Pipeline{Name: "Pipeline123"}
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()

	_, err := pipelineStore.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestDeletePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	err := pipelineStore.DeletePipeline(fakeUUID)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(fakeUUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipelineError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	err := pipelineStore.DeletePipeline(fakeUUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipeline, err := pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	assert.Nil(t, err)
	pipelineExpected := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineDeleting,
	}
	err = pipelineStore.UpdatePipelineStatus(pipeline.UUID, model.PipelineDeleting)
	assert.Nil(t, err)
	pipeline, err = pipelineStore.GetPipelineWithStatus(fakeUUID, model.PipelineDeleting)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)
}

func TestUpdatePipelineStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	err := pipelineStore.UpdatePipelineStatus(fakeUUID, model.PipelineDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
