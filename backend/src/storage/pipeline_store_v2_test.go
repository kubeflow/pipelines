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

	"github.com/googleprivate/ml/backend/src/model"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func initializePipelineDB() *gorm.DB {
	db := NewFakeDbOrFatal()
	pipeline1 := &model.PipelineDetailV2{
		PipelineV2: model.PipelineV2{
			UUID:           "1",
			Name:           "pp1",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 0,
			UpdatedAtInSec: 0,
		},
		ScheduledWorkflow: "scheduledworkflow1",
	}
	pipeline2 := &model.PipelineDetailV2{
		PipelineV2: model.PipelineV2{
			UUID:           "2",
			Name:           "pp2",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
		},
		ScheduledWorkflow: "scheduledworkflow2",
	}
	db.Create(pipeline1)
	db.Create(pipeline2)
	return db
}
func TestListPipelinesV2_Pagination(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())

	pipelinesExpected := []model.PipelineV2{
		{
			UUID:           "1",
			Name:           "pp1",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 0,
			UpdatedAtInSec: 0,
		}}
	pipelines, nextPageToken, err := pipelineStore.ListPipelines("" /*pageToken*/, 1 /*pageSize*/, "Name" /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pipelinesExpected, pipelines)
	pipelinesExpected2 := []model.PipelineV2{
		{
			UUID:           "2",
			Name:           "pp2",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
		}}
	pipelines, newToken, err := pipelineStore.ListPipelines(nextPageToken, 2 /*pageSize*/, "Name" /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Equal(t, "", newToken)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelinesV2_Pagination_LessThanPageSize(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())

	pipelinesExpected := []model.PipelineV2{
		{
			UUID:           "1",
			Name:           "pp1",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 0,
			UpdatedAtInSec: 0,
		},
		{
			UUID:           "2",
			Name:           "pp2",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
		}}
	pipelines, nextPageToken, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, model.GetPipelineV2TablePrimaryKeyColumn() /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelinesV2Error(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	db.Close()
	_, _, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, "Name" /*sortByFieldName*/)

	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to list pipeline to return error")
}

func TestGetPipelineV2(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())

	pipelineExpected := model.PipelineV2{
		UUID:           "1",
		Name:           "pp1",
		Namespace:      "n1",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 0,
		UpdatedAtInSec: 0,
	}

	pipeline, err := pipelineStore.GetPipeline("1")
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline")
}

func TestGetPipelineV2_NotFoundError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	_, err := pipelineStore.GetPipeline("notexist")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found error")
}

func TestGetPipelineV2_InternalError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	db.Close()
	_, err := pipelineStore.GetPipeline("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestDeletePipelineV2(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())

	err := pipelineStore.DeletePipeline("1")
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipelineV2_InternalError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	db.Close()

	err := pipelineStore.DeletePipeline("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete pipeline to return internal error")
}

func TestCreatePipelineV2(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	pipeline := &model.PipelineV2{
		UUID:      "1",
		Name:      "pp1",
		Namespace: "n1",
		PackageId: 1,
		Enabled:   true,
	}

	pipeline, err := pipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	pipelineExpected := &model.PipelineV2{
		UUID:           "1",
		Name:           "pp1",
		Namespace:      "n1",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	assert.Equal(t, pipelineExpected, pipeline, "Got unexpected pipelines")
}

func TestCreatePipelineV2Error(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	db.Close()
	pipeline := &model.PipelineV2{
		UUID:      "1",
		Name:      "pp1",
		Namespace: "n1",
		PackageId: 1,
		Enabled:   true,
	}

	pipeline, err := pipelineStore.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestEnablePipelineV2(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	err := pipelineStore.EnablePipeline("1", false)
	assert.Nil(t, err)

	pipelineExpected := model.PipelineV2{
		UUID:           "1",
		Name:           "pp1",
		Namespace:      "n1",
		PackageId:      1,
		Enabled:        false,
		CreatedAtInSec: 0,
		UpdatedAtInSec: 1,
	}

	pipeline, err := pipelineStore.GetPipeline("1")
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline")
}

func TestEnablePipelineV2_SkipUpdate(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	err := pipelineStore.EnablePipeline("1", true)
	assert.Nil(t, err)

	pipelineExpected := model.PipelineV2{
		UUID:           "1",
		Name:           "pp1",
		Namespace:      "n1",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 0,
		UpdatedAtInSec: 0,
	}

	pipeline, err := pipelineStore.GetPipeline("1")
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline")
}

func TestEnablePipelineV2_DatabaseError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStoreV2(db, util.NewFakeTimeForEpoch())
	db.Close()

	// Enabling the pipeline.
	err := pipelineStore.EnablePipeline("1", true)
	println(err.Error())
	assert.Contains(t, err.Error(), "Error when enabling pipeline 1 to true: sql: database is closed")
}
