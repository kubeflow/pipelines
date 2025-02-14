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
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultFakeExpId    = "123e4567-e89b-12d3-a456-426655440000"
	defaultFakeExpIdTwo = "123e4567-e89b-12d3-a456-426655440001"
)

func initializeDbAndStore() (*DB, *JobStore) {
	db := NewFakeDBOrFatal()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "exp1", Namespace: "n1"})
	expStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpIdTwo, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "exp2", Namespace: "n1"})
	expStore.CreateExperiment(&model.Experiment{Name: "exp2", Namespace: "n1"})
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpIdTwo, nil))
	pipeline, _ := pipelineStore.CreatePipeline(&model.Pipeline{Name: "p1"})
	jobStore := NewJobStore(db, util.NewFakeTimeForEpoch())
	job1 := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		Enabled:     true,
		Conditions:  "ready",
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		PipelineSpec: model.PipelineSpec{
			PipelineId:   pipeline.UUID,
			PipelineName: "p1",
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   defaultFakeExpId,
	}
	jobStore.CreateJob(job1.ToV1())
	job2 := &model.Job{
		UUID:        "2",
		DisplayName: "pp 2",
		K8SName:     "pp2",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   pipeline.UUID,
			PipelineName: "p1",
		},
		Conditions: "ready",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				CronScheduleEndTimeInSec:   util.Int64Pointer(2),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		NoCatchup:      true,
		Enabled:        true,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		ExperimentId:   defaultFakeExpIdTwo,
	}
	jobStore.CreateJob(job2.ToV1())
	return db, jobStore
}

func TestListJobs_Pagination(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	jobsExpected := []*model.Job{
		{
			UUID:        "1",
			DisplayName: "pp 1",
			K8SName:     "pp1",
			Namespace:   "n1",
			PipelineSpec: model.PipelineSpec{
				PipelineId:   DefaultFakePipelineIdTwo,
				PipelineName: "p1",
			},
			Conditions: "ready",
			Enabled:    true,
			Trigger: model.Trigger{
				PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
					PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
					IntervalSecond:                 util.Int64Pointer(3),
				},
			},
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
			ExperimentId:   defaultFakeExpId,
		},
	}
	jobsExpected[0] = jobsExpected[0].ToV1()
	opts, err := list.NewOptions(&model.Job{}, 1, "name", nil)
	assert.Nil(t, err)
	jobs, total_size, nextPageToken, err := jobStore.ListJobs(&model.FilterContext{}, opts)
	jobs[0] = jobs[0].ToV1()

	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, jobsExpected, jobs)

	jobsExpected2 := []*model.Job{
		{
			UUID:        "2",
			DisplayName: "pp 2",
			K8SName:     "pp2",
			Namespace:   "n1",
			PipelineSpec: model.PipelineSpec{
				PipelineId:   DefaultFakePipelineIdTwo,
				PipelineName: "p1",
			},
			Enabled: true,
			Trigger: model.Trigger{
				CronSchedule: model.CronSchedule{
					CronScheduleStartTimeInSec: util.Int64Pointer(1),
					CronScheduleEndTimeInSec:   util.Int64Pointer(2),
					Cron:                       util.StringPointer("1 * *"),
				},
			},
			NoCatchup:      true,
			CreatedAtInSec: 2,
			UpdatedAtInSec: 2,
			Conditions:     "ready",
			ExperimentId:   defaultFakeExpIdTwo,
		},
	}
	jobsExpected2[0] = jobsExpected2[0].ToV1()

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	jobs, total_size, newToken, err := jobStore.ListJobs(&model.FilterContext{}, opts)
	jobs[0] = jobs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, "", newToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, jobsExpected2, jobs)
}

func TestListJobs_TotalSizeWithNoFilter(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	opts, _ := list.NewOptions(&model.Job{}, 4, "name", nil)

	// No filter
	jobs, total_size, _, err := jobStore.ListJobs(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))
	assert.Equal(t, 2, total_size)
}

func TestListJobs_TotalSizeWithFilter(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	// Add a filter
	protoFilter := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key: "name",
				Op:  api.Predicate_IN,
				Value: &api.Predicate_StringValues{
					StringValues: &api.StringValues{
						Values: []string{"pp 1"},
					},
				},
			},
		},
	}
	newFilter, _ := filter.New(protoFilter)
	opts, _ := list.NewOptions(&model.Job{}, 4, "name", newFilter)
	jobs, total_size, _, err := jobStore.ListJobs(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 1, total_size)
}

func TestListJobs_Pagination_Descent(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	jobsExpected := []*model.Job{
		{
			UUID:        "2",
			DisplayName: "pp 2",
			K8SName:     "pp2",
			Namespace:   "n1",
			PipelineSpec: model.PipelineSpec{
				PipelineId:   DefaultFakePipelineIdTwo,
				PipelineName: "p1",
			},
			Enabled:    true,
			Conditions: "ready",
			Trigger: model.Trigger{
				CronSchedule: model.CronSchedule{
					CronScheduleStartTimeInSec: util.Int64Pointer(1),
					CronScheduleEndTimeInSec:   util.Int64Pointer(2),
					Cron:                       util.StringPointer("1 * *"),
				},
			},
			NoCatchup:      true,
			CreatedAtInSec: 2,
			UpdatedAtInSec: 2,
			ExperimentId:   defaultFakeExpIdTwo,
		},
	}
	jobsExpected[0] = jobsExpected[0].ToV1()
	opts, err := list.NewOptions(&model.Job{}, 1, "name desc", nil)
	assert.Nil(t, err)
	jobs, total_size, nextPageToken, err := jobStore.ListJobs(&model.FilterContext{}, opts)
	jobs[0] = jobs[0].ToV1()
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, jobsExpected, jobs)

	jobsExpected2 := []*model.Job{
		{
			UUID:        "1",
			DisplayName: "pp 1",
			K8SName:     "pp1",
			Namespace:   "n1",
			PipelineSpec: model.PipelineSpec{
				PipelineId:   DefaultFakePipelineIdTwo,
				PipelineName: "p1",
			},
			Enabled:    true,
			Conditions: "ready",
			Trigger: model.Trigger{
				PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
					PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
					IntervalSecond:                 util.Int64Pointer(3),
				},
			},
			NoCatchup:      false,
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
			ExperimentId:   defaultFakeExpId,
		},
	}
	jobsExpected2[0] = jobsExpected2[0].ToV1()
	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	jobs, total_size, newToken, err := jobStore.ListJobs(&model.FilterContext{}, opts)
	jobs[0] = jobs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, "", newToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, jobsExpected2, jobs)
}

func TestListJobs_Pagination_LessThanPageSize(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	jobsExpected := []*model.Job{
		{
			UUID:        "1",
			DisplayName: "pp 1",
			K8SName:     "pp1",
			Namespace:   "n1",
			PipelineSpec: model.PipelineSpec{
				PipelineId:   DefaultFakePipelineIdTwo,
				PipelineName: "p1",
			},
			Enabled:    true,
			Conditions: "ready",
			Trigger: model.Trigger{
				PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
					PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
					IntervalSecond:                 util.Int64Pointer(3),
				},
			},
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
			ExperimentId:   defaultFakeExpId,
		},
		{
			UUID:        "2",
			DisplayName: "pp 2",
			K8SName:     "pp2",
			Namespace:   "n1",
			PipelineSpec: model.PipelineSpec{
				PipelineId:   DefaultFakePipelineIdTwo,
				PipelineName: "p1",
			},
			Enabled:    true,
			Conditions: "ready",
			Trigger: model.Trigger{
				CronSchedule: model.CronSchedule{
					CronScheduleStartTimeInSec: util.Int64Pointer(1),
					CronScheduleEndTimeInSec:   util.Int64Pointer(2),
					Cron:                       util.StringPointer("1 * *"),
				},
			},
			NoCatchup:      true,
			CreatedAtInSec: 2,
			UpdatedAtInSec: 2,
			ExperimentId:   defaultFakeExpIdTwo,
		},
	}
	jobsExpected[0] = jobsExpected[0].ToV1()
	jobsExpected[1] = jobsExpected[1].ToV1()
	opts, err := list.NewOptions(&model.Job{}, 2, "name", nil)
	assert.Nil(t, err)
	jobs, total_size, nextPageToken, err := jobStore.ListJobs(&model.FilterContext{}, opts)
	jobs[0] = jobs[0].ToV1()
	jobs[1] = jobs[1].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, jobsExpected, jobs)
}

func TestListJobs_FilterByReferenceKey(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	jobsExpected := []*model.Job{
		{
			UUID:        "1",
			DisplayName: "pp 1",
			K8SName:     "pp1",
			Namespace:   "n1",
			PipelineSpec: model.PipelineSpec{
				PipelineId:   DefaultFakePipelineIdTwo,
				PipelineName: "p1",
			},
			Enabled:    true,
			Conditions: "ready",
			Trigger: model.Trigger{
				PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
					PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
					IntervalSecond:                 util.Int64Pointer(3),
				},
			},
			CreatedAtInSec: 1,
			UpdatedAtInSec: 1,
			ExperimentId:   defaultFakeExpId,
		},
	}
	jobsExpected[0] = jobsExpected[0].ToV1()
	opts, err := list.NewOptions(&model.Job{}, 2, "name", nil)
	assert.Nil(t, err)
	jobs, total_size, nextPageToken, err := jobStore.ListJobs(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: defaultFakeExpId}}, opts)
	jobs[0] = jobs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, jobsExpected, jobs)

	jobs, total_size, nextPageToken, err = jobStore.ListJobs(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: "n1"}}, opts)
	jobs[0] = jobs[0].ToV1()
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size) // both test jobs belong to namespace `n1`
}

func TestListJobsError(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	db.Close()
	opts, err := list.NewOptions(&model.Job{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = jobStore.ListJobs(
		&model.FilterContext{}, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected to list job to return error")
}

func TestGetJob(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	jobExpected := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
		},
		Conditions: "ready",
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   defaultFakeExpId,
	}
	jobExpected = jobExpected.ToV1()
	job, err := jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected, job.ToV1(), "Got unexpected job")
}

func TestGetJob_NotFoundError(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	_, err := jobStore.GetJob("notexist")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get job to return not found error")
}

func TestGetJob_InternalError(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	db.Close()
	_, err := jobStore.GetJob("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get job to return internal error")
}

func TestCreateJob(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	experiment, _ := expStore.CreateExperiment(&model.Experiment{Name: "exp1"})
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	pipeline, _ := pipelineStore.CreatePipeline(&model.Pipeline{Name: "p1"})
	jobStore := NewJobStore(db, util.NewFakeTimeForEpoch())
	job := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			Parameters:   `[{"name":"string1","value":"one"},{"name":"number2","value":"2"}]`,
			PipelineId:   pipeline.UUID,
			PipelineName: "p1",
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		Enabled:        true,
		ExperimentId:   experiment.UUID,
	}

	job, err := jobStore.CreateJob(job.ToV1())
	assert.Nil(t, err)
	jobExpected := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			Parameters:   `[{"name":"string1","value":"one"},{"name":"number2","value":"2"}]`,
			PipelineId:   pipeline.UUID,
			PipelineName: "p1",
		},
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   experiment.UUID,
	}
	jobExpected = jobExpected.ToV1()
	assert.Equal(t, jobExpected, job.ToV1(), "Got unexpected jobs")

	newJob, err := jobStore.GetJob(job.UUID)
	assert.Nil(t, err)
	assert.Equal(t, jobExpected, newJob.ToV1(), "Got unexpected jobs")

	// Check resource reference exists
	resourceReferenceStore := NewResourceReferenceStore(db)
	r, err := resourceReferenceStore.GetResourceReference("1", model.JobResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)
}

func TestCreateJob_V2(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "exp1"})
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	pipeline, _ := pipelineStore.CreatePipeline(&model.Pipeline{Name: "p1"})
	jobStore := NewJobStore(db, util.NewFakeTimeForEpoch())
	job := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   pipeline.UUID,
			PipelineName: "p1",
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   `[{"name":"param1","value":"world1"}]`,
				PipelineRoot: "gs://my-bucket/path/to/root/run1",
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		Enabled:        true,
		ExperimentId:   defaultFakeExpId,
	}

	job, err := jobStore.CreateJob(job.ToV1())
	assert.Nil(t, err)
	jobExpected := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   pipeline.UUID,
			PipelineName: "p1",
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   `[{"name":"param1","value":"world1"}]`,
				PipelineRoot: "gs://my-bucket/path/to/root/run1",
			},
		},
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   defaultFakeExpId,
	}
	jobExpected = jobExpected.ToV1()
	assert.Equal(t, jobExpected, job.ToV1(), "Got unexpected jobs")

	// Check resource reference exists
	resourceReferenceStore := NewResourceReferenceStore(db)
	r, err := resourceReferenceStore.GetResourceReference("1", model.JobResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)
}

func TestCreateJobError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	jobStore := NewJobStore(db, util.NewFakeTimeForEpoch())
	db.Close()
	job := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
		},
		Enabled:      true,
		ExperimentId: defaultFakeExpId,
	}

	job, err := jobStore.CreateJob(job.ToV2())
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create job to return error")
}

func TestEnableJob(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	err := jobStore.ChangeJobMode("1", false)
	assert.Nil(t, err)

	jobExpected := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
		},
		Conditions: "ENABLED",
		Enabled:    false,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 3,
		ExperimentId:   defaultFakeExpId,
	}

	job, err := jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected.ToV1(), job.ToV1(), "Got unexpected job")

	err = jobStore.ChangeJobMode("1", true)
	assert.Nil(t, err)

	jobExpected2 := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
		},
		Conditions: "ENABLED",
		Enabled:    true,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 4,
		ExperimentId:   defaultFakeExpId,
	}

	job, err = jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected2.ToV1(), job.ToV1(), "Got unexpected job")
}

func TestEnableJob_SkipUpdate(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	err := jobStore.ChangeJobMode("1", true)
	assert.Nil(t, err)

	jobExpected := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
		},
		Conditions: "ready",
		Enabled:    true,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   defaultFakeExpId,
	}

	job, err := jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected.ToV1(), job.ToV1(), "Got unexpected job")
}

func TestEnableJob_DatabaseError(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	db.Close()

	// Enabling the job.
	err := jobStore.ChangeJobMode("1", true)
	assert.Contains(t, err.Error(), "Error when enabling job 1 to true: sql: database is closed")
}

func TestUpdateJob_Success(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	jobExpected := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
		},
		Conditions: "ready",
		Enabled:    true,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   defaultFakeExpId,
	}

	job, err := jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected.ToV1(), job.ToV1())

	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v1beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "n1",
			UID:       "1",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        false,
			MaxConcurrency: util.Int64Pointer(200),
			NoCatchup:      util.BoolPointer(true),
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
			Conditions: []swfapi.ScheduledWorkflowCondition{
				{
					Type:               swfapi.ScheduledWorkflowEnabled,
					Status:             core.ConditionTrue,
					LastProbeTime:      metav1.NewTime(time.Unix(10, 0).UTC()),
					LastTransitionTime: metav1.NewTime(time.Unix(20, 0).UTC()),
					Reason:             string(swfapi.ScheduledWorkflowEnabled),
					Message:            "The schedule is enabled",
				},
			},
		},
	})

	err = jobStore.UpdateJob(swf)
	assert.Nil(t, err)

	jobExpected = &model.Job{
		UUID:           "1",
		DisplayName:    "pp 1",
		K8SName:        "MY_NAME",
		Namespace:      "n1",
		Enabled:        false,
		Conditions:     "ENABLED",
		CreatedAtInSec: 1,
		UpdatedAtInSec: 3,
		MaxConcurrency: 200,
		NoCatchup:      true,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
			Parameters:   "[{\"name\":\"PARAM1\",\"value\":\"NEW_VALUE1\"}]",
		},
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(10),
				CronScheduleEndTimeInSec:   util.Int64Pointer(20),
				Cron:                       util.StringPointer("MY_CRON"),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(30),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(40),
				IntervalSecond:                 util.Int64Pointer(50),
			},
		},
		ExperimentId: defaultFakeExpId,
	}
	job, err = jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected.ToV1(), job.ToV1())
}

func TestUpdateJob_MostlyEmptySpec(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	jobExpected := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
		},
		Conditions: "ready",
		Enabled:    true,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   defaultFakeExpId,
	}

	job, err := jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected.ToV1(), job.ToV1())

	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "n1",
			UID:       "1",
		},
	})

	err = jobStore.UpdateJob(swf)
	assert.Nil(t, err)

	jobExpected = &model.Job{
		UUID:           "1",
		DisplayName:    "pp 1",
		K8SName:        "MY_NAME",
		Namespace:      "n1",
		Enabled:        false,
		Conditions:     "STATUS_UNSPECIFIED",
		CreatedAtInSec: 1,
		UpdatedAtInSec: 3,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   DefaultFakePipelineIdTwo,
			PipelineName: "p1",
		},
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: nil,
				CronScheduleEndTimeInSec:   nil,
				Cron:                       util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: nil,
				PeriodicScheduleEndTimeInSec:   nil,
				IntervalSecond:                 util.Int64Pointer(0),
			},
		},
		ExperimentId: defaultFakeExpId,
	}

	job, err = jobStore.GetJob("1")
	assert.Nil(t, err)
	assert.Equal(t, jobExpected.ToV1(), job.ToV1())
}

func TestUpdateJob_RecordNotFound(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "UNKNOWN_UID",
		},
	})

	err := jobStore.UpdateJob(swf)
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "There is no job")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}

func TestUpdateJob_InternalError(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	db.Close()
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "UNKNOWN_UID",
		},
	})

	err := jobStore.UpdateJob(swf)
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "Internal Server Error")
	assert.Contains(t, err.(*util.UserError).Error(), "database is closed")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.Internal)
}

func TestDeleteJob(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()
	resourceReferenceStore := NewResourceReferenceStore(db)
	// Check resource reference exists
	r, err := resourceReferenceStore.GetResourceReference("1", model.JobResourceType, model.ExperimentResourceType)
	assert.Nil(t, err)
	assert.Equal(t, r.ReferenceUUID, defaultFakeExpId)

	// Delete job
	err = jobStore.DeleteJob("1")
	assert.Nil(t, err)
	_, err = jobStore.GetJob("1")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Job 1 not found")

	// Check resource reference deleted
	_, err = resourceReferenceStore.GetResourceReference("1", model.JobResourceType, model.ExperimentResourceType)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteJob_InternalError(t *testing.T) {
	db, jobStore := initializeDbAndStore()
	defer db.Close()

	db.Close()

	err := jobStore.DeleteJob("1")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete job to return internal error")
}

func TestJobAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.Job{}).APIToModelFieldMap() {
		assert.Contains(t, jobColumns, modelField)
	}
}
