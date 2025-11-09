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

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	testUUID1 = "123e4567-e89b-12d3-a456-426655441011"
	testUUID2 = "123e4567-e89b-12d3-a456-426655441012"
	testUUID3 = "123e4567-e89b-12d3-a456-426655441013"
	testUUID4 = "123e4567-e89b-12d3-a456-426655441014"
	testUUID5 = "123e4567-e89b-12d3-a456-426655441015"
)

// initializeTaskStore sets up a fake DB with a couple of runs and returns a TaskStore ready for testing.
func initializeTaskStore() (*DB, *TaskStore, *RunStore) {
	db := NewFakeDBOrFatal()
	fakeTime := util.NewFakeTimeForEpoch()
	// Seed a couple of runs to satisfy Task foreign key constraint.
	runStore := NewRunStore(db, fakeTime)
	run1 := &model.Run{
		UUID:         "run-1",
		ExperimentId: "exp-1",
		K8SName:      "run1",
		DisplayName:  "run1",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
			State:            model.RuntimeStateRunning,
		},
	}
	run2 := &model.Run{
		UUID:         "run-2",
		ExperimentId: "exp-2",
		K8SName:      "run2",
		DisplayName:  "run2",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns2",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Succeeded",
			State:            model.RuntimeStateSucceeded,
		},
	}
	_, _ = runStore.CreateRun(run1)
	_, _ = runStore.CreateRun(run2)

	// Create task store with controllable UUID generator
	taskStore := NewTaskStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil))
	return db, taskStore, runStore
}

func createTaskPod(name, uid string, typ apiv2beta1.PipelineTaskDetail_TaskPodType) *apiv2beta1.PipelineTaskDetail_TaskPod {
	return &apiv2beta1.PipelineTaskDetail_TaskPod{
		Name: name,
		Uid:  uid,
		Type: typ,
	}
}

func createTaskPodsAsJSONSlice(pods ...*apiv2beta1.PipelineTaskDetail_TaskPod) model.JSONSlice {
	podsAsSlice, err := model.ProtoSliceToJSONSlice(pods)
	if err != nil {
		panic(err)
	}
	return podsAsSlice
}

// Minimal test to ensure model<->DB mapping remains valid.
func TestTaskAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.Task{}).APIToModelFieldMap() {
		assert.Contains(t, taskColumns, modelField)
	}
}

func TestCreateTask_Success(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()
	pods := createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR))
	task := &model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             pods,
		Fingerprint:      "fp-1",
		Name:             "taskA",
		ParentTaskUUID:   strPTR(""),
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData(map[string]interface{}{"k": "v"}),
	}

	created, err := taskStore.CreateTask(task)
	assert.NoError(t, err)
	assert.Equal(t, testUUID1, created.UUID)
	// CreatedAt and StartedInSec should be auto-populated to the same timestamp (fake time starts from 0 -> 1)
	assert.Equal(t, created.CreatedAtInSec, created.StartedInSec)
	assert.Greater(t, created.CreatedAtInSec, int64(0))

	// Verify it can be fetched back
	fetched, err := taskStore.GetTask(created.UUID)
	assert.NoError(t, err)
	assert.Equal(t, created.UUID, fetched.UUID)
	assert.Equal(t, "ns1", fetched.Namespace)
	assert.Equal(t, "run-1", fetched.RunUUID)
	assert.Equal(t, pods, fetched.Pods)
	assert.Equal(t, "fp-1", fetched.Fingerprint)
	assert.Equal(t, "taskA", fetched.Name)
	assert.Equal(t, model.TaskStatus(1), fetched.State)
	assert.Equal(t, model.TaskType(0), fetched.Type)
}

func TestGetTask_NotFound(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()
	_, err := taskStore.GetTask(testUUID1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestListTasks_BasicAndFilters(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create a parent task and two child tasks under different runs/namespaces
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	parent, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-parent",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		ParentTaskUUID:   strPTR(parent.UUID),
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p2", "uid2", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-c1",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		RunUUID:          "run-2",
		ParentTaskUUID:   strPTR(parent.UUID),
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p3", "uid3", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-c2",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	// List all tasks
	opts, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	all, total, npt, err := taskStore.ListTasks(&model.FilterContext{}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(all))
	assert.Equal(t, 3, total)
	assert.Equal(t, "", npt)

	// Filter by RunUUID
	opts2, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	runFiltered, total2, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "run-1"}}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(runFiltered))
	assert.Equal(t, 2, total2)

	// Filter by ParentTaskUUID (child tasks)
	opts3, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	children, total3, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: parent.UUID}}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(children))
	assert.Equal(t, 2, total3)
}

func TestUpdateTask_Success(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	pod1 := createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)
	pod2 := createTaskPod("p2", "uid2", apiv2beta1.PipelineTaskDetail_EXECUTOR)
	// Create a task
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(pod1),
		Fingerprint:      "fp-0",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Deep copy the task for updates
	updatedTask := &model.Task{
		UUID:             created.UUID,
		Namespace:        created.Namespace,
		RunUUID:          created.RunUUID,
		Pods:             created.Pods,
		Fingerprint:      created.Fingerprint,
		Name:             created.Name,
		DisplayName:      created.DisplayName,
		ParentTaskUUID:   created.ParentTaskUUID,
		State:            created.State,
		StateHistory:     created.StateHistory,
		InputParameters:  created.InputParameters,
		OutputParameters: created.OutputParameters,
		Type:             created.Type,
		TypeAttrs:        created.TypeAttrs,
		CreatedAtInSec:   created.CreatedAtInSec,
		StartedInSec:     created.StartedInSec,
	}

	// Update some fields
	updatedTask.Name = "updatedName"
	updatedTask.Fingerprint = "fp-1"
	updatedTask.Pods = createTaskPodsAsJSONSlice(pod1, pod2)
	updatedTask.State = 2
	updated, err := taskStore.UpdateTask(updatedTask)
	assert.NoError(t, err)
	assert.Equal(t, created.UUID, updated.UUID)
	assert.Equal(t, "updatedName", updated.Name)
	assert.Equal(t, "fp-1", updated.Fingerprint)
	assert.Equal(t, createTaskPodsAsJSONSlice(pod1, pod2), updated.Pods)
	assert.Equal(t, model.TaskStatus(2), updated.State)
}

func TestUpdateTask_MergesParameters(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create a task with initial input parameters
	val1, _ := structpb.NewValue("initial-input")
	initialParam := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val1,
		ParameterKey: "common-param",
		Type:         apiv2beta1.IOType_COMPONENT_INPUT,
	}
	initialParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{initialParam})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-0",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  initialParams,
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Simulate first update from iteration 0
	valIter0, _ := structpb.NewValue("output-from-iter-0")
	iter0Param := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        valIter0,
		ParameterKey: "loop-output",
		Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "loop-task",
			Iteration: int64PTR(0),
		},
	}
	iter0Params, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{iter0Param})
	assert.NoError(t, err)

	update1 := &model.Task{
		UUID:             created.UUID,
		OutputParameters: iter0Params,
	}
	updated1, err := taskStore.UpdateTask(update1)
	assert.NoError(t, err)

	// Verify first update has both initial input param and iter0 output param
	assert.Equal(t, 1, len(updated1.InputParameters))
	assert.Equal(t, 1, len(updated1.OutputParameters))

	// Simulate second update from iteration 1
	valIter1, _ := structpb.NewValue("output-from-iter-1")
	iter1Param := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        valIter1,
		ParameterKey: "loop-output",
		Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "loop-task",
			Iteration: int64PTR(1),
		},
	}
	iter1Params, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{iter1Param})
	assert.NoError(t, err)

	update2 := &model.Task{
		UUID:             created.UUID,
		OutputParameters: iter1Params,
	}
	updated2, err := taskStore.UpdateTask(update2)
	assert.NoError(t, err)

	// Verify second update preserves both iter0 and iter1 output params
	assert.Equal(t, 1, len(updated2.InputParameters))
	assert.Equal(t, 2, len(updated2.OutputParameters), "Should have both iteration 0 and 1 output parameters")

	// Verify both iterations are present
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
		return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}
	outputProtos, err := model.JSONSliceToProtoSlice(updated2.OutputParameters, typeFunc)
	assert.NoError(t, err)

	iterations := make(map[int64]bool)
	for _, p := range outputProtos {
		if p.Producer != nil && p.Producer.Iteration != nil {
			iterations[*p.Producer.Iteration] = true
		}
	}
	assert.True(t, iterations[0], "Should have iteration 0 parameter")
	assert.True(t, iterations[1], "Should have iteration 1 parameter")
}

func TestGetChildTasks_ReturnsChildren(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	parent, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Name:             "parent",
		DisplayName:      "Parent Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-p",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		ParentTaskUUID:   strPTR(parent.UUID),
		Name:             "child-a",
		DisplayName:      "First Child",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-a",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		ParentTaskUUID:   strPTR(parent.UUID),
		Name:             "child-b",
		DisplayName:      "Second Child",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-b",
		State:            2,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	children, err := taskStore.GetChildTasks(parent.UUID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(children))
}

func TestListTasks_FilterPredicates_EqualsOnColumns(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Seed 3 tasks across 2 runs with differing names, statuses and fingerprints.
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	_, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Name:             "alpha",
		DisplayName:      "Alpha Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-alpha",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Name:             "beta",
		DisplayName:      "Beta Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-beta",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		RunUUID:          "run-2",
		Name:             "gamma",
		DisplayName:      "Gamma Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-gamma",
		State:            2,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// name == "beta"
	f1Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "name", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_StringValue{StringValue: "beta"}},
		}}
	f1, err := filter.New(f1Proto)
	assert.NoError(t, err)
	opts1, err := list.NewOptions(&model.Task{}, 20, "", f1)
	assert.NoError(t, err)
	res1, total1, _, err := taskStore.ListTasks(&model.FilterContext{}, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res1))
	assert.Equal(t, 1, total1)
	assert.Equal(t, "beta", res1[0].Name)

	// status == 2
	f2Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "status", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_IntValue{IntValue: 2}},
		},
	}
	f2, err := filter.New(f2Proto)
	assert.NoError(t, err)
	opts2, err := list.NewOptions(&model.Task{}, 20, "", f2)
	assert.NoError(t, err)
	res2, total2, _, err := taskStore.ListTasks(&model.FilterContext{}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res2))
	assert.Equal(t, 1, total2)
	assert.Equal(t, model.TaskStatus(2), res2[0].State)

	// cache_fingerprint == "fp-alpha"
	f3Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "cache_fingerprint", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_StringValue{StringValue: "fp-alpha"}},
		}}
	f3, err := filter.New(f3Proto)
	assert.NoError(t, err)
	opts3, err := list.NewOptions(&model.Task{}, 20, "", f3)
	assert.NoError(t, err)
	res3, total3, _, err := taskStore.ListTasks(&model.FilterContext{}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res3))
	assert.Equal(t, 1, total3)
	assert.Equal(t, "fp-alpha", res3[0].Fingerprint)

	// Combined: run_id == "run-1" AND status == 1
	f4Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "run_id", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_StringValue{StringValue: "run-1"}},
			{Key: "status", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_IntValue{IntValue: 1}},
		}}
	f4, err := filter.New(f4Proto)
	assert.NoError(t, err)
	opts4, err := list.NewOptions(&model.Task{}, 20, "", f4)
	assert.NoError(t, err)
	res4, total4, _, err := taskStore.ListTasks(&model.FilterContext{}, opts4)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(res4))
	assert.Equal(t, 2, total4)
}

func TestListTasks_PaginationWithToken(t *testing.T) {
	// Setup
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Seed 5 tasks. FakeTime.Now() increments per call, so CreatedAtInSec/StartedInSec are strictly increasing
	uuids := []string{testUUID1, testUUID2, testUUID3, testUUID4, testUUID5}
	for i, id := range uuids {
		// Control UUID so key tie-breaker is predictable if same timestamp ever occurred
		taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(id, nil)
		_, err := taskStore.CreateTask(&model.Task{
			Namespace:        "ns1",
			RunUUID:          "run-1",
			Name:             fmt.Sprintf("task-%d", i+1),
			Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
			Fingerprint:      fmt.Sprintf("fp-%d", i+1),
			State:            1,
			StateHistory:     model.JSONSlice{},
			InputParameters:  model.JSONSlice{},
			OutputParameters: model.JSONSlice{},
			Type:             0,
			TypeAttrs:        map[string]interface{}{},
		})
		assert.NoError(t, err)
	}

	// Page 1
	opts1, _ := list.NewOptions(&model.Task{}, 2, "", nil)
	page1, total1, token1, err := taskStore.ListTasks(&model.FilterContext{}, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page1))
	assert.Equal(t, 5, total1)
	assert.NotEmpty(t, token1)

	// Page 2 using token
	opts2, err := list.NewOptionsFromToken(token1, 2)
	assert.NoError(t, err)
	page2, total2, token2, err := taskStore.ListTasks(&model.FilterContext{}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page2))
	assert.Equal(t, 5, total2)
	assert.NotEmpty(t, token2)

	// Page 3 using token (should be the last 1 item, then empty token)
	opts3, err := list.NewOptionsFromToken(token2, 2)
	assert.NoError(t, err)
	page3, total3, token3, err := taskStore.ListTasks(&model.FilterContext{}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(page3))
	assert.Equal(t, 5, total3)
	assert.Empty(t, token3)
}

func TestTaskParameters_PersistAndFetch(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Build two simple IOParameter protos for inputs and outputs
	inVal, _ := structpb.NewValue("in-val")
	outVal, _ := structpb.NewValue("out-val")
	inParam := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        inVal,
		ParameterKey: "in-name",
	}
	outParam := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        outVal,
		ParameterKey: "param-y",
		Producer: &apiv2beta1.IOProducer{
			TaskName: "task-x",
		},
	}
	inParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{inParam})
	assert.NoError(t, err)
	outParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{outParam})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-param",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  inParams,
		OutputParameters: outParams,
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	fetched, err := taskStore.GetTask(created.UUID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fetched.InputParameters))
	assert.Equal(t, 1, len(fetched.OutputParameters))
}

func TestHydrateArtifactsForTask_GetAndList(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create a task under run-1
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	task, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-art",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	// Create two artifacts via the ArtifactStore
	artifactStore := NewArtifactStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil))
	artIn, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      0,
		URI:       strPTR("s3://bucket/in"),
		Name:      "in-art",
	})
	assert.NoError(t, err)
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	artOut, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      0,
		URI:       strPTR("s3://bucket/out"),
		Name:      "out-art",
	})
	assert.NoError(t, err)

	// Link artifacts to task via artifact_tasks
	ats1 := NewArtifactTaskStore(db, util.NewFakeUUIDGeneratorOrFatal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", nil))
	// Input link with no producer fields -> ResolvedValue
	_, err = ats1.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  artIn.UUID,
		TaskID:      task.UUID,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		RunUUID:     task.RunUUID,
		ArtifactKey: "input-key",
	})
	assert.NoError(t, err)
	// Output link with producer fields -> PipelineChannel
	ats2 := NewArtifactTaskStore(db, util.NewFakeUUIDGeneratorOrFatal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2", nil))
	_, err = ats2.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID: artOut.UUID,
		TaskID:     task.UUID,
		Type:       model.IOType(apiv2beta1.IOType_OUTPUT),
		RunUUID:    task.RunUUID,
		Producer: model.JSONData{
			"taskName": "producer-task",
		},
		ArtifactKey: "output-key",
	})
	assert.NoError(t, err)

	// Verify GetTask hydrates artifacts
	fetched, err := taskStore.GetTask(task.UUID)
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(fetched.InputArtifactsHydrated)) {
		ia := fetched.InputArtifactsHydrated[0]
		assert.Equal(t, "input-key", ia.Key)
		if assert.NotNil(t, ia.Value) {
			assert.Equal(t, "in-art", ia.Value.Name)
		}
	}
	if assert.Equal(t, 1, len(fetched.OutputArtifactsHydrated)) {
		oa := fetched.OutputArtifactsHydrated[0]
		assert.NotNil(t, oa.Producer)
		assert.Equal(t, "producer-task", oa.Producer.TaskName)
		assert.Equal(t, "output-key", oa.Key)
		if assert.NotNil(t, oa.Value) {
			assert.Equal(t, "out-art", oa.Value.Name)
		}
	}

	// Verify ListTasks hydrates artifacts as well
	opts, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	tasks, _, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: task.RunUUID}}, opts)
	assert.NoError(t, err)
	var found *model.Task
	for _, tsk := range tasks {
		if tsk.UUID == task.UUID {
			found = tsk
			break
		}
	}
	if assert.NotNil(t, found) {
		assert.Equal(t, 1, len(found.InputArtifactsHydrated))
		assert.Equal(t, 1, len(found.OutputArtifactsHydrated))
	}
}

func int64PTR(i int64) *int64 { return &i }

func TestMergeParameters_EmptySlices(t *testing.T) {
	// Test merging nil slices - returns empty slice (semantically equivalent to nil)
	result, err := mergeParameters(nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result))

	// Test merging empty slices
	result, err = mergeParameters(model.JSONSlice{}, model.JSONSlice{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result))

	// Test merging nil with non-empty
	val1, _ := structpb.NewValue("value1")
	param1 := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val1,
		ParameterKey: "param1",
		Type:         apiv2beta1.IOType_COMPONENT_INPUT,
	}
	params1, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param1})
	assert.NoError(t, err)

	result, err = mergeParameters(nil, params1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))

	result, err = mergeParameters(params1, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
}

func TestMergeParameters_NoOverlap(t *testing.T) {
	// Create two parameters with different keys
	val1, _ := structpb.NewValue("value1")
	param1 := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val1,
		ParameterKey: "param1",
		Type:         apiv2beta1.IOType_COMPONENT_INPUT,
	}

	val2, _ := structpb.NewValue("value2")
	param2 := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val2,
		ParameterKey: "param2",
		Type:         apiv2beta1.IOType_OUTPUT,
	}

	oldParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param1})
	assert.NoError(t, err)
	newParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param2})
	assert.NoError(t, err)

	result, err := mergeParameters(oldParams, newParams)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))

	// Convert back to verify both parameters are present
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
		return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}
	resultProtos, err := model.JSONSliceToProtoSlice(result, typeFunc)
	assert.NoError(t, err)

	keys := make(map[string]bool)
	for _, p := range resultProtos {
		keys[p.ParameterKey] = true
	}
	assert.True(t, keys["param1"])
	assert.True(t, keys["param2"])
}

func TestMergeParameters_WithProducer_NoIteration(t *testing.T) {
	// Create parameters with producer but no iteration
	val1, _ := structpb.NewValue("value-from-task1")
	param1 := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val1,
		ParameterKey: "output-param",
		Type:         apiv2beta1.IOType_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName: "task1",
		},
	}

	val2, _ := structpb.NewValue("value-from-task2")
	param2 := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val2,
		ParameterKey: "output-param",
		Type:         apiv2beta1.IOType_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName: "task2",
		},
	}

	oldParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param1})
	assert.NoError(t, err)
	newParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param2})
	assert.NoError(t, err)

	result, err := mergeParameters(oldParams, newParams)
	assert.NoError(t, err)
	// Different task names create different keys, so we should have 2 parameters
	assert.Equal(t, 2, len(result))

	typeFunc := func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
		return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}
	resultProtos, err := model.JSONSliceToProtoSlice(result, typeFunc)
	assert.NoError(t, err)

	taskNames := make(map[string]bool)
	for _, p := range resultProtos {
		taskNames[p.Producer.TaskName] = true
	}
	assert.True(t, taskNames["task1"])
	assert.True(t, taskNames["task2"])
}

func TestMergeParameters_WithProducer_WithIteration(t *testing.T) {
	// Create parameters with producer including iteration
	val1, _ := structpb.NewValue("value-iteration-0")
	param1 := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val1,
		ParameterKey: "loop-output",
		Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "loop-task",
			Iteration: int64PTR(0),
		},
	}

	val2, _ := structpb.NewValue("value-iteration-1")
	param2 := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val2,
		ParameterKey: "loop-output",
		Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "loop-task",
			Iteration: int64PTR(1),
		},
	}

	oldParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param1})
	assert.NoError(t, err)
	newParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{param2})
	assert.NoError(t, err)

	result, err := mergeParameters(oldParams, newParams)
	assert.NoError(t, err)
	// Different iterations create different keys, so we should have 2 parameters
	assert.Equal(t, 2, len(result))

	typeFunc := func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
		return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}
	resultProtos, err := model.JSONSliceToProtoSlice(result, typeFunc)
	assert.NoError(t, err)

	iterations := make(map[int64]bool)
	for _, p := range resultProtos {
		if p.Producer != nil && p.Producer.Iteration != nil {
			iterations[*p.Producer.Iteration] = true
		}
	}
	assert.True(t, iterations[0])
	assert.True(t, iterations[1])
}

func TestMergeParameters_RaceConditionScenario(t *testing.T) {
	// Simulate a race condition where two driver tasks from different iterations
	// within a loop try to update parameters simultaneously

	// Initial state - task already has some parameters
	valExisting, _ := structpb.NewValue("existing-param")
	existingParam := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        valExisting,
		ParameterKey: "common-param",
		Type:         apiv2beta1.IOType_COMPONENT_INPUT,
	}

	// Update from iteration 0
	valIter0, _ := structpb.NewValue("output-from-iter-0")
	iter0Param := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        valIter0,
		ParameterKey: "loop-output",
		Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "loop-task",
			Iteration: int64PTR(0),
		},
	}

	// Update from iteration 1 (happening concurrently)
	valIter1, _ := structpb.NewValue("output-from-iter-1")
	iter1Param := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        valIter1,
		ParameterKey: "loop-output",
		Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "loop-task",
			Iteration: int64PTR(1),
		},
	}

	existingParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{existingParam})
	assert.NoError(t, err)

	// First update: merge existing with iteration 0
	iter0Update, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{iter0Param})
	assert.NoError(t, err)

	result1, err := mergeParameters(existingParams, iter0Update)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result1))

	// Second update: merge result1 with iteration 1
	iter1Update, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{iter1Param})
	assert.NoError(t, err)

	result2, err := mergeParameters(result1, iter1Update)
	assert.NoError(t, err)
	// Should have 3 parameters: existing + iter0 + iter1
	assert.Equal(t, 3, len(result2))

	// Verify all three parameters are present
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
		return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}
	resultProtos, err := model.JSONSliceToProtoSlice(result2, typeFunc)
	assert.NoError(t, err)

	hasExisting := false
	hasIter0 := false
	hasIter1 := false

	for _, p := range resultProtos {
		if p.ParameterKey == "common-param" && p.Producer == nil {
			hasExisting = true
		}
		if p.Producer != nil && p.Producer.Iteration != nil {
			if *p.Producer.Iteration == 0 {
				hasIter0 = true
			}
			if *p.Producer.Iteration == 1 {
				hasIter1 = true
			}
		}
	}

	assert.True(t, hasExisting, "Should preserve existing parameter")
	assert.True(t, hasIter0, "Should preserve iteration 0 parameter")
	assert.True(t, hasIter1, "Should preserve iteration 1 parameter")
}

// TestCreateTask_AutoPopulatesStateHistory verifies that state_history is automatically
// populated when creating a task with a state (mirrors Run behavior).
func TestCreateTask_AutoPopulatesStateHistory(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	pods := createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR))
	task := &model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             pods,
		Fingerprint:      "fp-1",
		Name:             "taskA",
		State:            model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING),
		StateHistory:     model.JSONSlice{}, // Empty state history
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData(map[string]interface{}{"k": "v"}),
	}

	created, err := taskStore.CreateTask(task)
	assert.NoError(t, err)

	// Verify state_history was auto-populated with initial state
	assert.NotNil(t, created.StateHistory)
	assert.Equal(t, 1, len(created.StateHistory), "Should have exactly 1 state history entry")

	// Fetch task from DB to verify persistence
	fetched, err := taskStore.GetTask(created.UUID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fetched.StateHistory), "Fetched task should have 1 state history entry")

	// Convert and verify state
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_TaskStatus {
		return &apiv2beta1.PipelineTaskDetail_TaskStatus{}
	}
	histProtos, err := model.JSONSliceToProtoSlice(fetched.StateHistory, typeFunc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(histProtos))
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, histProtos[0].GetState())
	assert.NotNil(t, histProtos[0].GetUpdateTime())
	assert.Greater(t, histProtos[0].GetUpdateTime().GetSeconds(), int64(0))
}

// TestUpdateTask_AutoPopulatesStateHistory verifies that state transitions
// are automatically tracked in state_history when updating a task.
func TestUpdateTask_AutoPopulatesStateHistory(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create initial task in RUNNING state
	pods := createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR))
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             pods,
		Fingerprint:      "fp-0",
		State:            model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING),
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(created.StateHistory), "Should have 1 entry after creation")

	// Update to SUCCEEDED state
	update := &model.Task{
		UUID:  created.UUID,
		State: model.TaskStatus(apiv2beta1.PipelineTaskDetail_SUCCEEDED),
	}
	updated, err := taskStore.UpdateTask(update)
	assert.NoError(t, err)

	// Should now have 2 entries: RUNNING and SUCCEEDED
	assert.Equal(t, 2, len(updated.StateHistory), "Should have 2 state history entries after state change")

	// Verify states in order
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_TaskStatus {
		return &apiv2beta1.PipelineTaskDetail_TaskStatus{}
	}
	histProtos, err := model.JSONSliceToProtoSlice(updated.StateHistory, typeFunc)
	assert.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, histProtos[0].GetState())
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, histProtos[1].GetState())
	assert.Greater(t, histProtos[1].GetUpdateTime().GetSeconds(), histProtos[0].GetUpdateTime().GetSeconds(),
		"Second state timestamp should be after first")
}

// TestUpdateTask_StateHistory_MultipleTransitions verifies that multiple
// state transitions are all captured in history.
func TestUpdateTask_StateHistory_MultipleTransitions(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	pods := createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR))
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          "run-1",
		Pods:             pods,
		Fingerprint:      "fp-0",
		State:            model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING),
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Transition: RUNNING → SUCCEEDED
	update1 := &model.Task{
		UUID:  created.UUID,
		State: model.TaskStatus(apiv2beta1.PipelineTaskDetail_SUCCEEDED),
	}
	updated1, err := taskStore.UpdateTask(update1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(updated1.StateHistory))

	// Transition: SUCCEEDED → FAILED (hypothetical retry scenario)
	update2 := &model.Task{
		UUID:  created.UUID,
		State: model.TaskStatus(apiv2beta1.PipelineTaskDetail_FAILED),
	}
	updated2, err := taskStore.UpdateTask(update2)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(updated2.StateHistory))

	// Verify all states are preserved
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_TaskStatus {
		return &apiv2beta1.PipelineTaskDetail_TaskStatus{}
	}
	histProtos, err := model.JSONSliceToProtoSlice(updated2.StateHistory, typeFunc)
	assert.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, histProtos[0].GetState())
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, histProtos[1].GetState())
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_FAILED, histProtos[2].GetState())
}

// Test helper function getLastTaskState
func TestGetLastTaskState(t *testing.T) {
	// Empty history
	assert.Equal(t, model.TaskStatus(0), getLastTaskState(model.JSONSlice{}))
	assert.Equal(t, model.TaskStatus(0), getLastTaskState(nil))

	// Valid history with one entry
	status1 := &apiv2beta1.PipelineTaskDetail_TaskStatus{
		UpdateTime: &timestamppb.Timestamp{Seconds: 100},
		State:      apiv2beta1.PipelineTaskDetail_RUNNING,
	}
	history1, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_TaskStatus{status1})
	assert.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING), getLastTaskState(history1))

	// Valid history with multiple entries
	status2 := &apiv2beta1.PipelineTaskDetail_TaskStatus{
		UpdateTime: &timestamppb.Timestamp{Seconds: 200},
		State:      apiv2beta1.PipelineTaskDetail_SUCCEEDED,
	}
	history2, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_TaskStatus{status1, status2})
	assert.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTaskDetail_SUCCEEDED), getLastTaskState(history2))
}
