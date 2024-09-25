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
	"fmt"
	"testing"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	fakeID      = "123e4567-e89b-12d3-a456-426655440000"
	fakeIDTwo   = "123e4567-e89b-12d3-a456-426655440001"
	fakeIDThree = "123e4567-e89b-12d3-a456-426655440002"
	fakeIDFour  = "123e4567-e89b-12d3-a456-426655440003"
)

func createExperiment(name string) *model.Experiment {
	return createExperimentInNamespace(name, "")
}

func createExperimentInNamespace(name string, namespace string) *model.Experiment {
	return &model.Experiment{
		Name:        name,
		Description: fmt.Sprintf("My name is %s", name),
		Namespace:   namespace,
	}
}

func TestListExperiments_Pagination(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil)
	experimentStore.CreateExperiment(createExperiment("experiment3"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDThree, nil)
	experimentStore.CreateExperiment(createExperiment("experiment4"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDFour, nil)
	experimentStore.CreateExperiment(createExperiment("experiment2"))
	expectedExperiment1 := &model.Experiment{
		UUID:                  fakeID,
		CreatedAtInSec:        1,
		LastRunCreatedAtInSec: 0,
		Name:                  "experiment1",
		Description:           "My name is experiment1",
		StorageState:          "AVAILABLE",
	}
	expectedExperiment4 := &model.Experiment{
		UUID:                  fakeIDFour,
		CreatedAtInSec:        4,
		LastRunCreatedAtInSec: 0,
		Name:                  "experiment2",
		Description:           "My name is experiment2",
		StorageState:          "AVAILABLE",
	}
	experimentsExpected := []*model.Experiment{expectedExperiment1, expectedExperiment4}
	opts, err := list.NewOptions(&model.Experiment{}, 2, "name", nil)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(&model.FilterContext{}, opts)

	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, experimentsExpected, experiments)
	assert.Equal(t, 4, total_size)

	expectedExperiment2 := &model.Experiment{
		UUID:                  fakeIDTwo,
		CreatedAtInSec:        2,
		LastRunCreatedAtInSec: 0,
		Name:                  "experiment3",
		Description:           "My name is experiment3",
		StorageState:          "AVAILABLE",
	}
	expectedExperiment3 := &model.Experiment{
		UUID:                  fakeIDThree,
		CreatedAtInSec:        3,
		LastRunCreatedAtInSec: 0,
		Name:                  "experiment4",
		Description:           "My name is experiment4",
		StorageState:          "AVAILABLE",
	}
	experimentsExpected2 := []*model.Experiment{expectedExperiment2, expectedExperiment3}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err = experimentStore.ListExperiments(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, experimentsExpected2, experiments)
}

func TestListExperiments_Pagination_Descend(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil)
	experimentStore.CreateExperiment(createExperiment("experiment3"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDThree, nil)
	experimentStore.CreateExperiment(createExperiment("experiment4"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDFour, nil)
	experimentStore.CreateExperiment(createExperiment("experiment2"))

	expectedExperiment2 := &model.Experiment{
		UUID:           fakeIDTwo,
		CreatedAtInSec: 2,
		Name:           "experiment3",
		Description:    "My name is experiment3",
		StorageState:   "AVAILABLE",
	}
	expectedExperiment3 := &model.Experiment{
		UUID:           fakeIDThree,
		CreatedAtInSec: 3,
		Name:           "experiment4",
		Description:    "My name is experiment4",
		StorageState:   "AVAILABLE",
	}
	experimentsExpected := []*model.Experiment{expectedExperiment3, expectedExperiment2}

	opts, err := list.NewOptions(&model.Experiment{}, 2, "name desc", nil)
	assert.Nil(t, err)
	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(&model.FilterContext{}, opts)

	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, experimentsExpected, experiments)

	expectedExperiment1 := &model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		StorageState:   "AVAILABLE",
	}
	expectedExperiment4 := &model.Experiment{
		UUID:           fakeIDFour,
		CreatedAtInSec: 4,
		Name:           "experiment2",
		Description:    "My name is experiment2",
		StorageState:   "AVAILABLE",
	}
	experimentsExpected2 := []*model.Experiment{expectedExperiment4, expectedExperiment1}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err = experimentStore.ListExperiments(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, experimentsExpected2, experiments)
}

func TestListExperiments_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	expectedExperiment1 := &model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		StorageState:   "AVAILABLE",
	}
	experimentsExpected := []*model.Experiment{expectedExperiment1}

	opts, err := list.NewOptions(&model.Experiment{}, 2, "", nil)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(&model.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, experimentsExpected, experiments)
}

func TestListExperimentsError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	db.Close()

	opts, err := list.NewOptions(&model.Experiment{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = experimentStore.ListExperiments(&model.FilterContext{}, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetExperiment(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentExpected := model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		StorageState:   "AVAILABLE",
	}

	experiment, err := experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment")
}

func TestGetExperiment_NotFoundError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))

	_, err := experimentStore.GetExperiment(fakeID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get experiment to return not found")
}

func TestGetExperiment_InternalError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	db.Close()
	_, err := experimentStore.GetExperiment("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get experiment to return internal error")
}

func TestCreateExperiment(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentExpected := model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		StorageState:   "AVAILABLE",
	}

	experiment := createExperiment("experiment1")
	experiment, err := experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment")
}

func TestCreateExperiment_DifferentNamespaces(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentExpected := model.Experiment{
		UUID:           fakeID,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		Namespace:      "namespace1",
		StorageState:   "AVAILABLE",
	}

	experiment := createExperimentInNamespace("experiment1", "namespace1")
	experiment, err := experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment")

	experimentStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil))
	experiment = createExperimentInNamespace("experiment1", "namespace2")
	experimentExpected = model.Experiment{
		UUID:           fakeIDTwo,
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		Namespace:      "namespace2",
		StorageState:   "AVAILABLE",
	}

	experiment, err = experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)
	assert.Equal(t, experimentExpected, *experiment, "Got unexpected experiment")
}

func TestCreateExperiment_DuplicatedKey(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experiment := createExperiment("experiment1")
	_, err := experimentStore.CreateExperiment(experiment)
	assert.Nil(t, err)

	experimentStore = NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil))
	_, err = experimentStore.CreateExperiment(experiment)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name experiment1 already exist")
}

func TestCreateExperiment_InternalServerError(t *testing.T) {
	experiment := &model.Experiment{Name: "Experiment123"}
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	db.Close()

	_, err := experimentStore.CreateExperiment(experiment)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create experiment to return error")
}

func TestCreateExperiment_CreateUUIDFailure(t *testing.T) {
	experiment := &model.Experiment{Name: "Experiment123"}
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, errors.New("error")))
	db.Close()

	_, err := experimentStore.CreateExperiment(experiment)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Failed to create an experiment id")
}

func TestDeleteExperiment(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experiment, err := experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.NotNil(t, experiment)

	err = experimentStore.DeleteExperiment(fakeID)
	assert.Nil(t, err)
	_, err = experimentStore.GetExperiment(fakeID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_InternalError(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experiment, err := experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.NotNil(t, experiment)

	db.Close()
	err = experimentStore.DeleteExperiment(fakeID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete experiment to return internal error")
}

func TestListExperiments_Filtering(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDTwo, nil)
	experimentStore.CreateExperiment(createExperiment("experiment2"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDThree, nil)
	experimentStore.CreateExperiment(createExperiment("experiment3"))
	experimentStore.uuid = util.NewFakeUUIDGeneratorOrFatal(fakeIDFour, nil)
	experimentStore.CreateExperiment(createExperiment("experiment4"))

	filterProto := &apiv1beta1.Filter{
		Predicates: []*apiv1beta1.Predicate{
			{
				Key: "name",
				Op:  apiv1beta1.Predicate_IN,
				Value: &apiv1beta1.Predicate_StringValues{
					StringValues: &apiv1beta1.StringValues{
						Values: []string{"experiment2", "experiment4", "experiment3"},
					},
				},
			},
		},
	}
	newFilter, _ := filter.New(filterProto)

	opts, err := list.NewOptions(&model.Experiment{}, 2, "id", newFilter)
	assert.Nil(t, err)
	experiments, total_size, nextPageToken, err := experimentStore.ListExperiments(&model.FilterContext{}, opts)

	expected := []*model.Experiment{
		{
			UUID:           fakeIDTwo,
			CreatedAtInSec: 2,
			Name:           "experiment2",
			Description:    "My name is experiment2",
			StorageState:   "AVAILABLE",
		},
		{
			UUID:           fakeIDThree,
			CreatedAtInSec: 3,
			Name:           "experiment3",
			Description:    "My name is experiment3",
			StorageState:   "AVAILABLE",
		},
	}

	assert.Nil(t, err)
	assert.NotEqual(t, "", nextPageToken)
	assert.Equal(t, expected, experiments)
	assert.Equal(t, 3, total_size)

	// Next page should give experiment4.
	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	experiments, total_size, nextPageToken, err = experimentStore.ListExperiments(&model.FilterContext{}, opts)

	expected = []*model.Experiment{
		{
			UUID:           fakeIDFour,
			CreatedAtInSec: 4,
			Name:           "experiment4",
			Description:    "My name is experiment4",
			StorageState:   "AVAILABLE",
		},
	}

	assert.Nil(t, err)
	// No more pages.
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, expected, experiments)
	assert.Equal(t, 3, total_size)
}

func TestArchiveExperiment_InternalError(t *testing.T) {
	db := NewFakeDBOrFatal()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	experimentStore.CreateExperiment(createExperiment("experiment1"))
	db.Close()

	err := experimentStore.ArchiveExperiment(fakeID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected archive experiment to return internal error")
}

func TestArchiveAndUnarchiveExperiment(t *testing.T) {
	db := NewFakeDBOrFatal()
	defer db.Close()

	// Initial state: 1 experiment and 2 runs in it.
	// The experiment is unarchived.
	// One run is archived and the other is not.
	experimentStore := NewExperimentStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(fakeID, nil),
	)
	experimentStore.CreateExperiment(
		createExperiment("experiment1"),
	)
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())
	run1 := &model.Run{
		UUID:         "1",
		DisplayName:  "run1",
		K8SName:      "run1",
		StorageState: model.StorageStateAvailableV1,
		Namespace:    "n1",
		ExperimentId: fakeID,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			State:                   "RUNNING",
			WorkflowRuntimeManifest: "workflow1",
		},
		PipelineSpec: model.PipelineSpec{},
	}
	run2 := &model.Run{
		UUID:         "2",
		DisplayName:  "run2",
		K8SName:      "run2",
		StorageState: model.StorageStateArchivedV1,
		Namespace:    "n1",
		ExperimentId: fakeID,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          2,
			ScheduledAtInSec:        2,
			State:                   "FINISHED",
			WorkflowRuntimeManifest: "workflow1",
		},
		PipelineSpec: model.PipelineSpec{},
	}
	runStore.CreateRun(run1)
	runStore.CreateRun(run2)
	opts, err := list.NewOptions(&model.Run{}, 10, "id", nil)
	runs, total_run_size, _, err := runStore.ListRuns(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: fakeID}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_run_size)
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_AVAILABLE.String(), string(runs[0].StorageState.ToV1()))
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED.String(), string(runs[1].StorageState.ToV1()))
	assert.Equal(t, apiv2beta1.Run_AVAILABLE.String(), runs[0].StorageState.ToString())
	assert.Equal(t, apiv2beta1.Run_ARCHIVED.String(), runs[1].StorageState.ToString())

	jobStore := NewJobStore(db, util.NewFakeTimeForEpoch())
	job1 := &model.Job{
		UUID:        "1",
		DisplayName: "pp 1",
		K8SName:     "pp1",
		Namespace:   "n1",
		Enabled:     true,
		Conditions:  "ENABLED",
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				PeriodicScheduleEndTimeInSec:   util.Int64Pointer(2),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ExperimentId:   fakeID,
	}
	job2 := &model.Job{
		UUID:        "2",
		DisplayName: "pp 2",
		K8SName:     "pp2",
		Namespace:   "n1",
		Conditions:  "ready",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				CronScheduleEndTimeInSec:   util.Int64Pointer(2),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		NoCatchup:      true,
		Enabled:        false,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		ExperimentId:   fakeID,
	}
	jobStore.CreateJob(job1)
	jobStore.CreateJob(job2)

	// Archive experiment and verify the experiment and two runs in it are all archived.
	err = experimentStore.ArchiveExperiment(fakeID)
	assert.Nil(t, err)
	exp, err := experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.Equal(t, "ARCHIVED", exp.StorageState.ToString())
	opts, err = list.NewOptions(&model.Run{}, 10, "id", nil)
	runs, total_run_size, _, err = runStore.ListRuns(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: fakeID}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_run_size)
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED.String(), string(runs[0].StorageState.ToV1()))
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED.String(), string(runs[1].StorageState.ToV1()))
	assert.Equal(t, apiv2beta1.Run_ARCHIVED.String(), runs[0].StorageState.ToString())
	assert.Equal(t, apiv2beta1.Run_ARCHIVED.String(), runs[1].StorageState.ToString())
	jobs, total_job_size, _, err := jobStore.ListJobs(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: fakeID}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_job_size)
	assert.Equal(t, false, jobs[0].Enabled)
	assert.Equal(t, false, jobs[1].Enabled)

	// Unarchive the experiment, and verify the experiment is unarchived while two runs in it stay archived.
	err = experimentStore.UnarchiveExperiment(fakeID)
	assert.Nil(t, err)
	exp, err = experimentStore.GetExperiment(fakeID)
	assert.Nil(t, err)
	assert.Equal(t, "AVAILABLE", exp.StorageState.ToString())
	runs, total_run_size, _, err = runStore.ListRuns(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: fakeID}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, total_run_size, 2)
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED.String(), string(runs[0].StorageState.ToV1()))
	assert.Equal(t, apiv1beta1.Run_STORAGESTATE_ARCHIVED.String(), string(runs[1].StorageState.ToV1()))
	assert.Equal(t, apiv2beta1.Run_ARCHIVED.String(), runs[0].StorageState.ToString())
	assert.Equal(t, apiv2beta1.Run_ARCHIVED.String(), runs[1].StorageState.ToString())
	jobs, total_job_size, _, err = jobStore.ListJobs(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: fakeID}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, total_job_size, 2)
	assert.Equal(t, false, jobs[0].Enabled)
	assert.Equal(t, false, jobs[1].Enabled)
}

func TestUnarchiveExperiment_InternalError(t *testing.T) {
	db := NewFakeDBOrFatal()
	experimentStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(fakeID, nil))
	db.Close()

	err := experimentStore.UnarchiveExperiment(fakeID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected unarchive experiment to return internal error")
}
