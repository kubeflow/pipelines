// Copyright 2018 The Kubeflow Authors
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

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	defaultFakePipelineId      = "123e4567-e89b-12d3-a456-426655440000"
	defaultFakePipelineIdTwo   = "123e4567-e89b-12d3-a456-426655440001"
	defaultFakePipelineIdThree = "123e4567-e89b-12d3-a456-426655440002"
	defaultFakePipelineIdFour  = "123e4567-e89b-12d3-a456-426655440003"
	defaultFakePipelineIdFive  = "123e4567-e89b-12d3-a456-426655440004"
	defaultFakePipelineIdSix   = "123e4567-e89b-12d3-a456-426655440005"
	defaultFakePipelineIdSeven = "123e4567-e89b-12d3-a456-426655440006"
)

func createPipelineV1(name string) *model.Pipeline {
	return &model.Pipeline{
		Name:   name,
		Status: model.PipelineReady,
	}
}

func createPipeline(name string, description string, namespace string, url string) *model.Pipeline {
	return &model.Pipeline{
		Name:        name,
		Description: description,
		Status:      model.PipelineReady,
		Namespace:   namespace,
	}
}

func createPipelineVersion(pipelineId string, name string, description string, url string, pipelineSpec string, pipelineSpecURI string) *model.PipelineVersion {
	return &model.PipelineVersion{
		Name:            name,
		Parameters:      `[{"Name": "param1"}]`,
		PipelineId:      pipelineId,
		CodeSourceUrl:   url,
		Description:     description,
		Status:          model.PipelineVersionReady,
		PipelineSpec:    pipelineSpec,
		PipelineSpecURI: pipelineSpecURI,
	}
}

func TestListPipelinesAndVersions_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create pipelines
	p1 := createPipelineV1("pipeline1")
	p2 := createPipelineV1("pipeline2")
	p3 := createPipeline("pipeline3", "pipeline three", "user2", "url://foo/bar/p3")

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil)
	_p1, err := pipelineStore.CreatePipeline(p1)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	_p2, err := pipelineStore.CreatePipeline(p2)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	_p3, err := pipelineStore.CreatePipeline(p3)
	assert.Nil(t, err)

	expectedPipeline1 := &model.Pipeline{
		UUID:           defaultFakePipelineId,
		CreatedAtInSec: _p1.CreatedAtInSec,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	expectedPipeline2 := &model.Pipeline{
		UUID:           defaultFakePipelineIdTwo,
		CreatedAtInSec: _p2.CreatedAtInSec,
		Name:           "pipeline2",
		Status:         model.PipelineReady,
	}
	expectedPipeline3 := &model.Pipeline{
		UUID:           defaultFakePipelineIdThree,
		CreatedAtInSec: _p3.CreatedAtInSec,
		Name:           "pipeline3",
		Status:         model.PipelineReady,
		Description:    "pipeline three",
		Namespace:      "user2",
	}

	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline2, expectedPipeline3}
	opts, err := list.NewOptions(&model.Pipeline{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 3, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	// Create pipeline versions
	pv1 := createPipelineVersion(_p1.UUID, "pipeline1 v1", "pipeline one v1", "url://foo/bar/p1/v1", "yaml:file", "uri://p1.yaml")
	pv2 := createPipelineVersion(_p2.UUID, "pipeline2 v1", "pipeline two v1", "url://foo/bar/p2/v1", "yaml:file", "uri://p2.yaml")
	pv3 := createPipelineVersion(_p3.UUID, "pipeline3 v1", "pipeline three v1", "url://foo/bar/p3/v1", "yaml:file", "uri://p3.yaml")
	pv4 := createPipelineVersion(_p3.GetFieldValue("UUID").(string), "pipeline3 v2", "pipeline three v2", "url://foo/bar/p3/v2", "yaml:file", "uri://p4.yaml")

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	_pv1, err := pipelineStore.CreatePipelineVersion(pv1)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFive, nil)
	_pv2, err := pipelineStore.CreatePipelineVersion(pv2)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdSix, nil)
	_pv3, err := pipelineStore.CreatePipelineVersion(pv3)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdSeven, nil)
	_pv4, err := pipelineStore.CreatePipelineVersion(pv4)
	assert.Nil(t, err)

	expectedPipelineVersion1 := &model.PipelineVersion{
		UUID:            defaultFakePipelineIdFour,
		CreatedAtInSec:  _pv1.CreatedAtInSec,
		PipelineId:      _pv1.PipelineId,
		Name:            "pipeline1 v1",
		Parameters:      `[{"Name": "param1"}]`,
		Description:     "pipeline one v1",
		CodeSourceUrl:   "url://foo/bar/p1/v1",
		Status:          model.PipelineVersionReady,
		PipelineSpec:    "yaml:file",
		PipelineSpecURI: "uri://p1.yaml",
	}
	expectedPipelineVersion2 := &model.PipelineVersion{
		UUID:            defaultFakePipelineIdFive,
		CreatedAtInSec:  _pv2.CreatedAtInSec,
		PipelineId:      _pv2.PipelineId,
		Name:            "pipeline2 v1",
		Parameters:      `[{"Name": "param1"}]`,
		Description:     "pipeline two v1",
		CodeSourceUrl:   "url://foo/bar/p2/v1",
		Status:          model.PipelineVersionReady,
		PipelineSpec:    "yaml:file",
		PipelineSpecURI: "uri://p2.yaml",
	}
	expectedPipelineVersion3 := &model.PipelineVersion{
		UUID:            defaultFakePipelineIdSix,
		CreatedAtInSec:  _pv3.CreatedAtInSec,
		PipelineId:      _pv3.PipelineId,
		Name:            "pipeline3 v1",
		Parameters:      `[{"Name": "param1"}]`,
		Description:     "pipeline three v1",
		CodeSourceUrl:   "url://foo/bar/p3/v1",
		Status:          model.PipelineVersionReady,
		PipelineSpec:    "yaml:file",
		PipelineSpecURI: "uri://p3.yaml",
	}
	expectedPipelineVersion4 := &model.PipelineVersion{
		UUID:            defaultFakePipelineIdSeven,
		CreatedAtInSec:  _pv4.CreatedAtInSec,
		PipelineId:      _pv4.PipelineId,
		Name:            "pipeline3 v2",
		Parameters:      `[{"Name": "param1"}]`,
		Description:     "pipeline three v2",
		CodeSourceUrl:   "url://foo/bar/p3/v2",
		Status:          model.PipelineVersionReady,
		PipelineSpec:    "yaml:file",
		PipelineSpecURI: "uri://p4.yaml",
	}

	pipelinesVersionsExpected1 := []*model.PipelineVersion{expectedPipelineVersion1}
	pipelinesVersionsExpected2 := []*model.PipelineVersion{expectedPipelineVersion2}
	pipelinesVersionsExpected3 := []*model.PipelineVersion{expectedPipelineVersion3, expectedPipelineVersion4}
	opts2, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)

	pipelineVersions, total_size, nextPageToken, err := pipelineStore.ListPipelineVersions(_p1.UUID, opts2)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelinesVersionsExpected1, pipelineVersions)

	pipelineVersions, total_size, nextPageToken, err = pipelineStore.ListPipelineVersions(_p2.UUID, opts2)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelinesVersionsExpected2, pipelineVersions)

	pipelineVersions, total_size, nextPageToken, err = pipelineStore.ListPipelineVersions(_p3.UUID, opts2)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, pipelinesVersionsExpected3, pipelineVersions)
}

func TestListPipelines_WithFilter(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline_foo"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline_bar"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)

	expectedPipeline1 := &model.Pipeline{
		UUID:           defaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline_foo",
		Status:         model.PipelineReady,
	}
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

	pipelines, totalSize, nextPageToken, err := pipelineStore.ListPipelinesV1(&common.FilterContext{}, opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelines_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline2"))
	expectedPipeline1 := &model.Pipeline{
		UUID:           defaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	expectedPipeline4 := &model.Pipeline{
		UUID:           defaultFakePipelineIdFour,
		CreatedAtInSec: 4,
		Name:           "pipeline2",
		Status:         model.PipelineReady,
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline4}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelinesV1(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline2 := &model.Pipeline{
		UUID:           defaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline3",
		Status:         model.PipelineReady,
	}
	expectedPipeline3 := &model.Pipeline{
		UUID:           defaultFakePipelineIdThree,
		CreatedAtInSec: 3,
		Name:           "pipeline4",
		Status:         model.PipelineReady,
	}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline2, expectedPipeline3}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	pipelines, total_size, nextPageToken, err = pipelineStore.ListPipelinesV1(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline2"))

	expectedPipeline2 := &model.Pipeline{
		UUID:           defaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline3",
		Status:         model.PipelineReady,
	}
	expectedPipeline3 := &model.Pipeline{
		UUID:           defaultFakePipelineIdThree,
		CreatedAtInSec: 3,
		Name:           "pipeline4",
		Status:         model.PipelineReady,
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline3, expectedPipeline2}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelinesV1(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline1 := &model.Pipeline{
		UUID:           defaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	expectedPipeline4 := &model.Pipeline{
		UUID:           defaultFakePipelineIdFour,
		CreatedAtInSec: 4,
		Name:           "pipeline2",
		Status:         model.PipelineReady,
	}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline4, expectedPipeline1}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err = pipelineStore.ListPipelinesV1(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	expectedPipeline1 := &model.Pipeline{
		UUID:           defaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelinesV1(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelinesError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = pipelineStore.ListPipelinesV1(&common.FilterContext{}, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	pipelineExpected := model.Pipeline{
		UUID:           defaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}

	pipeline, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline.")
}

func TestGetPipeline_NotFound_Creating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline3",
			Status: model.PipelineCreating,
		},
	)

	_, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	_, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	_, err := pipelineStore.GetPipeline("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestGetPipelineByNameAndNamespace(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	p := createPipelineV1("pipeline1")
	p.Namespace = "ns1"
	resPipeline, err := pipelineStore.CreatePipeline(p)
	pipeline, err := pipelineStore.GetPipelineByNameAndNamespace("pipeline1", "ns1")
	assert.Nil(t, err)
	assert.Equal(t, resPipeline, pipeline)
}

func TestGetPipelineByNameAndNamespace_NotFound(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	p := createPipelineV1("pipeline1")
	p.Namespace = "ns1"
	_, err := pipelineStore.CreatePipeline(p)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineByNameAndNamespace(p.Name, "wrong_namespace")
	assert.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Failed to get pipeline by name and namespace")
}

func TestCreatePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineExpected := createPipeline("pipeline1", "pipeline one", "user1", "uri://pipeline1")
	pipelineExpected.UUID = defaultFakePipelineId
	pipelineExpected.CreatedAtInSec = 1
	pipelineActual, err := pipelineStore.CreatePipeline(pipelineExpected)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, pipelineActual, "Got unexpected pipeline.")
}

func TestCreatePipeline_DuplicateKey(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	pipeline := createPipelineV1("pipeline1")
	_, err := pipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	_, err = pipelineStore.CreatePipeline(pipeline)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name pipeline1 already exist")
}

func TestCreatePipeline_InternalServerError(t *testing.T) {
	pipeline := &model.Pipeline{
		Name: "Pipeline123",
	}
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()

	_, err := pipelineStore.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestDeletePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	_, err := pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	p2, err := pipelineStore.CreatePipeline(createPipelineV1("pipeline2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	_, err = pipelineStore.CreatePipeline(createPipelineV1("pipeline3"))

	p1v1 := createPipelineVersion(defaultFakeExpId, "pipeline1/v1", "pipeline one v1", "url://pipeline1/v1", "yaml:spec", "uri://p1/v1.yaml")
	p1v2 := createPipelineVersion(defaultFakeExpId, "pipeline1/v2", "pipeline one v2", "url://pipeline1/v2", "yaml:spec", "uri://p1/v2.yaml")
	p2v1 := createPipelineVersion(defaultFakePipelineIdTwo, "pipeline2/v1", "pipeline two v1", "url://pipeline2/v1", "yaml:spec", "uri://p2/v1.yaml")
	p2v1.SetFieldValue("Pipeline", p2)

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil)
	_, err = pipelineStore.CreatePipelineVersion(p1v1)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	_, err = pipelineStore.CreatePipelineVersion(p1v2)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	_, err = pipelineStore.CreatePipelineVersion(p2v1)

	// Delete pipeline # 2
	err = pipelineStore.DeletePipeline(defaultFakePipelineIdTwo)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(defaultFakePipelineIdThree)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(defaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	// Check pipeline versions
	_, err = pipelineStore.GetPipelineVersion(defaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	assert.Nil(t, err)
	// p3, err := pipelineStore.GetPipelineVersion(defaultFakePipelineIdThree)
	// assert.Equal(t, "", p3.PipelineId)

	// Delete pipeline # 3
	err = pipelineStore.DeletePipeline(defaultFakePipelineIdThree)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineVersion(defaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(defaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	_, err = pipelineStore.GetPipeline(defaultFakePipelineIdThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	// Check pipeline versions
	_, err = pipelineStore.GetPipelineVersion(defaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	assert.Nil(t, err)
	// p3, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdThree)
	// assert.Equal(t, "", p3.PipelineId)

	// Delete pipeline # 1
	err = pipelineStore.DeletePipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	_, err = pipelineStore.GetPipeline(defaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	_, err = pipelineStore.GetPipeline(defaultFakePipelineIdThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	// Check pipeline versions
	// p3, err = pipelineStore.GetPipelineVersion(defaultFakePipelineId)
	// assert.Equal(t, "", p3.PipelineId)
	// p3, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	// assert.Equal(t, "", p3.PipelineId)
	// p3, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdThree)
	// assert.Equal(t, "", p3.PipelineId)
}

func TestDeletePipelineError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	err := pipelineStore.DeletePipeline(defaultFakePipelineId)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipeline, err := pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	assert.Nil(t, err)
	pipelineExpected := model.Pipeline{
		UUID:           defaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineDeleting,
	}
	err = pipelineStore.UpdatePipelineStatus(pipeline.UUID, model.PipelineDeleting)
	assert.Nil(t, err)
	pipeline, err = pipelineStore.GetPipelineWithStatus(defaultFakePipelineId, model.PipelineDeleting)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)
}

func TestUpdatePipelineStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	err := pipelineStore.UpdatePipelineStatus(defaultFakePipelineId, model.PipelineDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestCreatePipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil),
	)

	// Create a pipeline first.
	_, err := pipelineStore.CreatePipeline(
		createPipeline("p1", "pipeline one", "user1", "url://p1.yaml"),
	)

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		Description:   "pipeline_version_description",
		PipelineId:    defaultFakePipelineId,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(
		pipelineVersion,
	)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           defaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Description:    "pipeline_version_description",
		Status:         model.PipelineVersionCreating,
		PipelineId:     defaultFakePipelineId,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline.")
}

func TestCreatePipelineVersionNotUpdateDefaultVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline first.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		PipelineId:    defaultFakePipelineId,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(pipelineVersion)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           defaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineVersionCreating,
		PipelineId:     defaultFakePipelineId,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline.")
}

func TestCreatePipelineVersion_DuplicateKey(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)

	// Create another new version with same name.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param2"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name pipeline_version_1 already exist")
}

func TestCreatePipelineVersion_InternalServerError_DBClosed(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	db.Close()
	// Try to create a new version but db is closed.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
		},
	)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline version to return error")
}

func TestDeletePipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create a second version, which will become the default version.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Delete version with id being defaultFakePipelineIdThree.
	err := pipelineStore.DeletePipelineVersion(defaultFakePipelineIdThree)
	assert.Nil(t, err)

	// Check version removed.
	_, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipelineVersionError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	db.Close()
	// On closed db, create pipeline version ends in internal error.
	err := pipelineStore.DeletePipelineVersion(defaultFakePipelineIdTwo)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Get pipeline version.
	pipelineVersion, err := pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version.")
}

func TestGetLatestPipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		},
	)

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Get pipeline version 1.
	pipelineVersion, err := pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version.",
	)

	// Get pipeline version 2.
	pipelineVersion, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdThree)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			Name:           "pipeline_version_2",
			CreatedAtInSec: 3,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version.")

	// Get the latest pipeline version.
	pipelineVersion, err = pipelineStore.GetLatestPipelineVersion(defaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			Name:           "pipeline_version_2",
			CreatedAtInSec: 3,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
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
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

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
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)

	_, err := pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipelineVersion_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	_, err := pipelineStore.GetPipelineVersion(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestListPipelineVersion_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a first version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create a second version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create a third version with status creating.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)

	pipelineVersionsExpected := []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady},
	}

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)

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
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create "version_1" with defaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_3" with defaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_2" with defaultFakePipelineIdFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_4" with defaultFakePipelineIdFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// List results in 2 pages: first page containing version_1 and version_2;
	// and second page containing verion_3 and version_4.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdFour,
			CreatedAtInSec: 4,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err =
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)

	// Second page.
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
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
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create "version_1" with defaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:        "pipeline_version_1",
			Parameters:  `[{"Name": "param1"}]`,
			PipelineId:  defaultFakePipelineId,
			Status:      model.PipelineVersionReady,
			Description: "version_1",
		},
	)

	// Create "version_3" with defaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_2" with defaultFakePipelineIdFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_4" with defaultFakePipelineIdFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// List result in 2 pages: first page "version_4" and "version_3"; second
	// page "version_2" and "version_1".
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err =
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// Second Page.
	assert.Equal(
		t, pipelineVersions, []*model.PipelineVersion{
			&model.PipelineVersion{
				UUID:           defaultFakePipelineIdFour,
				CreatedAtInSec: 4,
				Name:           "pipeline_version_2",
				Parameters:     `[{"Name": "param1"}]`,
				PipelineId:     defaultFakePipelineId,
				Status:         model.PipelineVersionReady,
			},
			&model.PipelineVersion{
				UUID:           defaultFakePipelineIdTwo,
				CreatedAtInSec: 2,
				Name:           "pipeline_version_1",
				Description:    "version_1",
				Parameters:     `[{"Name": "param1"}]`,
				PipelineId:     defaultFakePipelineId,
				Status:         model.PipelineVersionReady,
			},
		},
	)
}

func TestListPipelineVersions_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
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
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create "version_1" with defaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_2" with defaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

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
	_, totalSize, nextPageToken, err := pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)

	// Return 2 pipeline versions without filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, totalSize)

	// Return 2 pipeline versions with prefix filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", prefixFilterProto)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
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
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineVersionStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineVersion, _ := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

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
		UUID:           defaultFakePipelineIdTwo,
		Name:           "pipeline_version_1",
		CreatedAtInSec: 2,
		Parameters:     `[{"Name": "param1"}]`,
		PipelineId:     defaultFakePipelineId,
		Status:         model.PipelineVersionDeleting,
	})
}

func TestUpdatePipelineVersionStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	err := pipelineStore.UpdatePipelineVersionStatus(
		defaultFakePipelineId, model.PipelineVersionDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
