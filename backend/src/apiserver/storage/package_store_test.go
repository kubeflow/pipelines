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
	pipelineStore.CreatePipeline(createPipeline("pkg1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pkg2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipeline(&model.Pipeline{Name: "pkg3", Status: model.PipelineCreating})
	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline2 := model.Pipeline{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pkgsExpected := []model.Pipeline{expectedPipeline1, expectedPipeline2}

	pkgs, nextPageToken, err := pipelineStore.ListPipelines("" /*pageToken*/, 10 /*pageSize*/, model.GetPipelineTablePrimaryKeyColumn() /*sortByFieldName*/, false)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)
}

func TestListPipelines_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pkg1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pkg2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipeline(createPipeline("pkg2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	pipelineStore.CreatePipeline(createPipeline("pkg1"))
	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline4 := model.Pipeline{
		UUID:           fakeUUIDFour,
		CreatedAtInSec: 4,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pkgsExpected := []model.Pipeline{expectedPipeline1, expectedPipeline4}
	pkgs, nextPageToken, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, "Name" /*sortByFieldName*/, false)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)

	expectedPipeline2 := model.Pipeline{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline3 := model.Pipeline{
		UUID:           fakeUUIDThree,
		CreatedAtInSec: 3,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pkgsExpected2 := []model.Pipeline{expectedPipeline2, expectedPipeline3}

	pkgs, nextPageToken, err = pipelineStore.ListPipelines(nextPageToken, 2 /*pageSize*/, "Name" /*sortByFieldName*/, false)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, pkgsExpected2, pkgs)
}

func TestListPipelines_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pkg1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pkg2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDThree, nil)
	pipelineStore.CreatePipeline(createPipeline("pkg2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeUUIDFour, nil)
	pipelineStore.CreatePipeline(createPipeline("pkg1"))

	expectedPipeline2 := model.Pipeline{
		UUID:           fakeUUIDTwo,
		CreatedAtInSec: 2,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline3 := model.Pipeline{
		UUID:           fakeUUIDThree,
		CreatedAtInSec: 3,
		Name:           "pkg2",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pkgsExpected := []model.Pipeline{expectedPipeline3, expectedPipeline2}
	pkgs, nextPageToken, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, "Name" /*sortByFieldName*/, true /*isDesc*/)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)

	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	expectedPipeline4 := model.Pipeline{
		UUID:           fakeUUIDFour,
		CreatedAtInSec: 4,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pkgsExpected2 := []model.Pipeline{expectedPipeline4, expectedPipeline1}
	pkgs, nextPageToken, err = pipelineStore.ListPipelines(nextPageToken, 2 /*pageSize*/, "Name" /*sortByFieldName*/, true /*isDesc*/)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, pkgsExpected2, pkgs)
}

func TestListPipelines_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pkg1"))
	expectedPipeline1 := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}
	pkgsExpected := []model.Pipeline{expectedPipeline1}

	pkgs, nextPageToken, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, model.GetPipelineTablePrimaryKeyColumn() /*sortByFieldName*/, false /*isDesc*/)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pkgsExpected, pkgs)
}

func TestListPipelinesError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	_, _, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, model.GetPipelineTablePrimaryKeyColumn() /*sortByFieldName*/, false /*isDesc*/)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pkg1"))
	pkgExpected := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady,
	}

	pkg, err := pipelineStore.GetPipeline(fakeUUID)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected pipeline.")
}

func TestGetPipeline_NotFound_Creating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(&model.Pipeline{Name: "pkg3", Status: model.PipelineCreating})

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
	pkgExpected := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineReady}

	pkg := createPipeline("pkg1")
	pkg, err := pipelineStore.CreatePipeline(pkg)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg, "Got unexpected pipeline.")
}

func TestCreatePipelineError(t *testing.T) {
	pkg := &model.Pipeline{Name: "Pipeline123"}
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()

	_, err := pipelineStore.CreatePipeline(pkg)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestDeletePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	pipelineStore.CreatePipeline(createPipeline("pkg1"))
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
	pkg, err := pipelineStore.CreatePipeline(createPipeline("pkg1"))
	assert.Nil(t, err)
	pkgExpected := model.Pipeline{
		UUID:           fakeUUID,
		CreatedAtInSec: 1,
		Name:           "pkg1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineDeleting,
	}
	err = pipelineStore.UpdatePipelineStatus(pkg.UUID, model.PipelineDeleting)
	assert.Nil(t, err)
	pkg, err = pipelineStore.GetPipelineWithStatus(fakeUUID, model.PipelineDeleting)
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, *pkg)
}

func TestUpdatePipelineStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeUUID, nil))
	db.Close()
	err := pipelineStore.UpdatePipelineStatus(fakeUUID, model.PipelineDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
