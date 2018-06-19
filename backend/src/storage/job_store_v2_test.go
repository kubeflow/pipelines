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

func initializeDB() *gorm.DB {
	db := NewFakeDbOrFatal()
	job1 := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Condition:        "running",
		},
		Workflow: "workflow1",
	}
	job2 := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "2",
			Name:             "job2",
			Namespace:        "n2",
			PipelineID:       "1",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Condition:        "done",
		},
		Workflow: "workflow1",
	}
	job3 := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "3",
			Name:             "job3",
			Namespace:        "n3",
			PipelineID:       "2",
			CreatedAtInSec:   3,
			ScheduledAtInSec: 3,
			Condition:        "done",
		},
		Workflow: "workflow3",
	}
	db.Create(job1)
	db.Create(job2)
	db.Create(job3)
	return db
}

func TestListJobsV2_Pagination(t *testing.T) {
	db := initializeDB()
	defer db.Close()
	jobStore := NewJobStoreV2(db, util.NewFakeTimeForEpoch())

	expectedFirstPageJobs := []model.JobV2{
		{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Condition:        "running",
		}}
	expectedSecondPageJobs := []model.JobV2{
		{
			UUID:             "2",
			Name:             "job2",
			Namespace:        "n2",
			PipelineID:       "1",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Condition:        "done",
		}}
	jobs, nextPageToken, err := jobStore.ListJobs("1", "", 1, model.GetJobV2TablePrimaryKeyColumn())
	assert.Nil(t, err)
	assert.Equal(t, expectedFirstPageJobs, jobs, "Unexpected Job listed.")
	assert.NotEmpty(t, nextPageToken)

	jobs, nextPageToken, err = jobStore.ListJobs("1", nextPageToken, 1, model.GetJobV2TablePrimaryKeyColumn())
	assert.Nil(t, err)
	assert.Equal(t, expectedSecondPageJobs, jobs, "Unexpected Job listed.")
	assert.Empty(t, nextPageToken)
}

func TestListJobsV2_Pagination_LessThanPageSize(t *testing.T) {
	db := initializeDB()
	defer db.Close()
	jobStore := NewJobStoreV2(db, util.NewFakeTimeForEpoch())

	expectedJobs := []model.JobV2{
		{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Condition:        "running",
		},
		{
			UUID:             "2",
			Name:             "job2",
			Namespace:        "n2",
			PipelineID:       "1",
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Condition:        "done",
		}}
	jobs, nextPageToken, err := jobStore.ListJobs("1", "", 10, model.GetJobV2TablePrimaryKeyColumn())
	assert.Nil(t, err)
	assert.Equal(t, expectedJobs, jobs, "Unexpected Job listed.")
	assert.Empty(t, nextPageToken)
}

func TestListJobsV2Error(t *testing.T) {
	db := initializeDB()
	defer db.Close()
	jobStore := NewJobStoreV2(db, util.NewFakeTimeForEpoch())
	db.Close()
	_, _, err := jobStore.ListJobs("1", "", 10, model.GetJobV2TablePrimaryKeyColumn())
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to throw an internal error")
}

func TestGetJobV2(t *testing.T) {
	db := initializeDB()
	defer db.Close()
	jobStore := NewJobStoreV2(db, util.NewFakeTimeForEpoch())

	expectedJob := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Condition:        "running",
		},
		Workflow: "workflow1",
	}

	jobDetail, err := jobStore.GetJob("1", "1")
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, jobDetail)
}

func TestGetJobV2_NotFoundError(t *testing.T) {
	db := initializeDB()
	defer db.Close()
	jobStore := NewJobStoreV2(db, util.NewFakeTimeForEpoch())

	_, err := jobStore.GetJob("1", "notfound")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected not to find the job")
}

func TestGetJobV2_InternalError(t *testing.T) {
	db := initializeDB()
	defer db.Close()
	jobStore := NewJobStoreV2(db, util.NewFakeTimeForEpoch())
	db.Close()

	_, err := jobStore.GetJob("1", "1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get job to return internal error")
}
