// Copyright 2021 The Kubeflow Authors
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
	"reflect"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

const (
	defaultFakeTaskId      = "123e4567-e89b-12d3-a456-426655440010"
	defaultFakeTaskIdTwo   = "123e4567-e89b-12d3-a456-426655440011"
	defaultFakeTaskIdThree = "123e4567-e89b-12d3-a456-426655440012"
	defaultFakeTaskIdFour  = "123e4567-e89b-12d3-a456-426655440013"
	defaultFakeTaskIdFive  = "123e4567-e89b-12d3-a456-426655440014"
)

func initializeTaskStore() (*DB, *TaskStore) {
	db := NewFakeDBOrFatal()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "e1", Namespace: "ns1"})
	expStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpIdTwo, nil)
	expStore.CreateExperiment(&model.Experiment{Name: "e2", Namespace: "ns2"})
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())

	run1 := &model.Run{
		UUID:           defaultFakeRunId,
		ExperimentId:   defaultFakeExpId,
		DisplayName:    "run1",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          4,
			Conditions:              "done",
			State:                   model.RuntimeStateSucceeded,
			WorkflowRuntimeManifest: "workflow1",
		},
	}
	runStore.CreateRun(run1)

	run2 := &model.Run{
		UUID:           defaultFakeRunIdTwo,
		ExperimentId:   defaultFakeExpIdTwo,
		DisplayName:    "run2",
		K8SName:        "workflow-name",
		Namespace:      "ns2",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          4,
			Conditions:              "done",
			State:                   model.RuntimeStateSucceeded,
			WorkflowRuntimeManifest: "workflow2",
		},
	}

	run3 := &model.Run{
		UUID:           defaultFakeRunIdThree,
		ExperimentId:   defaultFakeExpId,
		DisplayName:    "run3",
		K8SName:        "workflow-name",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		StorageState:   model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          5,
			Conditions:              "Running",
			State:                   model.RuntimeStateRunning,
			WorkflowRuntimeManifest: "workflow1",
		},
	}

	// runStore.CreateRun(run1)
	runStore.CreateRun(run2)
	runStore.CreateRun(run3)

	taskStore := NewTaskStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskId, nil))
	task1 := &model.Task{
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/pipeline1",
		RunId:             run1.UUID,
		MLMDExecutionID:   "1",
		StartedTimestamp:  1,
		FinishedTimestamp: 2,
		Fingerprint:       "1",
	}
	taskStore.CreateTask(task1)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdTwo, nil)
	task2 := &model.Task{
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/pipeline1",
		RunId:             run1.UUID,
		MLMDExecutionID:   "2",
		StartedTimestamp:  3,
		FinishedTimestamp: 4,
		Fingerprint:       "2",
	}
	taskStore.CreateTask(task2)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdThree, nil)
	task3 := &model.Task{
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/pipeline1",
		RunId:             run3.UUID,
		MLMDExecutionID:   "3",
		StartedTimestamp:  5,
		FinishedTimestamp: 6,
		Fingerprint:       "1",
	}
	taskStore.CreateTask(task3)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdFour, nil)
	task4 := &model.Task{
		Namespace:         "ns2",
		PipelineName:      "namespace/ns2/pipeline/pipeline2",
		RunId:             run2.UUID,
		MLMDExecutionID:   "4",
		StartedTimestamp:  5,
		FinishedTimestamp: 6,
		Fingerprint:       "1",
	}
	taskStore.CreateTask(task4)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdFive, nil)
	task5 := &model.Task{
		Namespace:         "ns2",
		PipelineName:      "namespace/ns2/pipeline/pipeline2",
		RunId:             run2.UUID,
		MLMDExecutionID:   "5",
		StartedTimestamp:  7,
		FinishedTimestamp: 8,
		Fingerprint:       "10",
	}
	taskStore.CreateTask(task5)
	return db, taskStore
}

func TestListTasks(t *testing.T) {
	db, taskStore := initializeTaskStore()
	defer db.Close()

	expectedFirstPageTasks := []*model.Task{
		{
			UUID:              defaultFakeTaskIdFour,
			Namespace:         "ns2",
			PipelineName:      "namespace/ns2/pipeline/pipeline2",
			RunId:             defaultFakeRunIdTwo,
			MLMDExecutionID:   "4",
			StartedTimestamp:  5,
			FinishedTimestamp: 6,
			Fingerprint:       "1",
		},
	}
	expectedSecondPageTasks := []*model.Task{
		{
			UUID:              defaultFakeTaskIdFive,
			Namespace:         "ns2",
			PipelineName:      "namespace/ns2/pipeline/pipeline2",
			RunId:             defaultFakeRunIdTwo,
			MLMDExecutionID:   "5",
			StartedTimestamp:  7,
			FinishedTimestamp: 8,
			Fingerprint:       "10",
		},
	}

	opts, err := list.NewOptions(&model.Task{}, 1, "", nil)
	assert.Nil(t, err)

	tasks, total_size, nextPageToken, err := taskStore.ListTasks(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: defaultFakeRunIdTwo}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageTasks, tasks, "Unexpected Tasks listed")
	assert.NotEmpty(t, nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	tasks, total_size, nextPageToken, err = taskStore.ListTasks(
		&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: defaultFakeRunIdTwo}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageTasks, tasks, "Unexpected Tasks listed")
	assert.Empty(t, nextPageToken)
}

func TestTaskStore_GetTask(t *testing.T) {
	db, taskStore := initializeTaskStore()
	defer db.Close()

	task1 := &model.Task{
		UUID:              defaultFakeTaskIdFour,
		Namespace:         "ns2",
		PipelineName:      "namespace/ns2/pipeline/pipeline2",
		RunId:             defaultFakeRunIdTwo,
		MLMDExecutionID:   "4",
		StartedTimestamp:  5,
		FinishedTimestamp: 6,
		Fingerprint:       "1",
	}
	task2 := &model.Task{
		UUID:              defaultFakeTaskIdFive,
		Namespace:         "ns2",
		PipelineName:      "namespace/ns2/pipeline/pipeline2",
		RunId:             defaultFakeRunIdTwo,
		MLMDExecutionID:   "5",
		StartedTimestamp:  7,
		FinishedTimestamp: 8,
		Fingerprint:       "10",
	}

	tests := []struct {
		name    string
		id      string
		want    *model.Task
		wantErr bool
		errMsg  string
	}{
		{
			"valid -task 1",
			defaultFakeTaskIdFour,
			task1,
			false,
			"",
		},
		{
			"valid -task 2",
			defaultFakeTaskIdFive,
			task2,
			false,
			"",
		},
		{
			"not found",
			"This does not exist",
			nil,
			true,
			"not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := taskStore.GetTask(tt.id)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TaskStore.GetTask() = %v, want %v", got, tt.want)
			}
		})
	}
}
