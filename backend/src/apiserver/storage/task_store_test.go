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
	"fmt"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
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
	db := NewFakeDbOrFatal()
	expStore := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil))
	expStore.CreateExperiment(&model.Experiment{Name: "e1", Namespace: "ns1"})
	expStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpIdTwo, nil)
	expStore.CreateExperiment(&model.Experiment{Name: "e2", Namespace: "ns2"})
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch())

	run1 := &model.RunDetail{
		Run: model.Run{
			UUID:           defaultFakeRunId,
			ExperimentUUID: defaultFakeExpId,
			DisplayName:    "run1",
			Name:           "workflow-name",
			Namespace:      "ns1",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 4,
			Conditions:     "done",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  defaultFakeRunId,
					ResourceType:  common.Run,
					ReferenceUUID: defaultFakeExpId,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}
	runStore.CreateRun(run1)
	run2 := &model.RunDetail{
		Run: model.Run{
			UUID:           defaultFakeRunIdTwo,
			ExperimentUUID: defaultFakeExpIdTwo,
			DisplayName:    "run2",
			Name:           "workflow-name",
			Namespace:      "ns2",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 4,
			Conditions:     "Running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  defaultFakeRunIdTwo,
					ResourceType:  common.Run,
					ReferenceUUID: defaultFakeExpIdTwo,
					ReferenceName: "e2",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow2",
		},
	}

	run3 := &model.RunDetail{
		Run: model.Run{
			UUID:           defaultFakeRunIdThree,
			ExperimentUUID: defaultFakeExpId,
			DisplayName:    "run3",
			Name:           "workflow-name",
			Namespace:      "ns1",
			ServiceAccount: "pipeline-runner",
			StorageState:   api.Run_STORAGESTATE_AVAILABLE.String(),
			CreatedAtInSec: 5,
			Conditions:     "Running",
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  defaultFakeRunIdThree,
					ResourceType:  common.Run,
					ReferenceUUID: defaultFakeExpId,
					ReferenceName: "e1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner,
				},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: "workflow1",
		},
	}

	//runStore.CreateRun(run1)
	runStore.CreateRun(run2)
	runStore.CreateRun(run3)

	taskStore := NewTaskStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskId, nil))
	task1 := &model.Task{
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/pipeline1",
		RunUUID:           run1.UUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1,
		FinishedTimestamp: 2,
		Fingerprint:       "1",
	}
	taskStore.CreateTask(task1)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdTwo, nil)
	task2 := &model.Task{
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/pipeline1",
		RunUUID:           run1.UUID,
		MLMDExecutionID:   "2",
		CreatedTimestamp:  3,
		FinishedTimestamp: 4,
		Fingerprint:       "2",
	}
	taskStore.CreateTask(task2)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdThree, nil)
	task3 := &model.Task{
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/pipeline1",
		RunUUID:           run3.UUID,
		MLMDExecutionID:   "3",
		CreatedTimestamp:  5,
		FinishedTimestamp: 6,
		Fingerprint:       "1",
	}
	taskStore.CreateTask(task3)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdFour, nil)
	task4 := &model.Task{
		Namespace:         "ns2",
		PipelineName:      "namespace/ns2/pipeline/pipeline2",
		RunUUID:           run2.UUID,
		MLMDExecutionID:   "4",
		CreatedTimestamp:  5,
		FinishedTimestamp: 6,
		Fingerprint:       "1",
	}
	taskStore.CreateTask(task4)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakeTaskIdFive, nil)
	task5 := &model.Task{
		Namespace:         "ns2",
		PipelineName:      "namespace/ns2/pipeline/pipeline2",
		RunUUID:           run2.UUID,
		MLMDExecutionID:   "5",
		CreatedTimestamp:  7,
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
			RunUUID:           defaultFakeRunIdTwo,
			MLMDExecutionID:   "4",
			CreatedTimestamp:  5,
			FinishedTimestamp: 6,
			Fingerprint:       "1",
		}}
	expectedSecondPageTasks := []*model.Task{
		{
			UUID:              defaultFakeTaskIdFive,
			Namespace:         "ns2",
			PipelineName:      "namespace/ns2/pipeline/pipeline2",
			RunUUID:           defaultFakeRunIdTwo,
			MLMDExecutionID:   "5",
			CreatedTimestamp:  7,
			FinishedTimestamp: 8,
			Fingerprint:       "10",
		}}

	opts, err := list.NewOptions(&model.Task{}, 1, "", nil)
	assert.Nil(t, err)

	tasks, total_size, nextPageToken, err := taskStore.ListTasks(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Pipeline, ID: "namespace/ns2/pipeline/pipeline2"}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedFirstPageTasks, tasks, "Unexpected Tasks listed.")
	assert.NotEmpty(t, nextPageToken)
	fmt.Print("tasks")
	fmt.Print(nextPageToken)

	opts, err = list.NewOptionsFromToken(nextPageToken, 1)
	assert.Nil(t, err)
	tasks, total_size, nextPageToken, err = taskStore.ListTasks(
		&common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Pipeline, ID: "namespace/ns2/pipeline/pipeline2"}}, opts)
	assert.Nil(t, err)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, expectedSecondPageTasks, tasks, "Unexpected Tasks listed.")
	assert.Empty(t, nextPageToken)
}
