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
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	defaultFakeRunId      = "123e4567-e89b-12d3-a456-426655440020"
	defaultFakeRunIdTwo   = "123e4567-e89b-12d3-a456-426655440021"
	defaultFakeRunIdThree = "123e4567-e89b-12d3-a456-426655440023"
)

type RunMetricSorter []*model.RunMetric

func (r RunMetricSorter) Len() int           { return len(r) }
func (r RunMetricSorter) Less(i, j int) bool { return r[i].Name < r[j].Name }
func (r RunMetricSorter) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

var storeEnv *RuneStoreMysqlTestEnv

func TestMain(m *testing.M) {
	ctx := context.Background()
	storeEnv = NewRuneStoreMysqlSetupOrFatal(ctx)
	code := m.Run()
	storeEnv.StopOrFatal(ctx)
	os.Exit(code)
}

func initializeStorageAndCloseConnection() (*RunStore, error) {
	/*
		This function is used in fail-expected tests,
		where we intentionally expect an error to be throw
		due to operations on a closed database connection
	*/
	db := storeEnv.OpenExtraDbOrFatal()
	runStore := initializeRunStore(db)
	err := db.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close mysql connection: %v", err)
	}
	return runStore, nil
}

func initializeRunStore(db *DB) *RunStore {
	if db == nil {
		glog.Fatal("failed to initialize RunStore: db is nil")
	}
	storeEnv.CleanStorageOrFatal()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	_, err := expStore.CreateExperiment(&model.Experiment{Name: "exp1"})
	if err != nil {
		glog.Fatal("failed to create experiment:", err)
	}
	expStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpIdTwo, nil))
	_, err = expStore.CreateExperiment(&model.Experiment{Name: "exp2"})
	if err != nil {
		glog.Fatal("failed to create experiment:", err)
	}
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())

	run1 := &model.Run{
		UUID:         "1",
		ExperimentId: defaultFakeExpId,
		K8SName:      "run1",
		DisplayName:  "run1",
		StorageState: model.StorageStateAvailable,
		Namespace:    "n1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			Conditions:              "Running",
			State:                   model.RuntimeStateRunning,
			WorkflowRuntimeManifest: "workflow1",
		},
		PipelineSpec: model.PipelineSpec{
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   `[{"name":"param2","value":"world1"}]`,
				PipelineRoot: "gs://my-bucket/path/to/root/run1",
			},
		},
	}
	run2 := &model.Run{
		UUID:         "2",
		ExperimentId: defaultFakeExpId,
		K8SName:      "run2",
		DisplayName:  "run2",
		StorageState: model.StorageStateAvailable,
		Namespace:    "n2",
		RunDetails: model.RunDetails{
			CreatedAtInSec:          2,
			ScheduledAtInSec:        2,
			Conditions:              "Succeeded",
			State:                   model.RuntimeStateSucceeded,
			WorkflowRuntimeManifest: "workflow1",
		},
		PipelineSpec: model.PipelineSpec{
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   `[{"name":"param2","value":"world2"}]`,
				PipelineRoot: "gs://my-bucket/path/to/root/run2",
			},
		},
	}

	run3 := &model.Run{
		UUID:         "3",
		ExperimentId: defaultFakeExpIdTwo,
		K8SName:      "run3",
		DisplayName:  "run3",
		Namespace:    "n3",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          3,
			ScheduledAtInSec:        3,
			Conditions:              "Succeeded",
			State:                   model.RuntimeStateSucceeded,
			WorkflowRuntimeManifest: "workflow3",
		},
		PipelineSpec: model.PipelineSpec{
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   `[{"name":"param2","value":"world3"}]`,
				PipelineRoot: "gs://my-bucket/path/to/root/run3",
			},
		},
	}
	err = createRuns(runStore, run1, run2, run3)
	if err != nil {
		glog.Fatal("initialize run store error while creating metric:", err)
	}

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
	err = createMetrics(runStore, metric1, metric2)
	if err != nil {
		glog.Fatal("initialize run store error while creating metric:", err)
	}
	return runStore
}

func TestListRuns_Pagination(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	expectedFirstPageRuns := []*model.Run{
		{
			UUID:         "1",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run1",
			DisplayName:  "run1",
			Namespace:    "n1",
			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          1,
				ScheduledAtInSec:        1,
				Conditions:              "Running",
				State:                   model.RuntimeStateRunning,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 1,
						State:           model.RuntimeStateRunning,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run1",
				},
			},
		},
	}
	expectedFirstPageRuns[0] = expectedFirstPageRuns[0].ToV1()

	expectedSecondPageRuns := []*model.Run{
		{
			UUID:         "2",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run2",
			DisplayName:  "run2",
			Namespace:    "n2",
			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          2,
				ScheduledAtInSec:        2,
				Conditions:              "Succeeded",
				State:                   model.RuntimeStateSucceeded,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 2,
						State:           model.RuntimeStateSucceeded,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world2\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run2",
				},
			},
		},
	}
	expectedSecondPageRuns[0] = expectedSecondPageRuns[0].ToV1()

	opts, err := list.NewOptions(&model.Run{}, 1, "", nil)
	assert.NoError(t, err)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, total_size)
	runs[0] = runs[0].ToV1()
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	runs[0] = runs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_Pagination_WithSortingOnMetrics(t *testing.T) {
	t.Skip("Skipped due to incompatibility with MySQL")
	//False positive: this test passes on SQLite but fails on MySQL due to stricter GROUP BY semantics.
	runStore := initializeRunStore(storeEnv.Db)

	expectedFirstPageRuns := []*model.Run{
		{
			UUID:         "1",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run1",
			DisplayName:  "run1",
			Namespace:    "n1",
			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          1,
				ScheduledAtInSec:        1,
				Conditions:              "Running",
				State:                   model.RuntimeStateRunning,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 1,
						State:           model.RuntimeStateRunning,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run1",
				},
			},
		},
	}
	expectedFirstPageRuns[0] = expectedFirstPageRuns[0].ToV1()
	expectedSecondPageRuns := []*model.Run{
		{
			UUID:         "2",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run2",
			DisplayName:  "run2",
			StorageState: model.StorageStateAvailable,
			Namespace:    "n2",
			RunDetails: model.RunDetails{
				CreatedAtInSec:          2,
				ScheduledAtInSec:        2,
				Conditions:              "Succeeded",
				State:                   model.RuntimeStateSucceeded,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 2,
						State:           model.RuntimeStateSucceeded,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world2\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run2",
				},
			},
		},
	}
	expectedSecondPageRuns[0] = expectedSecondPageRuns[0].ToV1()

	// Sort in asc order
	opts, err := list.NewOptions(&model.Run{}, 1, "metric:dummymetric", nil)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	assert.NoError(t, err)
	runs[0] = runs[0].ToV1()
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.NoError(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	assert.NoError(t, err)
	runs[0] = runs[0].ToV1()
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed")
	assert.Empty(t, nextPageToken)

	// Sort in desc order
	opts, err = list.NewOptions(&model.Run{}, 1, "metric:dummymetric desc", nil)
	assert.Nil(t, err)

	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	assert.NoError(t, err)
	runs[0] = runs[0].ToV1()
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	runs[0] = runs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_TotalSizeWithNoFilter(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	opts, _ := list.NewOptions(&model.Run{}, 4, "", nil)

	// No filter
	runs, total_size, _, err := runStore.ListRuns(&model.FilterContext{}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(runs))
	assert.Equal(t, 3, total_size)
}

func TestListRuns_TotalSizeWithFilter(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	// Add a filter
	filterProto := &api.Filter{
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
	}
	newFilter, _ := filter.New(filterProto)
	opts, _ := list.NewOptions(&model.Run{}, 4, "", newFilter)
	runs, total_size, _, err := runStore.ListRuns(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs))
	assert.Equal(t, 2, total_size)
}

func TestListRuns_Pagination_Descend(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	expectedFirstPageRuns := []*model.Run{
		{
			UUID:         "2",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run2",
			DisplayName:  "run2",
			Namespace:    "n2",
			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          2,
				ScheduledAtInSec:        2,
				Conditions:              "Succeeded",
				State:                   model.RuntimeStateSucceeded,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 2,
						State:           model.RuntimeStateSucceeded,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world2\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run2",
				},
			},
		},
	}
	expectedFirstPageRuns[0] = expectedFirstPageRuns[0].ToV1()
	expectedSecondPageRuns := []*model.Run{
		{
			UUID:         "1",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run1",
			DisplayName:  "run1",
			Namespace:    "n1",
			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          1,
				ScheduledAtInSec:        1,
				Conditions:              "Running",
				State:                   model.RuntimeStateRunning,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 1,
						State:           model.RuntimeStateRunning,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run1",
				},
			},
		},
	}
	expectedSecondPageRuns[0] = expectedSecondPageRuns[0].ToV1()

	opts, err := list.NewOptions(&model.Run{}, 1, "id desc", nil)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	for i, run := range runs {
		runs[i] = run.ToV1()
		fmt.Printf("%+v\n", run)
	}

	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err = runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	runs[0] = runs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_Pagination_LessThanPageSize(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	expectedRuns := []*model.Run{
		{
			UUID:         "1",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run1",
			DisplayName:  "run1",
			Namespace:    "n1",

			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          1,
				ScheduledAtInSec:        1,
				State:                   model.RuntimeStateRunning,
				Conditions:              "Running",
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 1,
						State:           model.RuntimeStateRunning,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run1",
				},
			},
		},
		{
			UUID:         "2",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run2",
			DisplayName:  "run2",
			Namespace:    "n2",

			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          2,
				ScheduledAtInSec:        2,
				State:                   model.RuntimeStateSucceeded,
				Conditions:              "Succeeded",
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 2,
						State:           model.RuntimeStateSucceeded,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world2\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run2",
				},
			},
		},
	}
	expectedRuns[0] = expectedRuns[0].ToV1()
	expectedRuns[1] = expectedRuns[1].ToV1()

	opts, err := list.NewOptions(&model.Run{}, 10, "", nil)
	assert.Nil(t, err)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)

	runs[0] = runs[0].ToV1()
	runs[1] = runs[1].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedRuns, runs, "Unexpected Run listed")
	assert.Empty(t, nextPageToken)
}

func TestListRunsError(t *testing.T) {
	runStore, err := initializeStorageAndCloseConnection()
	assert.NoError(t, err)

	opts, err := list.NewOptions(&model.Run{}, 1, "", nil)
	_, _, _, err = runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
}

func TestGetRun(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	expectedRun := &model.Run{
		UUID:         "1",
		ExperimentId: defaultFakeExpId,
		K8SName:      "run1",
		DisplayName:  "run1",
		Namespace:    "n1",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: "workflow1",
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			Conditions:              "Running",
			State:                   model.RuntimeStateRunning,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 1,
					State:           model.RuntimeStateRunning,
				},
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
		},
		PipelineSpec: model.PipelineSpec{
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
				PipelineRoot: "gs://my-bucket/path/to/root/run1",
			},
		},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun.ToV1(), runDetail.ToV1())
}

func TestGetRun_NotFoundError(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	_, err := runStore.GetRun("notfound")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected not to find the run")
}

func TestGetRun_InternalError(t *testing.T) {
	runStore, err := initializeStorageAndCloseConnection()
	assert.NoError(t, err)

	_, err = runStore.GetRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get run to return internal error")
}

func TestCreateAndUpdateRun_UpdateSuccess(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	expectedRun := &model.Run{
		UUID:         "1",
		ExperimentId: defaultFakeExpId,
		K8SName:      "run1",
		DisplayName:  "run1",
		Namespace:    "n1",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			Conditions:              "Running",
			State:                   model.RuntimeStateRunning,
			WorkflowRuntimeManifest: "workflow1",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 1,
					State:           model.RuntimeStateRunning,
				},
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
		},
		PipelineSpec: model.PipelineSpec{
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
				PipelineRoot: "gs://my-bucket/path/to/root/run1",
			},
		},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun.ToV1(), runDetail.ToV1())

	runDetail = &model.Run{
		UUID:         "1",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			FinishedAtInSec:         100,
			WorkflowRuntimeManifest: "workflow1_done",
			Conditions:              "Succeeded",
			ScheduledAtInSec:        200, // This is will be ignored
			State:                   model.RuntimeStateSucceeded,
		},
	}

	err = runStore.UpdateRun(runDetail)
	assert.Nil(t, err)

	expectedRun = &model.Run{
		UUID:         "1",
		ExperimentId: defaultFakeExpId,
		K8SName:      "run1",
		DisplayName:  "run1",
		Namespace:    "n1",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			FinishedAtInSec:         100,
			Conditions:              "Succeeded",
			State:                   model.RuntimeStateSucceeded,
			WorkflowRuntimeManifest: "workflow1_done",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 4,
					State:           model.RuntimeStateSucceeded,
				},
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
		},
		PipelineSpec: model.PipelineSpec{
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
				PipelineRoot: "gs://my-bucket/path/to/root/run1",
			},
		},
	}

	runDetail, err = runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun.ToV1(), runDetail.ToV1())
}

func TestCreateAndUpdateRun_CreateSuccess(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	expStore := NewExperimentStore(storeEnv.Db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "exp1"})
	// Checking that the run is not yet in the DB
	_, err := runStore.GetRun("2000")
	assert.NotNil(t, err)

	runDetail := &model.Run{
		UUID:         "2000",
		ExperimentId: defaultFakeExpId,
		K8SName:      "MY_NAME",
		Namespace:    "MY_NAMESPACE",
		RunDetails: model.RunDetails{
			CreatedAtInSec:          11,
			Conditions:              "Running",
			State:                   model.RuntimeStateRunning,
			WorkflowRuntimeManifest: "workflow_runtime_spec",
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "workflow_spec",
		},
	}

	err = runStore.UpdateRun(runDetail)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Run 2000 not found")
	_, err = runStore.CreateRun(runDetail)
	assert.Nil(t, err)
	expectedRun := &model.Run{
		UUID:         "2000",
		ExperimentId: defaultFakeExpId,
		K8SName:      "MY_NAME",
		Namespace:    "MY_NAMESPACE",
		RunDetails: model.RunDetails{
			CreatedAtInSec:          11,
			Conditions:              "Running",
			State:                   model.RuntimeStateRunning,
			WorkflowRuntimeManifest: "workflow_runtime_spec",
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 4,
					State:           model.RuntimeStateRunning,
				},
			},
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "workflow_spec",
		},
		StorageState: model.StorageStateAvailable,
	}

	runDetail, err = runStore.GetRun("2000")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun.ToV1(), runDetail.ToV1())
}

func TestCreateAndUpdateRun_UpdateNotFound(t *testing.T) {
	runStore, err := initializeStorageAndCloseConnection()
	assert.NoError(t, err)

	run := &model.Run{
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: "workflow1_done",
			Conditions:              "Succeeded",
			State:                   model.RuntimeStateSucceeded,
		},
	}
	_, err = runStore.CreateRun(run)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create a new transaction to create run")
	err = runStore.UpdateRun(&model.Run{DisplayName: "Test display name"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "transaction creation failed")
}

func TestCreateOrUpdateRun_NoStorageStateValue(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	runDetail := &model.Run{
		UUID:         "1000",
		K8SName:      "run1",
		ExperimentId: defaultFakeExpId,
		Namespace:    "n1",
		RunDetails: model.RunDetails{
			WorkflowRuntimeManifest: "workflow1",
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			Conditions:              "Running",
			State:                   model.RuntimeStateRunning,
		},
	}

	run, err := runStore.CreateRun(runDetail)
	assert.Nil(t, err)
	assert.Equal(t, model.StorageStateAvailable, run.StorageState)
}

func TestCreateOrUpdateRun_DuplicateUUID(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	runDetail := &model.Run{
		UUID:         "1",
		ExperimentId: defaultFakeExpId,
		K8SName:      "run1",
		StorageState: "bad value",
		Namespace:    "n1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			Conditions:              "Running",
			WorkflowRuntimeManifest: "workflow1",
			State:                   model.RuntimeStateRunning,
		},
		Metrics: []*model.RunMetric{
			{
				RunUUID:     "1",
				NodeID:      "node1",
				Name:        "dummymetric",
				NumberValue: 1.0,
				Format:      "PERCENTAGE",
			},
		},
	}

	_, err := runStore.CreateRun(runDetail)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Duplicate entry '1' for key 'run_details.PRIMARY'")
}

func TestUpdateRun_RunNotExist(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	err := runStore.UpdateRun(&model.Run{UUID: "not-exist", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded}})
	assert.NotNil(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	assert.Contains(t, err.Error(), "not found")
}

func TestTerminateRun(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	err := runStore.TerminateRun("1")
	assert.Nil(t, err)

	expectedRun := &model.Run{
		UUID:         "1",
		ExperimentId: defaultFakeExpId,
		K8SName:      "run1",
		DisplayName:  "run1",
		Namespace:    "n1",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			Conditions:              "Terminating",
			WorkflowRuntimeManifest: "workflow1",
			State:                   model.RuntimeStateCancelling,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 1,
					State:           model.RuntimeStateRunning,
				},
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
		},
		PipelineSpec: model.PipelineSpec{
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
				PipelineRoot: "gs://my-bucket/path/to/root/run1",
			},
		},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun.ToV1(), runDetail.ToV1())
}

func TestTerminateRun_RunDoesNotExist(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	err := runStore.TerminateRun("does-not-exist")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Row not found")
}

func TestTerminateRun_RunHasAlreadyFinished(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	err := runStore.TerminateRun("2")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Row not found")
}

func TestCreateMetric_Success(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	metric := &model.RunMetric{
		RunUUID:     "1",
		NodeID:      "node1",
		Name:        "acurracy",
		NumberValue: 0.77,
		Format:      "PERCENTAGE",
	}
	runStore.CreateMetric(metric)

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err, "Got error: %+v", err)
	sort.Sort(RunMetricSorter(runDetail.Metrics))
	assert.Equal(t, []*model.RunMetric{
		metric,
		{
			RunUUID:     "1",
			NodeID:      "node1",
			Name:        "dummymetric",
			NumberValue: 1.0,
			Format:      "PERCENTAGE",
		},
	}, runDetail.Metrics)
}

func TestCreateMetric_DupReports_Fail(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

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
	runStore.CreateMetric(metric1)

	err := runStore.CreateMetric(metric2)
	_, ok := err.(*util.UserError)
	assert.True(t, ok)
}

func TestGetRun_InvalidMetricPayload_Ignore(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)

	sql, args, _ := sq.
		Insert("run_metrics").
		SetMap(sq.Eq{
			"RunUUID":     "1",
			"NodeID":      "node1",
			"Name":        "accuracy",
			"NumberValue": 0.88,
			"Format":      "RAW",
			"Payload":     "{ invalid; json,",
		}).ToSql()
	storeEnv.Db.Exec(sql, args...)

	run, err := runStore.GetRun("1")
	assert.Nil(t, err, "Got error: %+v", err)
	assert.Empty(t, run.Metrics)
}

func TestListRuns_WithMetrics(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
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
	err := createMetrics(runStore, metric1, metric2, metric3)
	assert.NoError(t, err)

	expectedRuns := []*model.Run{
		{
			UUID:         "1",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run1",
			DisplayName:  "run1",
			Namespace:    "n1",
			StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          1,
				ScheduledAtInSec:        1,
				Conditions:              "Running",
				State:                   model.RuntimeStateRunning,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 1,
						State:           model.RuntimeStateRunning,
					},
				},
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run1",
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
				metric2,
			},
		},
		{
			UUID:         "2",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run2",
			DisplayName:  "run2",
			Namespace:    "n2",
			StorageState: model.StorageStateAvailable,

			RunDetails: model.RunDetails{
				CreatedAtInSec:          2,
				ScheduledAtInSec:        2,
				Conditions:              "Succeeded",
				State:                   model.RuntimeStateSucceeded,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 2,
						State:           model.RuntimeStateSucceeded,
					},
				},
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world2\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run2",
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
				metric3,
			},
		},
	}
	expectedRuns[0] = expectedRuns[0].ToV1()
	expectedRuns[1] = expectedRuns[1].ToV1()

	opts, err := list.NewOptions(&model.Run{}, 2, "id", nil)
	assert.Nil(t, err)
	runs, total_size, _, err := runStore.ListRuns(&model.FilterContext{}, opts)
	runs[0] = runs[0].ToV1()
	runs[1] = runs[1].ToV1()
	assert.Equal(t, 3, total_size)
	assert.Nil(t, err)
	for _, run := range expectedRuns {
		sort.Sort(RunMetricSorter(run.Metrics))
	}
	for _, run := range runs {
		sort.Sort(RunMetricSorter(run.Metrics))
	}
	assert.Equal(t, expectedRuns, runs, "Unexpected Run listed")
}

func TestArchiveRun(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	resourceReferenceStore := NewResourceReferenceStore(storeEnv.Db, nil)
	// Check resource reference exists
	r, err := resourceReferenceStore.GetResourceReference("1", model.RunResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)

	// Archive run
	err = runStore.ArchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr := runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.StorageState, model.StorageStateArchived)

	// Check resource reference wasn't deleted
	_, err = resourceReferenceStore.GetResourceReference("1", model.RunResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
}

func TestArchiveRun_InternalError(t *testing.T) {
	runStore, err := initializeStorageAndCloseConnection()
	assert.NoError(t, err)
	err = runStore.ArchiveRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected archive run to return internal error")
}

func TestUnarchiveRun(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	resourceReferenceStore := NewResourceReferenceStore(storeEnv.Db, nil)
	// Check resource reference exists
	r, err := resourceReferenceStore.GetResourceReference("1", model.RunResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)

	// Archive run
	err = runStore.ArchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr := runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.StorageState, model.StorageStateArchived)

	// Unarchive it back
	err = runStore.UnarchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr = runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.StorageState, model.StorageStateAvailable)

	// Check resource reference wasn't deleted
	_, err = resourceReferenceStore.GetResourceReference("1", model.RunResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
}

func TestUnarchiveRun_InternalError(t *testing.T) {
	runStore, err := initializeStorageAndCloseConnection()
	assert.NoError(t, err)
	err = runStore.UnarchiveRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected unarchive run to return internal error")
}

func TestArchiveRun_IncludedInRunList(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	// Archive run
	err := runStore.ArchiveRun("1")
	assert.Nil(t, err)
	run, getRunErr := runStore.GetRun("1")
	assert.Nil(t, getRunErr)
	assert.Equal(t, run.StorageState, model.StorageStateArchived)

	expectedRuns := []*model.Run{
		{
			UUID:         "1",
			ExperimentId: defaultFakeExpId,
			K8SName:      "run1",
			DisplayName:  "run1",
			Namespace:    "n1",
			StorageState: model.StorageStateArchived,

			RunDetails: model.RunDetails{
				CreatedAtInSec:          1,
				ScheduledAtInSec:        1,
				Conditions:              "Running",
				State:                   model.RuntimeStateRunning,
				WorkflowRuntimeManifest: "workflow1",
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 1,
						State:           model.RuntimeStateRunning,
					},
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
			},
			PipelineSpec: model.PipelineSpec{
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "[{\"name\":\"param2\",\"value\":\"world1\"}]",
					PipelineRoot: "gs://my-bucket/path/to/root/run1",
				},
			},
		},
	}
	expectedRuns[0] = expectedRuns[0].ToV1()
	opts, err := list.NewOptions(&model.Run{}, 1, "", nil)
	runs, total_size, nextPageToken, err := runStore.ListRuns(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	runs[0] = runs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedRuns, runs)
	assert.NotEmpty(t, nextPageToken)
}

func TestDeleteRun(t *testing.T) {
	runStore := initializeRunStore(storeEnv.Db)
	resourceReferenceStore := NewResourceReferenceStore(storeEnv.Db, nil)
	// Check resource reference exists
	r, err := resourceReferenceStore.GetResourceReference("1", model.RunResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)

	// Delete run
	err = runStore.DeleteRun("1")
	assert.Nil(t, err)
	_, err = runStore.GetRun("1")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Run 1 not found")

	// Check resource reference deleted
	_, err = resourceReferenceStore.GetResourceReference("1", model.RunResourceType, model.ExperimentResourceType)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteRun_InternalError(t *testing.T) {
	runStore, err := initializeStorageAndCloseConnection()
	assert.NoError(t, err)
	err = runStore.DeleteRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete run to return internal error")
}

func TestParseMetrics(t *testing.T) {
	expectedModelRunMetrics := []*model.RunMetric{
		{
			RunUUID:     "run-1",
			Name:        "metric-1",
			NodeID:      "node-1",
			NumberValue: 0.88,
			Format:      "RAW",
		},
	}
	metricsByte, _ := json.Marshal(expectedModelRunMetrics)
	metricsString := string(metricsByte)
	metricsNullString := sql.NullString{
		Valid:  true,
		String: metricsString,
	}
	parsedMetrics, err := parseMetrics(metricsNullString)
	assert.Nil(t, err)
	assert.Equal(t, expectedModelRunMetrics, parsedMetrics)
}

func TestParseRuntimeConfig(t *testing.T) {
	expectedRuntimeConfig := model.RuntimeConfig{
		Parameters:   `[{"name":"param2","value":"world1"}]`,
		PipelineRoot: "gs://my-bucket/path/to/root/run1",
	}
	parametersNullString := sql.NullString{
		Valid:  true,
		String: `[{"name":"param2","value":"world1"}]`,
	}
	pipelineRootNullString := sql.NullString{
		Valid:  true,
		String: "gs://my-bucket/path/to/root/run1",
	}
	actualRuntimeConfig := parseRuntimeConfig(parametersNullString, pipelineRootNullString)
	assert.Equal(t, expectedRuntimeConfig, actualRuntimeConfig)
}

func TestParseResourceReferences(t *testing.T) {
	expectedResourceReferences := []*model.ResourceReference{
		{
			ResourceUUID: "2", ResourceType: model.RunResourceType,
			ReferenceUUID: defaultFakeExpId, ReferenceName: "e1",
			ReferenceType: model.ExperimentResourceType, Relationship: model.CreatorRelationship,
		},
	}
	resourceReferencesBytes, _ := json.Marshal(expectedResourceReferences)
	resourceReferencesString := string(resourceReferencesBytes)
	resourceReferencesNullString := sql.NullString{
		Valid:  true,
		String: resourceReferencesString,
	}
	actualResourceReferences, err := parseResourceReferences(resourceReferencesNullString)
	assert.Nil(t, err)
	assert.Equal(t, expectedResourceReferences, actualResourceReferences)
}

func TestRunAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.Run{}).APIToModelFieldMap() {
		assert.Contains(t, runColumns, modelField)
	}
}
