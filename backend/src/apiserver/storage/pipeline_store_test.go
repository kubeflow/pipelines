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

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	fakeUUID      = "123e4567-e89b-12d3-a456-426655440000"
	fakeUUIDTwo   = "123e4567-e89b-12d3-a456-426655440001"
	fakeUUIDThree = "123e4567-e89b-12d3-a456-426655440002"
	fakeUUIDFour  = "123e4567-e89b-12d3-a456-426655440003"
	fakeUUIDFive  = "123e4567-e89b-12d3-a456-426655440004"
)

func createPipeline(name string) *model.Pipeline {
	return &model.Pipeline{
		Name:       name,
		Parameters: `[{"Name": "param1"}]`,
		Status:     model.PipelineReady,
		DefaultVersion: &model.PipelineVersion{
			Name:       name,
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineVersionReady,
		}}
}

func TestListPipelines_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipeline(&model.Pipeline{
		Name:   "pipeline3",
		Status: model.PipelineCreating,
		DefaultVersion: &model.PipelineVersion{
			Name:   "pipeline3",
			Status: model.PipelineVersionCreating}})

	expectedPipeline1 := &model.Pipeline{
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline2 := &model.Pipeline{
		UUID:             fakeUUIDTwo,
		CreatedAtInSec:   2,
		Name:             "pipeline2",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUIDTwo,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUIDTwo,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline2}

	opts, err := list.NewOptions(&model.Pipeline{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelines_WithFilter(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline_foo"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline_bar"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)

	expectedPipeline1 := &model.Pipeline{
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline_foo",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline_foo",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1}

	filterProto := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_f"},
			},
		},
	}
	opts, err := list.NewOptions(&model.Pipeline{}, 10, "id", filterProto)
	assert.Nil(t, err)

	pipelines, totalSize, nextPageToken, err := pipelineStore.ListPipelines(opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)
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
	expectedPipeline1 := &model.Pipeline{
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline4 := &model.Pipeline{
		UUID:             fakeUUIDFour,
		CreatedAtInSec:   4,
		Name:             "pipeline2",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUIDFour,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUIDFour,
			CreatedAtInSec: 4,
			Name:           "pipeline2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUIDFour,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline4}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline2 := &model.Pipeline{
		UUID:             fakeUUIDTwo,
		CreatedAtInSec:   2,
		Name:             "pipeline3",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUIDTwo,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUIDTwo,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline3 := &model.Pipeline{
		UUID:             fakeUUIDThree,
		CreatedAtInSec:   3,
		Name:             "pipeline4",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUIDThree,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUIDThree,
			CreatedAtInSec: 3,
			Name:           "pipeline4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUIDThree,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline2, expectedPipeline3}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	pipelines, total_size, nextPageToken, err = pipelineStore.ListPipelines(opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
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

	expectedPipeline2 := &model.Pipeline{
		UUID:             fakeUUIDTwo,
		CreatedAtInSec:   2,
		Name:             "pipeline3",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUIDTwo,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUIDTwo,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline3 := &model.Pipeline{
		UUID:             fakeUUIDThree,
		CreatedAtInSec:   3,
		Name:             "pipeline4",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUIDThree,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUIDThree,
			CreatedAtInSec: 3,
			Name:           "pipeline4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUIDThree,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline3, expectedPipeline2}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline1 := &model.Pipeline{
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline4 := &model.Pipeline{
		UUID:             fakeUUIDFour,
		CreatedAtInSec:   4,
		Name:             "pipeline2",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUIDFour,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUIDFour,
			CreatedAtInSec: 4,
			Name:           "pipeline2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUIDFour,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline4, expectedPipeline1}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err = pipelineStore.ListPipelines(opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	expectedPipeline1 := &model.Pipeline{
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelinesError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = pipelineStore.ListPipelines(opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineExpected := model.Pipeline{
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		}}

	pipeline, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline.")
}

func TestGetPipeline_NotFound_Creating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline3",
			Status: model.PipelineCreating,
			DefaultVersion: &model.PipelineVersion{
				Name:   "pipeline3",
				Status: model.PipelineVersionCreating,
			}})

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
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			Status:         model.PipelineVersionReady,
			PipelineId:     fakeUUID,
		}}

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
	pipeline := &model.Pipeline{
		Name:           "Pipeline123",
		DefaultVersion: &model.PipelineVersion{}}
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
		UUID:             fakeUUID,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineDeleting,
		DefaultVersionId: fakeUUID,
		DefaultVersion: &model.PipelineVersion{
			UUID:           fakeUUID,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			Status:         model.PipelineVersionDeleting,
			PipelineId:     fakeUUID,
		},
	}
	err = pipelineStore.UpdatePipelineStatus(pipeline.UUID, model.PipelineDeleting)
	assert.Nil(t, err)
	err = pipelineStore.UpdatePipelineVersionStatus(pipeline.UUID, model.PipelineVersionDeleting)
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

func TestCreatePipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline first.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		PipelineId:    fakeUUID,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(
		pipelineVersion, true)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineVersionCreating,
		PipelineId:     fakeUUID,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline.")

	// Check whether pipeline has updated default version id.
	pipeline, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipeline.DefaultVersionId, fakeUUIDTwo, "Got unexpected default version id.")
}

func TestCreatePipelineVersionNotUpdateDefaultVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline first.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		PipelineId:    fakeUUID,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(
		pipelineVersion, false)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineVersionCreating,
		PipelineId:     fakeUUID,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline.")

	// Check whether pipeline has updated default version id.
	pipeline, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Nil(t, err)
	assert.NotEqual(t, pipeline.DefaultVersionId, fakeUUIDTwo, "Got unexpected default version id.")
	assert.Equal(t, pipeline.DefaultVersionId, fakeUUID, "Got unexpected default version id.")

}

func TestCreatePipelineVersion_DuplicateKey(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionCreating,
		}, true)

	// Create another new version with same name.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param2"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionCreating,
		}, true)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name pipeline_version_1 already exist")
}

func TestCreatePipelineVersion_InternalServerError_DBClosed(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	db.Close()
	// Try to create a new version but db is closed.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
		}, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline version to return error")
}

func TestDeletePipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create a second version, which will become the default version.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Delete version with id being fakeUUIDThree.
	err := pipelineStore.DeletePipelineVersion(fakeUUIDThree)
	assert.Nil(t, err)

	// Check version removed.
	_, err = pipelineStore.GetPipelineVersion(fakeUUIDThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Check new default version is version with id being fakeUUIDTwo.
	pipeline, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pipeline.DefaultVersionId, fakeUUIDTwo)
}

func TestDeletePipelineVersionError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	db.Close()
	// On closed db, create pipeline version ends in internal error.
	err := pipelineStore.DeletePipelineVersion(fakeUUIDTwo)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Get pipeline version.
	pipelineVersion, err := pipelineStore.GetPipelineVersion(fakeUUIDTwo)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version.")
}

func TestGetPipelineVersion_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	db.Close()
	// Internal error because of closed DB.
	_, err := pipelineStore.GetPipelineVersion("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestGetPipelineVersion_NotFound_VersionStatusCreating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionCreating,
		}, true)

	_, err := pipelineStore.GetPipelineVersion(fakeUUIDTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipelineVersion_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	_, err := pipelineStore.GetPipelineVersion(fakeUUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestListPipelineVersion_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a first version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create a second version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create a third version with status creating.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionCreating,
		}, true)

	pipelineVersionsExpected := []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady},
		&model.PipelineVersion{
			UUID:           fakeUUIDThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady}}

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(fakeUUID, opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, pipelineVersionsExpected, pipelineVersions)
}

func TestListPipelineVersions_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create "version_1" with fakeUUIDTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_3" with fakeUUIDThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_2" with fakeUUIDFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_4" with fakeUUIDFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// List results in 2 pages: first page containing version_1 and version_2;
	// and second page containing verion_3 and version_4.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           fakeUUIDFour,
			CreatedAtInSec: 4,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err =
		pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)

	// Second page.
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           fakeUUIDThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           fakeUUIDFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create "version_1" with fakeUUIDTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_3" with fakeUUIDThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_2" with fakeUUIDFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_4" with fakeUUIDFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// List result in 2 pages: first page "version_4" and "version_3"; second
	// page "version_2" and "version_1".
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           fakeUUIDFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           fakeUUIDThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err =
		pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// Second Page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           fakeUUIDFour,
			CreatedAtInSec: 4,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           fakeUUIDTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     fakeUUID,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_WithFilter(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create "version_1" with fakeUUIDTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_2" with fakeUUIDThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Filter for name being equal to pipeline_version_1
	equalFilterProto := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key:   "name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_version_1"},
			},
		},
	}

	// Filter for name prefix being pipeline_version
	prefixFilterProto := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_version"},
			},
		},
	}

	// Only return 1 pipeline version with equal filter.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", equalFilterProto)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err := pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)

	// Return 2 pipeline versions without filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, totalSize)

	// Return 2 pipeline versions with prefix filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", prefixFilterProto)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, totalSize)
}

func TestListPipelineVersionsError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	db.Close()
	// Internal error because of closed DB.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = pipelineStore.ListPipelineVersions(fakeUUID, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineVersionStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineVersion, _ := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: fakeUUID,
			Status:     model.PipelineVersionReady,
		}, true)

	// Change version to deleting status
	err := pipelineStore.UpdatePipelineVersionStatus(
		pipelineVersion.UUID, model.PipelineVersionDeleting)
	assert.Nil(t, err)

	// Check the new status by retrieving this pipeline version.
	retrievedPipelineVersion, err :=
		pipelineStore.GetPipelineVersionWithStatus(
			pipelineVersion.UUID, model.PipelineVersionDeleting)
	assert.Nil(t, err)
	assert.Equal(t, *retrievedPipelineVersion, model.PipelineVersion{
		UUID:           fakeUUIDTwo,
		Name:           "pipeline_version_1",
		CreatedAtInSec: 2,
		Parameters:     `[{"Name": "param1"}]`,
		PipelineId:     fakeUUID,
		Status:         model.PipelineVersionDeleting,
	})
}

func TestUpdatePipelineVersionStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))

	db.Close()
	// Internal error because of closed DB.
	err := pipelineStore.UpdatePipelineVersionStatus(
		fakeUUID, model.PipelineVersionDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
