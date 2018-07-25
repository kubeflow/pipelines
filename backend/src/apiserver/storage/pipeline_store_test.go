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

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	swfapi "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func initializePipelineDB() *gorm.DB {
	db := NewFakeDbOrFatal()
	pipeline1 := &model.PipelineDetail{
		Pipeline: model.Pipeline{
			UUID:           "1",
			DisplayName:    "pp 1",
			Name:           "pp1",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 0,
			UpdatedAtInSec: 0,
		},
		ScheduledWorkflow: "scheduledworkflow1",
	}
	pipeline2 := &model.PipelineDetail{
		Pipeline: model.Pipeline{
			UUID:           "2",
			DisplayName:    "pp 2",
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
func TestListPipelines_Pagination(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	pipelinesExpected := []model.Pipeline{
		{
			UUID:           "1",
			DisplayName:    "pp 1",
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
	pipelinesExpected2 := []model.Pipeline{
		{
			UUID:           "2",
			DisplayName:    "pp 2",
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

func TestListPipelines_Pagination_LessThanPageSize(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	pipelinesExpected := []model.Pipeline{
		{
			UUID:           "1",
			DisplayName:    "pp 1",
			Name:           "pp1",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 0,
			UpdatedAtInSec: 0,
		},
		{
			UUID:           "2",
			DisplayName:    "pp 2",
			Name:           "pp2",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
		}}
	pipelines, nextPageToken, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, model.GetPipelineTablePrimaryKeyColumn() /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelinesError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	_, _, err := pipelineStore.ListPipelines("" /*pageToken*/, 2 /*pageSize*/, "Name" /*sortByFieldName*/)

	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to list pipeline to return error")
}

func TestGetPipeline(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	pipelineExpected := model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
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

func TestGetPipeline_NotFoundError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	_, err := pipelineStore.GetPipeline("notexist")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found error")
}

func TestGetPipeline_InternalError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	_, err := pipelineStore.GetPipeline("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestDeletePipeline(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	err := pipelineStore.DeletePipeline("1")
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipeline_InternalError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	db.Close()

	err := pipelineStore.DeletePipeline("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete pipeline to return internal error")
}

func TestCreatePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	pipeline := &model.Pipeline{
		UUID:        "1",
		DisplayName: "pp 1",
		Name:        "pp1",
		Namespace:   "n1",
		PackageId:   1,
		Enabled:     true,
	}

	pipeline, err := pipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	pipelineExpected := &model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
		Name:           "pp1",
		Namespace:      "n1",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	assert.Equal(t, pipelineExpected, pipeline, "Got unexpected pipelines")
}

func TestCreatePipelineError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	pipeline := &model.Pipeline{
		UUID:        "1",
		DisplayName: "pp 1",
		Name:        "pp1",
		Namespace:   "n1",
		PackageId:   1,
		Enabled:     true,
	}

	pipeline, err := pipelineStore.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestEnablePipeline(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	err := pipelineStore.EnablePipeline("1", false)
	assert.Nil(t, err)

	pipelineExpected := model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
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

func TestEnablePipeline_SkipUpdate(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	err := pipelineStore.EnablePipeline("1", true)
	assert.Nil(t, err)

	pipelineExpected := model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
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

func TestEnablePipeline_DatabaseError(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	db.Close()

	// Enabling the pipeline.
	err := pipelineStore.EnablePipeline("1", true)
	println(err.Error())
	assert.Contains(t, err.Error(), "Error when enabling pipeline 1 to true: sql: database is closed")
}

func TestUpdatePipeline_Success(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	pipelineExpected := model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
		Name:           "pp1",
		Namespace:      "n1",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 0,
		UpdatedAtInSec: 0,
	}

	pipeline, err := pipelineStore.GetPipeline("1")
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)

	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        false,
			MaxConcurrency: util.Int64Pointer(200),
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "PARAM1", Value: "NEW_VALUE1"},
				},
			},
			Trigger: swfapi.Trigger{
				CronSchedule: &swfapi.CronSchedule{
					StartTime: util.MetaV1TimePointer(metav1.NewTime(time.Unix(10, 0).UTC())),
					EndTime:   util.MetaV1TimePointer(metav1.NewTime(time.Unix(20, 0).UTC())),
					Cron:      "MY_CRON",
				},
				PeriodicSchedule: &swfapi.PeriodicSchedule{
					StartTime:      util.MetaV1TimePointer(metav1.NewTime(time.Unix(30, 0).UTC())),
					EndTime:        util.MetaV1TimePointer(metav1.NewTime(time.Unix(40, 0).UTC())),
					IntervalSecond: 50,
				},
			},
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{{
				Type:               swfapi.ScheduledWorkflowEnabled,
				Status:             core.ConditionTrue,
				LastProbeTime:      metav1.NewTime(time.Unix(10, 0).UTC()),
				LastTransitionTime: metav1.NewTime(time.Unix(20, 0).UTC()),
				Reason:             string(swfapi.ScheduledWorkflowEnabled),
				Message:            "The schedule is enabled.",
			},
			},
		},
	})

	err = pipelineStore.UpdatePipeline(swf)
	assert.Nil(t, err)

	pipelineExpected = model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
		Name:           "MY_NAME",
		Namespace:      "MY_NAMESPACE",
		PackageId:      1,
		Enabled:        false,
		Conditions:     "Enabled:",
		CreatedAtInSec: 0,
		UpdatedAtInSec: 1,
		MaxConcurrency: 200,
		Parameters:     "[{\"name\":\"PARAM1\",\"value\":\"NEW_VALUE1\"}]",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(10),
				CronScheduleEndTimeInSec:   util.Int64Pointer(20),
				Cron: util.StringPointer("MY_CRON"),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(30),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(40),
				IntervalSecond:                 util.Int64Pointer(50),
			},
		},
	}

	pipeline, err = pipelineStore.GetPipeline("1")
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)
}

func TestUpdatePipeline_MostlyEmptySpec(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	pipelineExpected := model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
		Name:           "pp1",
		Namespace:      "n1",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 0,
		UpdatedAtInSec: 0,
	}

	pipeline, err := pipelineStore.GetPipeline("1")
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)

	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
	})

	err = pipelineStore.UpdatePipeline(swf)
	assert.Nil(t, err)

	pipelineExpected = model.Pipeline{
		UUID:           "1",
		DisplayName:    "pp 1",
		Name:           "MY_NAME",
		Namespace:      "MY_NAMESPACE",
		PackageId:      1,
		Enabled:        false,
		Conditions:     "NO_STATUS:",
		CreatedAtInSec: 0,
		UpdatedAtInSec: 1,
		MaxConcurrency: 0,
		Parameters:     "[]",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: nil,
				CronScheduleEndTimeInSec:   nil,
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: nil,
				PeriodicScheduleEndTimeInSec:   nil,
				IntervalSecond:                 util.Int64Pointer(0),
			},
		},
	}

	pipeline, err = pipelineStore.GetPipeline("1")
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)
}

func TestUpdatePipeline_MissingField(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	// Name
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
	})

	err := pipelineStore.UpdatePipeline(swf)
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_NAME",
			UID:  "1",
		},
	})

	err = pipelineStore.UpdatePipeline(swf)
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
		},
	})

	err = pipelineStore.UpdatePipeline(swf)
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}

func TestUpdatePipeline_RecordNotFound(t *testing.T) {
	db := initializePipelineDB()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())

	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "UNKNOWN_UID",
		},
	})

	err := pipelineStore.UpdatePipeline(swf)
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "There is no pipeline")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}

func TestUpdatePipeline_InternalError(t *testing.T) {
	db := initializePipelineDB()
	//defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch())
	db.Close()

	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "UNKNOWN_UID",
		},
	})

	err := pipelineStore.UpdatePipeline(swf)
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "Internal Server Error")
	assert.Contains(t, err.(*util.UserError).Error(), "database is closed")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.Internal)
}
