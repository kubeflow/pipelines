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

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

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
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
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
			Name:             "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
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
			Name:             "run3",
			Namespace:        "n3",
			CreatedAtInSec:   3,
			ScheduledAtInSec: 3,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "3", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpIdTwo, ReferenceType: common.Experiment,
					Relationship: common.Creator,
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
	return db, runStore
}

func TestListRuns_Pagination(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedFirstPageRuns := []model.Run{
		{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
		}}
	expectedSecondPageRuns := []model.Run{
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
		}}
	runs, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}},
		&common.PaginationContext{
			PageSize:        1,
			KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
			SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
			IsDesc:          false,
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	runs, nextPageToken, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}},
		&common.PaginationContext{
			Token: &common.Token{
				SortByFieldValue: "2",
				// The value of the key field of the next row to be returned.
				KeyFieldValue: "2"},
			PageSize:        1,
			KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
			SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
			IsDesc:          false,
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_Pagination_Descend(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedFirstPageRuns := []model.Run{
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
		}}
	expectedSecondPageRuns := []model.Run{
		{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
		}}
	runs, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}},
		&common.PaginationContext{
			PageSize:        1,
			KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
			SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
			IsDesc:          true,
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	runs, nextPageToken, err = runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}},
		&common.PaginationContext{
			Token: &common.Token{
				SortByFieldValue: "1",
				// The value of the key field of the next row to be returned.
				KeyFieldValue: "1"},
			PageSize:        1,
			KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
			SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
			IsDesc:          true,
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedSecondPageRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)
}

func TestListRuns_Pagination_LessThanPageSize(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedRuns := []model.Run{
		{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
		},
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
		}}
	runs, nextPageToken, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}},
		&common.PaginationContext{
			PageSize:        10,
			KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
			SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
			IsDesc:          false,
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedRuns, runs, "Unexpected Run listed.")
	assert.Empty(t, nextPageToken)
}

func TestListRunsError(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	db.Close()
	_, _, err := runStore.ListRuns(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: defaultFakeExpId}},
		&common.PaginationContext{
			PageSize:        1,
			KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
			SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
			IsDesc:          false,
		})
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
}

func TestGetRun(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
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
	defer db.Close()
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
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
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
			Conditions:       "done",
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1_done"},
	}
	err = runStore.CreateOrUpdateRun(runDetail)
	assert.Nil(t, err)

	expectedRun = &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
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
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow_runtime_spec"},
	}

	runDetail, err = runStore.GetRun("2000")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestCreateOrUpdateRun_UpdateNotFound(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()
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

func TestUpdateRun_RunNotExist(t *testing.T) {
	db, runStore := initializeRunStore()
	defer db.Close()

	err := runStore.UpdateRun("not-exist", "done", "workflow_done")
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
	assert.Equal(t, []*model.RunMetric{metric}, runDetail.Run.Metrics)
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

	expectedRuns := []model.Run{
		{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "1", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
			Metrics: []*model.RunMetric{metric1, metric2},
		},
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID: "2", ResourceType: common.Run,
					ReferenceUUID: defaultFakeExpId, ReferenceType: common.Experiment,
					Relationship: common.Creator,
				},
			},
			Metrics: []*model.RunMetric{metric3},
		},
	}
	runs, _, err := runStore.ListRuns(&common.FilterContext{}, &common.PaginationContext{
		PageSize:        2,
		KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
		SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
		IsDesc:          false,
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedRuns, runs, "Unexpected Run listed.")
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
	defer db.Close()

	db.Close()

	err := runStore.DeleteRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete run to return internal error")
}
