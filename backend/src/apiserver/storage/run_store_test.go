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
	"fmt"
	"sort"
	"testing"

	sq "github.com/Masterminds/squirrel"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

type RunMetricSorter []*model.RunMetric

func (r RunMetricSorter) Len() int           { return len(r) }
func (r RunMetricSorter) Less(i, j int) bool { return r[i].Name < r[j].Name }
func (r RunMetricSorter) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

func initializeRunStore() (*DB, *RunStore) {
	db := NewFakeDbOrFatal()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "exp1"})
	expStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpIdTwo, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "exp2"})
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())

	run1 := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}
	run2 := &model.RunDetail{
		Run: model.Run{
			UUID:             "2",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run2",
			DisplayName:      "run2",
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}
	run3 := &model.RunDetail{
		Run: model.Run{
			UUID:             "3",
			ExperimentUUID:   defaultFakeExpIdTwo,
			Name:             "run3",
			DisplayName:      "run3",
			Namespace:        "n3",
			CreatedAtInSec:   3,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			ScheduledAtInSec: 3,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "3", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpIdTwo, ReferenceName: "e2",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow3",
		},
	}
	runStore.CreateRun(run1)
	runStore.CreateRun(run2)
	runStore.CreateRun(run3)

	metric1 := &model.RunMetric{
		RunUUID:     "1",
		NodeID:      "node1",
		Name:        "dummymetric",
		NumberValue: 1.0,
		Format:      "PERCENTAGE",
	}
	metric2 := &model.RunMetric{
		RunUUID:     "2",
		NodeID:      "node2",
		Name:        "dummymetric",
		NumberValue: 2.0,
		Format:      "PERCENTAGE",
	}
	runStore.ReportMetric(metric1)
	runStore.ReportMetric(metric2)

	return db, runStore
}

func TestListRuns_Pagination(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedFirstPageRuns := []*model.Run{
		{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}
	expectedSecondPageRuns := []*model.Run{
		{
			UUID:             "2",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run2",
			DisplayName:      "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "done",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "2",
					NodeID:      "node2",
					Name:        "dummymetric",
					NumberValue: 2.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}

	opts, err := list.NewOptions(&model.Run{}, 1, "", nil)
	assert.Nil(t, err)

	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_Pagination_WithSortingOnMetrics(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedFirstPageRuns := []*model.Run{
		{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}
	expectedSecondPageRuns := []*model.Run{
		{
			UUID:             "2",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run2",
			DisplayName:      "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "done",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "2",
					NodeID:      "node2",
					Name:        "dummymetric",
					NumberValue: 2.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}

	// Sort in asc order
	opts, err := list.NewOptions(&model.Run{}, 1, "metric:dummymetric", nil)
	assert.Nil(t, err)

	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)

	// Sort in desc order
	opts, err = list.NewOptions(&model.Run{}, 1, "metric:dummymetric desc", nil)
	assert.Nil(t, err)

	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_TotalSizeWithNoFilter(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	opts, _ := list.NewOptions(&model.Run{}, 4, "", nil)

	// No filter
	runs, total_size, _, err := runStore.ListRuns(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(runs))
	assert.Equal(t, 3, total_size)
}

func TestListRuns_TotalSizeWithFilter(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	// Add a filter
	opts, _ := list.NewOptions(&model.Run{}, 4, "", &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key: "name",
				Op:  api.Predicate_IN,
				Value: &api.Predicate_StringValues{
					StringValues: &api.StringValues{
						Values: []string{"run1", "run3"},
					},
				},
			},
		},
	})
	runs, total_size, _, err := runStore.ListRuns(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs))
	assert.Equal(t, 2, total_size)
}

func TestListRuns_Pagination_Descend(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedFirstPageRuns := []*model.Run{
		{
			UUID:             "2",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run2",
			DisplayName:      "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "done",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "2",
					NodeID:      "node2",
					Name:        "dummymetric",
					NumberValue: 2.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}
	expectedSecondPageRuns := []*model.Run{
		{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}

	opts, err := list.NewOptions(&model.Run{}, 1, "id desc", nil)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)

	for _, run := range runs {
		fmt.Printf("%+v\n", run)
	}

	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_Pagination_LessThanPageSize(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedRuns := []*model.Run{
		{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		{
			UUID:             "2",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run2",
			DisplayName:      "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "done",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "2",
					NodeID:      "node2",
					Name:        "dummymetric",
					NumberValue: 2.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}

	opts, err := list.NewOptions(&model.Run{}, 10, "", nil)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)
}

func TestListRunsError(t *testing.T) {
	db, runStore := initializeRunStore()
	db.Close()

	opts, err := list.NewOptions(&model.Run{}, 1, "", nil)
	_, _, _, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
}

func TestGetRun(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1"},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestGetRun_NotFoundError(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	_, err := runStore.GetRun("notfound")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected not to find the run")
}

func TestGetRun_InternalError(t *testing.T) {
	db, runStore := initializeRunStore()
	db.Close()

	_, err := runStore.GetRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get run to return internal error")
}

func TestCreateOrUpdateRun_UpdateSuccess(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1"},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)

	runDetail = &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			ScheduledAtInSec: 2, // This is will be ignored
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "done",
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1_done"},
	}
	err = runStore.CreateOrUpdateRun(runDetail)
	assert.Nil(t, err)

	expectedRun = &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "done",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1_done"},
	}

	runDetail, err = runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestCreateOrUpdateRun_CreateSuccess(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "exp1"})
	// Checking that the run is not yet in the DB
	_, err := runStore.GetRun("2000")
	assert.NotNil(t, err)

	runDetail := &model.RunDetail{
		Run: model.Run{
			UUID:           "2000",
			ExperimentUUID: defaultFakeExpId,
			Name:           "MY_NAME",
			Namespace:      "MY_NAMESPACE",
			CreatedAtInSec: 11,
			Conditions:     "Running",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: "workflow_spec",
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  "2000",
					ResourceType:  common.Run,
					ReferenceUUID: defaultFakeExpId,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow_runtime_spec",
		},
	}
	err = runStore.CreateOrUpdateRun(runDetail)
	assert.Nil(t, err)
	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:           "2000",
			ExperimentUUID: defaultFakeExpId,
			Name:           "MY_NAME",
			Namespace:      "MY_NAMESPACE",
			CreatedAtInSec: 11,
			Conditions:     "Running",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: "workflow_spec",
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  "2000",
					ResourceType:  common.Run,
					ReferenceUUID: defaultFakeExpId,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
			StorageState: api.Run_STORAGESTATE_AVAILABLE.String(),
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow_runtime_spec"},
	}

	runDetail, err = runStore.GetRun("2000")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestCreateOrUpdateRun_UpdateNotFound(t *testing.T) {
	db, runStore := initializeRunStore()
	db.Close()

	runDetail := &model.RunDetail{
		Run: model.Run{
			Conditions: "done",
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1_done"},
	}
	err := runStore.CreateOrUpdateRun(runDetail)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Error while creating or updating run")
}

func TestCreateOrUpdateRun_NoStorageStateValue(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	runDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             "1000",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}

	run, err := runStore.CreateRun(runDetail)
	assert.Nil(t, err)
	assert.Equal(t, run.StorageState, api.Run_STORAGESTATE_AVAILABLE.String())
}

func TestCreateOrUpdateRun_BadStorageStateValue(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	runDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			StorageState:     "bad value",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}

	_, err := runStore.CreateRun(runDetail)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid value for StorageState field")
}

func TestUpdateRun_RunNotExist(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	err := runStore.UpdateRun("not-exist", "done", 1, "workflow_done")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Row not found")
}

func TestTerminateRun(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	err := runStore.TerminateRun("1")
	assert.Nil(t, err)

	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Terminating",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1"},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestTerminateRun_RunDoesNotExist(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	err := runStore.TerminateRun("does-not-exist")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Row not found")
}

func TestTerminateRun_RunHasAlreadyFinished(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	err := runStore.TerminateRun("2")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Row not found")
}

func TestReportMetric_Success(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	metric := &model.RunMetric{
		RunUUID:     "1",
		NodeID:      "node1",
		Name:        "acurracy",
		NumberValue: 0.77,
		Format:      "PERCENTAGE",
	}
	runStore.ReportMetric(metric)

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err, "Got error: %+v", err)
	sort.Sort(RunMetricSorter(runDetail.Run.Metrics))
	assert.Equal(t, []*model.RunMetric{
		metric,
		{
			RunUUID:     "1",
			NodeID:      "node1",
			Name:        "dummymetric",
			NumberValue: 1.0,
			Format:      "PERCENTAGE",
		}}, runDetail.Run.Metrics)
}

func TestReportMetric_DupReports_Fail(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	metric1 := &model.RunMetric{
		RunUUID:     "1",
		NodeID:      "node1",
		Name:        "acurracy",
		NumberValue: 0.77,
		Format:      "PERCENTAGE",
	}
	metric2 := &model.RunMetric{
		RunUUID:     "1",
		NodeID:      "node1",
		Name:        "acurracy",
		NumberValue: 0.88,
		Format:      "PERCENTAGE",
	}
	runStore.ReportMetric(metric1)

	err := runStore.ReportMetric(metric2)
	_, ok := err.(*util.UserError)
	assert.True(t, ok)
}

func TestGetRun_InvalidMetricPayload_Ignore(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()
	sql, args, _ := sq.
		Insert("run_metrics").
		SetMap(sq.Eq{
			"RunUUID":     "1",
			"NodeID":      "node1",
			"Name":        "accuracy",
			"NumberValue": 0.88,
			"Format":      "RAW",
			"Payload":     "{ invalid; json,"}).ToSql()
	db.Exec(sql, args...)

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err, "Got error: %+v", err)
	assert.Empty(t, runDetail.Run.Metrics)
}

func TestListRuns_WithMetrics(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()
	metric1 := &model.RunMetric{
		RunUUID:     "1",
		NodeID:      "node1",
		Name:        "acurracy",
		NumberValue: 0.77,
		Format:      "PERCENTAGE",
	}
	metric2 := &model.RunMetric{
		RunUUID:     "1",
		NodeID:      "node2",
		Name:        "logloss",
		NumberValue: -1.2,
		Format:      "RAW",
	}
	metric3 := &model.RunMetric{
		RunUUID:     "2",
		NodeID:      "node2",
		Name:        "logloss",
		NumberValue: -1.3,
		Format:      "RAW",
	}
	runStore.ReportMetric(metric1)
	runStore.ReportMetric(metric2)
	runStore.ReportMetric(metric3)

	expectedRuns := []*model.Run{
		{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "Running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
				metric1,
				metric2},
		},
		{
			UUID:             "2",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run2",
			DisplayName:      "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "2",
					NodeID:      "node2",
					Name:        "dummymetric",
					NumberValue: 2.0,
					Format:      "PERCENTAGE",
				},
				metric3},
		},
	}

	opts, err := list.NewOptions(&model.Run{}, 2, "id", nil)
	assert.Nil(t, err)
	runs, total_size, _, err := runStore.ListRuns(&common.FilterContext{}, opts)
	assert.Equal(t, 3, total_size)
	assert.Nil(t, err)
	for _, run := range expectedRuns {
		sort.Sort(RunMetricSorter(run.Metrics))
	}
	for _, run := range runs {
		sort.Sort(RunMetricSorter(run.Metrics))
	}
	assert.Equal(t, expectedRuns, runs, "Unexpected Run listed.")
}

func TestArchiveRun(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()
	resourceReferenceStore := NewResourceReferenceStore(db)
	// Check resource reference exists
	r, err := resourceReferenceStore.GetResourceReference("1", common.Run, common.Experiment)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)

	// Archive run
	err = runStore.ArchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr := runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.Run.StorageState, api.Run_STORAGESTATE_ARCHIVED.String())

	// Check resource reference wasn't deleted
	_, err = resourceReferenceStore.GetResourceReference("1", common.Run, common.Experiment)
	assert.Nil(t, err)
}

func TestArchiveRun_InternalError(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	db.Close()

	err := runStore.ArchiveRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected archive run to return internal error")
}

func TestUnarchiveRun(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()
	resourceReferenceStore := NewResourceReferenceStore(db)
	// Check resource reference exists
	r, err := resourceReferenceStore.GetResourceReference("1", common.Run, common.Experiment)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)

	// Archive run
	err = runStore.ArchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr := runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.Run.StorageState, api.Run_STORAGESTATE_ARCHIVED.String())

	// Unarchive it back
	err = runStore.UnarchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr = runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.Run.StorageState, api.Run_STORAGESTATE_AVAILABLE.String())

	// Check resource reference wasn't deleted
	_, err = resourceReferenceStore.GetResourceReference("1", common.Run, common.Experiment)
	assert.Nil(t, err)
}

func TestUnarchiveRun_InternalError(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	db.Close()

	err := runStore.UnarchiveRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected unarchive run to return internal error")
}

func TestArchiveRun_IncludedInRunList(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	// Archive run
	err := runStore.ArchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr := runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.Run.StorageState, api.Run_STORAGESTATE_ARCHIVED.String())

	expectedRuns := []*model.Run{
		{
			UUID:             "1",
			ExperimentUUID:   defaultFakeExpId,
			Name:             "run1",
			DisplayName:      "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			StorageState:     api.Run_STORAGESTATE_ARCHIVED.String(),
			Conditions:       "Running",
			Metrics: []*model.RunMetric{
				{
					RunUUID:     "1",
					NodeID:      "node1",
					Name:        "dummymetric",
					NumberValue: 1.0,
					Format:      "PERCENTAGE",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
					ReferenceType: common.Experiment, Relationship: common.Creator,
				},
			},
		}}
	opts, err := list.NewOptions(&model.Run{}, 1, "", nil)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedRuns, runs)
	assert.NotEmpty(t, nextPageToken)
}

func TestDeleteRun(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()
	resourceReferenceStore := NewResourceReferenceStore(db)
	// Check resource reference exists
	r, err := resourceReferenceStore.GetResourceReference("1", common.Run, common.Experiment)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)

	// Delete run
	err = runStore.DeleteRun("1")
	assert.Nil(t, err)
	_, err = runStore.GetRun("1")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Run 1 not found")

	// Check resource reference deleted
	_, err = resourceReferenceStore.GetResourceReference("1", common.Run, common.Experiment)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_InternalError(t *testing.T) {
	db, runStore := initializeRunStore()
	db.Close()

	err := runStore.DeleteRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete run to return internal error")
}
