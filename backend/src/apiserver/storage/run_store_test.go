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
	"time"

	sq "github.com/Masterminds/squirrel"
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func initializePrepopulatedDB(runStore *RunStore) {
	run1 := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			JobID:            "1",
			Name:             "run1",
			Namespace:        "n1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}
	run2 := &model.RunDetail{
		Run: model.Run{
			UUID:             "2",
			JobID:            "1",
			Name:             "run2",
			Namespace:        "n2",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}
	run3 := &model.RunDetail{
		Run: model.Run{
			UUID:             "3",
			JobID:            "2",
			Name:             "run3",
			Namespace:        "n3",
			CreatedAtInSec:   3,
			ScheduledAtInSec: 3,
			Conditions:       "done",
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow3",
		},
	}
	runStore.CreateRun(run1)
	runStore.CreateRun(run2)
	runStore.CreateRun(run3)
}

func TestListRuns_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	expectedFirstPageRuns := []model.Run{
		{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		}}
	expectedSecondPageRuns := []model.Run{
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			JobID:            "1",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
		}}
	runs, nextPageToken, err := runStore.ListRuns(util.StringPointer("1"), &common.PaginationContext{
		PageSize:        1,
		KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
		SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
		IsDesc:          false,
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	runs, nextPageToken, err = runStore.ListRuns(util.StringPointer("1"), &common.PaginationContext{
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
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	expectedFirstPageRuns := []model.Run{
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			JobID:            "1",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
		}}
	expectedSecondPageRuns := []model.Run{
		{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		}}
	runs, nextPageToken, err := runStore.ListRuns(util.StringPointer("1"), &common.PaginationContext{
		PageSize:        1,
		KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
		SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
		IsDesc:          true,
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedFirstPageRuns, runs, "Unexpected Run listed.")
	assert.NotEmpty(t, nextPageToken)

	runs, nextPageToken, err = runStore.ListRuns(util.StringPointer("1"), &common.PaginationContext{
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
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	expectedRuns := []model.Run{
		{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			JobID:            "1",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
		}}
	runs, nextPageToken, err := runStore.ListRuns(util.StringPointer("1"), &common.PaginationContext{
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
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	db.Close()
	_, _, err := runStore.ListRuns(util.StringPointer("1"), &common.PaginationContext{
		PageSize:        1,
		KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
		SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
		IsDesc:          false,
	})
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
}

func TestGetRun(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1"},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestGetRun_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	_, err := runStore.GetRun("notfound")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected not to find the run")
}

func TestGetRun_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)
	db.Close()

	_, err := runStore.GetRun("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get run to return internal error")
}

func TestReportRun_UpdateSuccess(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow1"},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "n1",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/workflowEpoch": "100",
			},
			CreationTimestamp: metav1.NewTime(time.Unix(11, 0).UTC()),
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.NodeRunning,
		},
	})

	err = runStore.ReportRun(workflow)
	assert.Nil(t, err)

	expectedRun = &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running:",
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: workflow.ToStringForStore()},
	}

	runDetail, err = runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestReportRun_CreateSuccess(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	// Checking that the run is not yet in the DB
	runDetail, err := runStore.GetRun("2000")
	assert.NotNil(t, err)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "2000",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("3000"),
			}},
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/workflowEpoch": "100",
			},
			CreationTimestamp: metav1.NewTime(time.Unix(11, 0).UTC()),
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.NodeRunning,
		},
	})

	err = runStore.ReportRun(workflow)
	assert.Nil(t, err)
	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "2000",
			Name:             "MY_NAME",
			Namespace:        "MY_NAMESPACE",
			JobID:            "3000",
			CreatedAtInSec:   11,
			ScheduledAtInSec: 100,
			Conditions:       "Running:",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: workflow.GetSpec().ToStringForStore(),
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: workflow.ToStringForStore()},
	}

	runDetail, err = runStore.GetRun("2000")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestReportRun_UpdateError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)
	db.Close()

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/workflowEpoch": "100",
			},
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.NodeRunning,
		},
	})

	err := runStore.ReportRun(workflow)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Error while creating or updating run")
}

func TestReportRun_MostlyEmptySpec(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "n1",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
			CreationTimestamp: metav1.NewTime(time.Unix(11, 0).UTC()),
		},
	})

	err := runStore.ReportRun(workflow)
	assert.Nil(t, err)

	expectedRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "1",
			Name:             "run1",
			Namespace:        "n1",
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       ":",
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: workflow.ToStringForStore()},
	}

	runDetail, err := runStore.GetRun("1")
	assert.Nil(t, err)
	assert.Equal(t, expectedRun, runDetail)
}

func TestReportMetric_Success(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

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
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)

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
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)
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
	db := NewFakeDbOrFatal()
	defer db.Close()
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	initializePrepopulatedDB(runStore)
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
			JobID:            "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
			Metrics:          []*model.RunMetric{metric1, metric2},
		},
		{
			UUID:             "2",
			Name:             "run2",
			Namespace:        "n2",
			JobID:            "1",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
			Metrics:          []*model.RunMetric{metric3},
		},
	}
	runs, _, err := runStore.ListRuns(util.StringPointer("1"), &common.PaginationContext{
		PageSize:        2,
		KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
		SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
		IsDesc:          false,
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedRuns, runs, "Unexpected Run listed.")
}
