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
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func createPipelineV1(name string) *model.Pipeline {
	return &model.Pipeline{
		Name:   name,
		Status: model.PipelineReady,
	}
}

func createPipeline(name string, description string, namespace string) *model.Pipeline {
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
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create pipelines
	p1 := createPipelineV1("pipeline1")
	p2 := createPipelineV1("pipeline2")
	p3 := createPipeline("pipeline3", "pipeline three", "user2")

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil)
	_p1, err := pipelineStore.CreatePipeline(p1)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	_p2, err := pipelineStore.CreatePipeline(p2)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	_p3, err := pipelineStore.CreatePipeline(p3)
	assert.Nil(t, err)

	expectedPipeline1 := &model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: _p1.CreatedAtInSec,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	expectedPipeline2 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdTwo,
		CreatedAtInSec: _p2.CreatedAtInSec,
		Name:           "pipeline2",
		Status:         model.PipelineReady,
	}
	expectedPipeline3 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdThree,
		CreatedAtInSec: _p3.CreatedAtInSec,
		Name:           "pipeline3",
		Status:         model.PipelineReady,
		Description:    "pipeline three",
		Namespace:      "user2",
	}

	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline2, expectedPipeline3}
	opts, err := list.NewOptions(&model.Pipeline{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 3, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	// Create pipeline versions
	pv1 := createPipelineVersion(_p1.UUID, "pipeline1 v1", "pipeline one v1", "url://foo/bar/p1/v1", "yaml:file", "uri://p1.yaml")
	pv2 := createPipelineVersion(_p2.UUID, "pipeline2 v1", "pipeline two v1", "url://foo/bar/p2/v1", "yaml:file", "uri://p2.yaml")
	pv3 := createPipelineVersion(_p3.UUID, "pipeline3 v1", "pipeline three v1", "url://foo/bar/p3/v1", "yaml:file", "uri://p3.yaml")
	pv4 := createPipelineVersion(_p3.GetFieldValue("UUID").(string), "pipeline3 v2", "pipeline three v2", "url://foo/bar/p3/v2", "yaml:file", "uri://p4.yaml")

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFour, nil)
	_pv1, err := pipelineStore.CreatePipelineVersion(pv1)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFive, nil)
	_pv2, err := pipelineStore.CreatePipelineVersion(pv2)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdSix, nil)
	_pv3, err := pipelineStore.CreatePipelineVersion(pv3)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdSeven, nil)
	_pv4, err := pipelineStore.CreatePipelineVersion(pv4)
	assert.Nil(t, err)

	expectedPipelineVersion1 := &model.PipelineVersion{
		UUID:            DefaultFakePipelineIdFour,
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
		UUID:            DefaultFakePipelineIdFive,
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
		UUID:            DefaultFakePipelineIdSix,
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
		UUID:            DefaultFakePipelineIdSeven,
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
	assert.Nil(t, err)

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
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline_foo"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline_bar"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)

	expectedPipeline1 := &model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline_foo",
		Status:         model.PipelineReady,
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1}

	filterProto := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_f"},
			},
		},
	}
	newFilter, _ := filter.New(filterProto)
	opts, err := list.NewOptions(&model.Pipeline{}, 10, "id", newFilter)
	assert.Nil(t, err)

	pipelines, _, totalSize, nextPageToken, err := pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelines_Pagination(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline2"))
	expectedPipeline1 := &model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	expectedPipeline4 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdFour,
		CreatedAtInSec: 4,
		Name:           "pipeline2",
		Status:         model.PipelineReady,
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline4}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelines, _, total_size, nextPageToken, err := pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline2 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline3",
		Status:         model.PipelineReady,
	}
	expectedPipeline3 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdThree,
		CreatedAtInSec: 3,
		Name:           "pipeline4",
		Status:         model.PipelineReady,
	}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline2, expectedPipeline3}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	pipelines, _, total_size, nextPageToken, err = pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_Descend(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipeline(createPipelineV1("pipeline2"))

	expectedPipeline2 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline3",
		Status:         model.PipelineReady,
	}
	expectedPipeline3 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdThree,
		CreatedAtInSec: 3,
		Name:           "pipeline4",
		Status:         model.PipelineReady,
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline3, expectedPipeline2}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelines, _, total_size, nextPageToken, err := pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline1 := &model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	expectedPipeline4 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdFour,
		CreatedAtInSec: 4,
		Name:           "pipeline2",
		Status:         model.PipelineReady,
	}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline4, expectedPipeline1}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelines, _, total_size, nextPageToken, err = pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelinesV1_Pagination_NameAsc(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("bbb"))

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(createPipelineVersion(DefaultFakePipelineId, "pipeline1/v1", "", "", "", ""))

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(createPipelineVersion(DefaultFakePipelineId, "pipeline1/v2", "", "", "", ""))

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipelineV1("aaa"))

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(createPipelineVersion(DefaultFakePipelineIdTwo, "pipeline2/v1", "", "", "", ""))

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFive, nil)
	pipelineStore.CreatePipelineVersion(createPipelineVersion(DefaultFakePipelineIdTwo, "pipeline2/v2", "", "", "", ""))

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(createPipelineV1("ccc"))

	expectedPipeline1 := &model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "bbb",
		Status:         model.PipelineReady,
	}
	expectedPipeline2 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdTwo,
		CreatedAtInSec: 4,
		Name:           "aaa",
		Status:         model.PipelineReady,
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline2, expectedPipeline1}

	expectedPipelineVersion1 := &model.PipelineVersion{
		PipelineId:     DefaultFakePipelineId,
		UUID:           DefaultFakePipelineIdThree,
		CreatedAtInSec: 3,
		Name:           "pipeline1/v2",
		Status:         model.PipelineVersionReady,
		Parameters:     `[{"Name": "param1"}]`,
	}
	expectedPipelineVersion2 := &model.PipelineVersion{
		PipelineId:     DefaultFakePipelineIdTwo,
		UUID:           DefaultFakePipelineIdFive,
		CreatedAtInSec: 6,
		Name:           "pipeline2/v2",
		Status:         model.PipelineVersionReady,
		Parameters:     `[{"Name": "param1"}]`,
	}
	pipelineVersionsExpected := []*model.PipelineVersion{expectedPipelineVersion2, expectedPipelineVersion1}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name asc", nil)
	assert.Nil(t, err)
	pipelines, pipelineVersions, total_size, nextPageToken, err := pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 3, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)
	assert.Equal(t, pipelineVersionsExpected, pipelineVersions)

	expectedPipeline3 := &model.Pipeline{
		UUID:           DefaultFakePipelineIdThree,
		CreatedAtInSec: 7,
		Name:           "ccc",
		Status:         model.PipelineReady,
	}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline3}
	pipelineVersionsExpected2 := []*model.PipelineVersion{{}}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelines, pipelineVersions, total_size, nextPageToken, err = pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 3, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
	assert.Equal(t, pipelineVersionsExpected2, pipelineVersions)
}

func TestListPipelines_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	p := createPipelineV1("pipeline1")
	p1, err := pipelineStore.CreatePipeline(p)
	assert.Nil(t, err)
	pipelineStore.CreatePipelineVersion(
		createPipelineVersion(
			p1.UUID,
			"version1",
			"",
			"",
			"",
			"",
		),
	)
	expectedPipeline1 := &model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}
	expectedPipelineVersion1 := &model.PipelineVersion{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 2,
		Name:           "version1",
		Status:         model.PipelineVersionReady,
		PipelineId:     DefaultFakePipelineId,
		Parameters:     "[{\"Name\": \"param1\"}]",
	}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1}
	pipelineVersionsExpected := []*model.PipelineVersion{expectedPipelineVersion1}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	pipelines, pipelineVersions, total_size, nextPageToken, err := pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)
	assert.Equal(t, pipelineVersionsExpected, pipelineVersions)
}

func TestListPipelinesError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	db.Close()
	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, _, err = pipelineStore.ListPipelinesV1(&model.FilterContext{}, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipeline(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	pipelineExpected := model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineReady,
	}

	pipeline, err := pipelineStore.GetPipeline(DefaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline")
}

func TestGetPipeline_NotFound_Creating(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline3",
			Status: model.PipelineCreating,
		},
	)

	_, err := pipelineStore.GetPipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_NotFoundError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	_, err := pipelineStore.GetPipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_InternalError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	db.Close()
	_, err := pipelineStore.GetPipeline("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestGetPipelineByNameAndNamespace(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	p := createPipelineV1("pipeline1")
	p.Namespace = "ns1"
	resPipeline, err := pipelineStore.CreatePipeline(p)
	assert.Nil(t, err)
	pipeline, err := pipelineStore.GetPipelineByNameAndNamespace("pipeline1", "ns1")
	assert.Nil(t, err)
	assert.Equal(t, resPipeline, pipeline)
}

func TestGetPipelineByNameAndNamespace_NotFound(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	p := createPipelineV1("pipeline1")
	p.Namespace = "ns1"
	_, err := pipelineStore.CreatePipeline(p)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineByNameAndNamespace(p.Name, "wrong_namespace")
	assert.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Failed to get pipeline by name and namespace")
}

func TestGetPipelineByNameAndNamespaceV1(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	p := createPipelineV1("pipeline1")
	p.Namespace = "ns1"
	resPipeline, err := pipelineStore.CreatePipeline(p)
	assert.Nil(t, err)
	pv := createPipelineVersion(resPipeline.UUID, "pipeline1", "", "", "", "")
	resPipelineV, err := pipelineStore.CreatePipelineVersion(pv)
	assert.Nil(t, err)
	pipeline, pipelineVersion, err := pipelineStore.GetPipelineByNameAndNamespaceV1("pipeline1", "ns1")
	assert.Nil(t, err)
	assert.Equal(t, resPipeline, pipeline)
	assert.Equal(t, resPipelineV, pipelineVersion)
}

func TestGetPipelineByNameAndNamespaceV1_NotFound(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	p := createPipelineV1("pipeline1")
	p.Namespace = "ns1"
	_, err := pipelineStore.CreatePipeline(p)
	assert.Nil(t, err)
	_, _, err = pipelineStore.GetPipelineByNameAndNamespaceV1(p.Name, "wrong_namespace")
	assert.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Failed to get pipeline by name and namespace")
}

func TestPipelineStore_CreatePipelineAndPipelineVersion(t *testing.T) {
	tests := []struct {
		name                string
		p                   *model.Pipeline
		pv                  *model.PipelineVersion
		wantPipeline        *model.Pipeline
		wantPipelineVersion *model.PipelineVersion
		wantErr             bool
		errMsg              string
	}{
		{
			"Valid - pipeline v2",
			&model.Pipeline{
				Name:        "pipeline v2",
				Description: "pipeline two",
				Namespace:   "user1",
			},
			&model.PipelineVersion{
				Name:            "pipeline v2 version 1",
				Description:     "pipeline v2 version description",
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    v2SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_two.yaml",
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 3,
				Name:           "pipeline v2",
				Description:    "pipeline two",
				Namespace:      "user1",
				Status:         model.PipelineCreating,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  4,
				Name:            "pipeline v2 version 1",
				Description:     "pipeline v2 version description",
				PipelineId:      DefaultFakePipelineIdTwo,
				Status:          model.PipelineVersionCreating,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    v2SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_two.yaml",
			},
			false,
			"",
		},
		{
			"Valid - pipeline v1",
			&model.Pipeline{
				Name:        "pipeline v1",
				Description: "pipeline one",
				Parameters:  `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
			},
			&model.PipelineVersion{
				Name:            "pipeline v1 version 1",
				Parameters:      `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
				Description:     "pipeline v1 version description",
				CodeSourceUrl:   "gs://my-bucket/pipeline_v1.py",
				PipelineSpec:    v1SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_one.yaml",
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 3,
				Name:           "pipeline v1",
				Description:    "pipeline one",
				Parameters:     `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
				Status:         model.PipelineCreating,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  4,
				PipelineId:      DefaultFakePipelineIdTwo,
				Name:            "pipeline v1 version 1",
				Parameters:      `[{"name":"param1","value":"one"},{"name":"param2","value":"two"}]`,
				Description:     "pipeline v1 version description",
				Status:          model.PipelineVersionCreating,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v1.py",
				PipelineSpec:    v1SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_one.yaml",
			},
			false,
			"",
		},
		{
			"Invalid - duplicate pipeline v2",
			&model.Pipeline{
				Name:        "default pipeline",
				Description: "pipeline three",
				Namespace:   "kubeflow",
			},
			&model.PipelineVersion{
				Name:            "pipeline version three",
				Description:     "pipeline version three description",
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    v2SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_two.yaml",
			},
			nil,
			nil,
			true,
			"Failed to create a new pipeline. The name default pipeline already exists",
		},
		{
			"Valid - duplicate pipeline version v2",
			&model.Pipeline{
				Name:        "pipeline three",
				Description: "pipeline three",
				Namespace:   "kubeflow",
			},
			&model.PipelineVersion{
				Name:            "default version",
				Description:     "default version description",
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    v2SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_two.yaml",
			},
			&model.Pipeline{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 3,
				Name:           "pipeline three",
				Description:    "pipeline three",
				Namespace:      "kubeflow",
				Status:         model.PipelineCreating,
			},
			&model.PipelineVersion{
				UUID:            DefaultFakePipelineIdTwo,
				CreatedAtInSec:  4,
				PipelineId:      DefaultFakePipelineIdTwo,
				Name:            "default version",
				Description:     "default version description",
				Status:          model.PipelineVersionCreating,
				CodeSourceUrl:   "gs://my-bucket/pipeline_v2.py",
				PipelineSpec:    v2SpecHelloWorld,
				PipelineSpecURI: "pipeline_version_two.yaml",
			},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewFakeDBOrFatal()
			pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
			// Create a default pipeline
			pipelineStore.CreatePipeline(
				&model.Pipeline{
					Name:      "default pipeline",
					Namespace: "kubeflow",
					Status:    model.PipelineReady,
				})
			// Create a default version
			pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil)
			pipelineStore.CreatePipelineVersion(
				&model.PipelineVersion{
					Name:       "default version",
					Parameters: `[{"Name":"param3","value":"three"}]`,
					PipelineId: DefaultFakePipelineId,
					Status:     model.PipelineVersionReady,
				},
			)
			pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)

			gotPipeline, gotPipelineVersion, err := pipelineStore.CreatePipelineAndPipelineVersion(tt.p, tt.pv)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, gotPipeline)
				assert.Nil(t, gotPipelineVersion)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.wantPipeline, gotPipeline)
				assert.Equal(t, tt.wantPipelineVersion, gotPipelineVersion)
			}
			db.Close()
		})
	}
}

func TestCreatePipeline(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipelineExpected := createPipeline("pipeline1", "pipeline one", "user1")
	pipelineExpected.UUID = DefaultFakePipelineId
	pipelineExpected.CreatedAtInSec = 1
	pipelineActual, err := pipelineStore.CreatePipeline(pipelineExpected)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, pipelineActual, "Got unexpected pipeline")
}

func TestCreatePipeline_DuplicateKey(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

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
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	db.Close()

	_, err := pipelineStore.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestDeletePipeline(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	_, err := pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	p2, err := pipelineStore.CreatePipeline(createPipelineV1("pipeline2"))
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	_, err = pipelineStore.CreatePipeline(createPipelineV1("pipeline3"))
	assert.Nil(t, err)

	p1v1 := createPipelineVersion(defaultFakeExpId, "pipeline1/v1", "pipeline one v1", "url://pipeline1/v1", "yaml:spec", "uri://p1/v1.yaml")
	p1v2 := createPipelineVersion(defaultFakeExpId, "pipeline1/v2", "pipeline one v2", "url://pipeline1/v2", "yaml:spec", "uri://p1/v2.yaml")
	p2v1 := createPipelineVersion(DefaultFakePipelineIdTwo, "pipeline2/v1", "pipeline two v1", "url://pipeline2/v1", "yaml:spec", "uri://p2/v1.yaml")
	p2v1.PipelineId = p2.UUID

	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil)
	_, err = pipelineStore.CreatePipelineVersion(p1v1)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	_, err = pipelineStore.CreatePipelineVersion(p1v2)
	assert.Nil(t, err)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	_, err = pipelineStore.CreatePipelineVersion(p2v1)
	assert.Nil(t, err)

	// Delete pipeline # 2
	err = pipelineStore.DeletePipeline(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineIdThree)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	// Check pipeline versions
	_, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	// p3, err := pipelineStore.GetPipelineVersion(DefaultFakePipelineIdThree)
	// assert.Equal(t, "", p3.PipelineId)

	// Delete pipeline # 3
	err = pipelineStore.DeletePipeline(DefaultFakePipelineIdThree)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineIdThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	// Check pipeline versions
	_, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	// p3, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineIdThree)
	// assert.Equal(t, "", p3.PipelineId)

	// Delete pipeline # 1
	err = pipelineStore.DeletePipeline(DefaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	_, err = pipelineStore.GetPipeline(DefaultFakePipelineIdThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	// Check pipeline versions
	// p3, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineId)
	// assert.Equal(t, "", p3.PipelineId)
	// p3, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineIdTwo)
	// assert.Equal(t, "", p3.PipelineId)
	// p3, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineIdThree)
	// assert.Equal(t, "", p3.PipelineId)
}

func TestDeletePipelineError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	db.Close()
	err := pipelineStore.DeletePipeline(DefaultFakePipelineId)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineStatus(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	pipeline, err := pipelineStore.CreatePipeline(createPipelineV1("pipeline1"))
	assert.Nil(t, err)
	pipelineExpected := model.Pipeline{
		UUID:           DefaultFakePipelineId,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Status:         model.PipelineDeleting,
	}
	err = pipelineStore.UpdatePipelineStatus(pipeline.UUID, model.PipelineDeleting)
	assert.Nil(t, err)
	pipeline, err = pipelineStore.GetPipelineWithStatus(DefaultFakePipelineId, model.PipelineDeleting)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)
}

func TestUpdatePipelineStatusError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))
	db.Close()
	err := pipelineStore.UpdatePipelineStatus(DefaultFakePipelineId, model.PipelineDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestCreatePipelineVersion(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil),
	)

	// Create a pipeline first.
	_, err := pipelineStore.CreatePipeline(
		createPipeline("p1", "pipeline one", "user1"),
	)
	assert.Nil(t, err)

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		Description:   "pipeline_version_description",
		PipelineId:    DefaultFakePipelineId,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(
		pipelineVersion,
	)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           DefaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Description:    "pipeline_version_description",
		Status:         model.PipelineVersionCreating,
		PipelineId:     DefaultFakePipelineId,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline")
}

func TestUpdatePipelineDefaultVersion(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil),
	)

	// Create a pipeline first.
	p, err := pipelineStore.CreatePipeline(
		createPipeline("p1", "pipeline one", "user1"),
	)
	assert.Nil(t, err)
	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		Description:   "pipeline_version_description",
		PipelineId:    DefaultFakePipelineId,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(
		pipelineVersion,
	)
	assert.Nil(t, err)
	err = pipelineStore.UpdatePipelineDefaultVersion(p.UUID, pipelineVersionCreated.UUID)
	assert.Nil(t, err)
	err = pipelineStore.UpdatePipelineDefaultVersion(p.UUID, "something else")
	assert.Nil(t, err)
}

func TestCreatePipelineVersionNotUpdateDefaultVersion(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline first.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		PipelineId:    DefaultFakePipelineId,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(pipelineVersion)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           DefaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineVersionCreating,
		PipelineId:     DefaultFakePipelineId,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline")
}

func TestCreatePipelineVersion_DuplicateKey(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)

	// Create another new version with same name.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param2"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name pipeline_version_1 already exist")
}

func TestCreatePipelineVersion_InternalServerError_DBClosed(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	db.Close()
	// Try to create a new version but db is closed.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
		},
	)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline version to return error")
}

func TestDeletePipelineVersion(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create a second version, which will become the default version.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Delete version with id being DefaultFakePipelineIdThree.
	err := pipelineStore.DeletePipelineVersion(DefaultFakePipelineIdThree)
	assert.Nil(t, err)

	// Check version removed.
	_, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineIdThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipelineVersionError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	db.Close()
	// On closed db, create pipeline version ends in internal error.
	err := pipelineStore.DeletePipelineVersion(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineVersion(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Get pipeline version.
	pipelineVersion, err := pipelineStore.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           DefaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version")
}

func TestGetLatestPipelineVersion(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		},
	)

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Get pipeline version 1.
	pipelineVersion, err := pipelineStore.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           DefaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version",
	)

	// Get pipeline version 2.
	pipelineVersion, err = pipelineStore.GetPipelineVersion(DefaultFakePipelineIdThree)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           DefaultFakePipelineIdThree,
			Name:           "pipeline_version_2",
			CreatedAtInSec: 3,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version")

	// Get the latest pipeline version.
	pipelineVersion, err = pipelineStore.GetLatestPipelineVersion(DefaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           DefaultFakePipelineIdThree,
			Name:           "pipeline_version_2",
			CreatedAtInSec: 3,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version")
}

func TestGetPipelineVersion_InternalError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	_, err := pipelineStore.GetPipelineVersion("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestGetPipelineVersion_NotFound_VersionStatusCreating(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)

	_, err := pipelineStore.GetPipelineVersion(DefaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipelineVersion_NotFoundError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	_, err := pipelineStore.GetPipelineVersion(DefaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestListPipelineVersion_FilterOutNotReady(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a first version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create a second version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create a third version with status creating.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		},
	)

	pipelineVersionsExpected := []*model.PipelineVersion{
		{
			UUID:           DefaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		{
			UUID:           DefaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	}

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelineVersions, total_size, nextPageToken, err := pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, pipelineVersionsExpected, pipelineVersions)
}

func TestListPipelineVersions_Pagination(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create "version_1" with DefaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_3" with DefaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_2" with DefaultFakePipelineIdFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_4" with DefaultFakePipelineIdFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// List results in 2 pages: first page containing version_1 and version_2;
	// and second page containing verion_3 and version_4.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err := pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		{
			UUID:           DefaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		{
			UUID:           DefaultFakePipelineIdFour,
			CreatedAtInSec: 4,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err = pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)

	// Second page.
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		{
			UUID:           DefaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		{
			UUID:           DefaultFakePipelineIdFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_Pagination_Descend(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create "version_1" with DefaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:        "pipeline_version_1",
			Parameters:  `[{"Name": "param1"}]`,
			PipelineId:  DefaultFakePipelineId,
			Status:      model.PipelineVersionReady,
			Description: "version_1",
		},
	)

	// Create "version_3" with DefaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_2" with DefaultFakePipelineIdFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_4" with DefaultFakePipelineIdFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// List result in 2 pages: first page "version_4" and "version_3"; second
	// page "version_2" and "version_1".
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err := pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		{
			UUID:           DefaultFakePipelineIdFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		{
			UUID:           DefaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err = pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// Second Page.
	assert.Equal(
		t, pipelineVersions, []*model.PipelineVersion{
			{
				UUID:           DefaultFakePipelineIdFour,
				CreatedAtInSec: 4,
				Name:           "pipeline_version_2",
				Parameters:     `[{"Name": "param1"}]`,
				PipelineId:     DefaultFakePipelineId,
				Status:         model.PipelineVersionReady,
			},
			{
				UUID:           DefaultFakePipelineIdTwo,
				CreatedAtInSec: 2,
				Name:           "pipeline_version_1",
				Description:    "version_1",
				Parameters:     `[{"Name": "param1"}]`,
				PipelineId:     DefaultFakePipelineId,
				Status:         model.PipelineVersionReady,
			},
		},
	)
}

func TestListPipelineVersions_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err := pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		{
			UUID:           DefaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     DefaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_WithFilter(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create "version_1" with DefaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Create "version_2" with DefaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Filter for name being equal to pipeline_version_1
	equalFilterProto := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:   "name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_version_1"},
			},
		},
	}
	equalFilter, _ := filter.New(equalFilterProto)

	// Filter for name prefix being pipeline_version
	prefixFilterProto := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_version"},
			},
		},
	}
	prefixFilter, _ := filter.New(prefixFilterProto)

	// Only return 1 pipeline version with equal filter.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", equalFilter)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err := pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)

	// Return 2 pipeline versions without filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, totalSize)

	// Return 2 pipeline versions with prefix filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", prefixFilter)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, totalSize)
}

func TestListPipelineVersionsError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = pipelineStore.ListPipelineVersions(DefaultFakePipelineId, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineVersionStatus(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline_1",
			Status: model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineIdTwo, nil)
	pipelineVersion, _ := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: DefaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		},
	)

	// Change version to deleting status
	err := pipelineStore.UpdatePipelineVersionStatus(
		pipelineVersion.UUID, model.PipelineVersionDeleting)
	assert.Nil(t, err)

	// Check the new status by retrieving this pipeline version.
	retrievedPipelineVersion, err := pipelineStore.GetPipelineVersionWithStatus(
		pipelineVersion.UUID, model.PipelineVersionDeleting)
	assert.Nil(t, err)
	assert.Equal(t, *retrievedPipelineVersion, model.PipelineVersion{
		UUID:           DefaultFakePipelineIdTwo,
		Name:           "pipeline_version_1",
		CreatedAtInSec: 2,
		Parameters:     `[{"Name": "param1"}]`,
		PipelineId:     DefaultFakePipelineId,
		Status:         model.PipelineVersionDeleting,
	})
}

func TestUpdatePipelineVersionStatusError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	err := pipelineStore.UpdatePipelineVersionStatus(
		DefaultFakePipelineId, model.PipelineVersionDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

var v2SpecHelloWorld = `components:
comp-hello-world:
  executorLabel: exec-hello-world
  inputDefinitions:
	parameters:
	  param1:
		parameterType: STRING
deploymentSpec:
executors:
  exec-hello-world:
	container:
	  args:
	  - "--param1"
	  - "{{$.inputs.parameters['param1']}}"
	  command:
	  - sh
	  - "-ec"
	  - |
		program_path=$(mktemp)
		printf "%s" "$0" > "$program_path"
		python3 -u "$program_path" "$@"
	  - |
		def hello_world(param1):
			print(param1)
			return param1

		import argparse
		_parser = argparse.ArgumentParser(prog='Hello world', description='')
		_parser.add_argument("--param1", dest="param1", type=str, required=True, default=argparse.SUPPRESS)
		_parsed_args = vars(_parser.parse_args())

		_outputs = hello_world(**_parsed_args)
	  image: python:3.7
pipelineInfo:
name: hello-world
root:
dag:
  tasks:
	hello-world:
	  cachingOptions:
		enableCache: true
	  componentRef:
		name: comp-hello-world
	  inputs:
		parameters:
		  param1:
			componentInputParameter: param1
	  taskInfo:
		name: hello-world
inputDefinitions:
  parameters:
	param1:
	  parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-1.6.5
`
var v1SpecHelloWorld = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: dup
        value: "value1"
    container:
      image: docker/whalesay:latest`
