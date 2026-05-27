// Copyright 2025 The Kubeflow Authors
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

package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

// Helper function to create a run with tasks for testing
func initWithRunAndTasks(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Run) {
	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Create a run
	run := &model.Run{
		DisplayName: "test-run",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
	}
	createdRun, err := resourceManager.CreateRun(ctx, run)
	assert.Nil(t, err)

	// Create some tasks for this run
	for i := 1; i <= 3; i++ {
		task := &model.Task{
			Namespace: "ns1",
			RunUUID:   createdRun.UUID,
			Name:      fmt.Sprintf("task-%d", i),
			State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		}
		_, err := clientManager.TaskStore().CreateTask(task)
		assert.Nil(t, err)
	}

	return clientManager, resourceManager, createdRun
}

// TestGetRun_DefaultView tests that GetRun with DEFAULT view returns task_count but no tasks
func TestGetRun_DefaultView(t *testing.T) {
	clients, manager, createdRun := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Get run with DEFAULT view (or unspecified)
	defaultView := apiv2beta1.GetRunRequest_DEFAULT
	response, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: createdRun.UUID,
		View:  &defaultView,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, createdRun.UUID, response.RunId)

	// Verify task_count is populated
	assert.Equal(t, int32(3), response.TaskCount)

	// Verify tasks are NOT populated (or empty)
	assert.Nil(t, response.Tasks)
}

// TestGetRun_DefaultView_Unspecified tests that GetRun with no view parameter returns task_count but no tasks
func TestGetRun_DefaultView_Unspecified(t *testing.T) {
	clients, manager, createdRun := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Get run without specifying view (should default to DEFAULT behavior)
	response, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: createdRun.UUID,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, createdRun.UUID, response.RunId)

	// Verify task_count is populated
	assert.Equal(t, int32(3), response.TaskCount)

	// Verify tasks are NOT populated
	assert.Nil(t, response.Tasks)
}

// TestGetRun_FullView tests that GetRun with FULL view returns both task_count and full tasks
func TestGetRun_FullView(t *testing.T) {
	clients, manager, createdRun := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Get run with FULL view
	fullView := apiv2beta1.GetRunRequest_FULL
	response, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: createdRun.UUID,
		View:  &fullView,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, createdRun.UUID, response.RunId)

	// Verify task_count is populated
	assert.Equal(t, int32(3), response.TaskCount)

	// Verify tasks ARE populated with full details
	assert.NotNil(t, response.Tasks)
	assert.Equal(t, 3, len(response.Tasks))

	// Verify task details are populated
	for _, task := range response.Tasks {
		assert.NotEmpty(t, task.TaskId)
		assert.NotEmpty(t, task.Name)
		assert.Equal(t, createdRun.UUID, task.RunId)
		assert.NotNil(t, task.Inputs)
		assert.NotNil(t, task.Outputs)
	}
}

// TestListRuns_DefaultView tests that ListRuns with DEFAULT view returns task_count but no tasks
func TestListRuns_DefaultView(t *testing.T) {
	clients, manager, createdRun := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// List runs with DEFAULT view
	defaultView := apiv2beta1.ListRunsRequest_DEFAULT
	response, err := server.ListRuns(ctx, &apiv2beta1.ListRunsRequest{
		Namespace: "ns1",
		View:      &defaultView,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.True(t, len(response.Runs) > 0)

	// Find our created run
	var foundRun *apiv2beta1.Run
	for _, run := range response.Runs {
		if run.RunId == createdRun.UUID {
			foundRun = run
			break
		}
	}

	assert.NotNil(t, foundRun)

	// Verify task_count is populated
	assert.Equal(t, int32(3), foundRun.TaskCount)

	// Verify tasks are NOT populated
	assert.Nil(t, foundRun.Tasks)
}

// TestListRuns_DefaultView_Unspecified tests that ListRuns with no view parameter returns task_count but no tasks
func TestListRuns_DefaultView_Unspecified(t *testing.T) {
	clients, manager, createdRun := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// List runs without specifying view (should default to DEFAULT behavior)
	response, err := server.ListRuns(ctx, &apiv2beta1.ListRunsRequest{
		Namespace: "ns1",
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.True(t, len(response.Runs) > 0)

	// Find our created run
	var foundRun *apiv2beta1.Run
	for _, run := range response.Runs {
		if run.RunId == createdRun.UUID {
			foundRun = run
			break
		}
	}

	assert.NotNil(t, foundRun)

	// Verify task_count is populated
	assert.Equal(t, int32(3), foundRun.TaskCount)

	// Verify tasks are NOT populated
	assert.Nil(t, foundRun.Tasks)
}

// TestListRuns_FullView tests that ListRuns with FULL view returns both task_count and full tasks
func TestListRuns_FullView(t *testing.T) {
	clients, manager, createdRun := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// List runs with FULL view
	fullView := apiv2beta1.ListRunsRequest_FULL
	response, err := server.ListRuns(ctx, &apiv2beta1.ListRunsRequest{
		Namespace: "ns1",
		View:      &fullView,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.True(t, len(response.Runs) > 0)

	// Find our created run
	var foundRun *apiv2beta1.Run
	for _, run := range response.Runs {
		if run.RunId == createdRun.UUID {
			foundRun = run
			break
		}
	}

	assert.NotNil(t, foundRun)

	// Verify task_count is populated
	assert.Equal(t, int32(3), foundRun.TaskCount)

	// Verify tasks ARE populated with full details
	assert.NotNil(t, foundRun.Tasks)
	assert.Equal(t, 3, len(foundRun.Tasks))

	// Verify task details are populated
	for _, task := range foundRun.Tasks {
		assert.NotEmpty(t, task.TaskId)
		assert.NotEmpty(t, task.Name)
		assert.Equal(t, createdRun.UUID, task.RunId)
		assert.NotNil(t, task.Inputs)
		assert.NotNil(t, task.Outputs)
	}
}

// TestGetRun_NoTasks tests that GetRun returns task_count of 0 when run has no tasks
func TestGetRun_NoTasks(t *testing.T) {
	clients, manager, _ := initWithOneTimeRunV2(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Create a run without tasks
	run := &model.Run{
		DisplayName: "test-run-no-tasks",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
		},
	}
	createdRun, err := manager.CreateRun(ctx, run)
	assert.Nil(t, err)

	// Get run with DEFAULT view
	defaultView := apiv2beta1.GetRunRequest_DEFAULT
	response, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: createdRun.UUID,
		View:  &defaultView,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)

	// Verify task_count is 0
	assert.Equal(t, int32(0), response.TaskCount)

	// Verify tasks are not populated
	assert.Nil(t, response.Tasks)
}

// TestTaskCount_AlwaysPopulated tests that task_count is always populated regardless of view
func TestTaskCount_AlwaysPopulated(t *testing.T) {
	clients, manager, createdRun := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Test with DEFAULT view
	defaultView := apiv2beta1.GetRunRequest_DEFAULT
	defaultResponse, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: createdRun.UUID,
		View:  &defaultView,
	})
	assert.Nil(t, err)
	assert.Equal(t, int32(3), defaultResponse.TaskCount)

	// Test with FULL view
	fullView := apiv2beta1.GetRunRequest_FULL
	fullResponse, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: createdRun.UUID,
		View:  &fullView,
	})
	assert.Nil(t, err)
	assert.Equal(t, int32(3), fullResponse.TaskCount)

	// Both should have the same task count
	assert.Equal(t, defaultResponse.TaskCount, fullResponse.TaskCount)
}

func TestListRunsPageSizeForView(t *testing.T) {
	originalFullViewMaxPageSize := common.GetIntConfigWithDefault(
		common.ListRunsFullViewMaxPageSize,
		defaultListRunsFullViewMaxPageSize,
	)
	t.Cleanup(func() {
		viper.Set(common.ListRunsFullViewMaxPageSize, originalFullViewMaxPageSize)
	})

	fullView := apiv2beta1.ListRunsRequest_FULL
	defaultView := apiv2beta1.ListRunsRequest_DEFAULT

	viper.Set(common.ListRunsFullViewMaxPageSize, 7)

	assert.Equal(t, 7, listRunsPageSizeForView(10, &fullView))
	assert.Equal(t, 5, listRunsPageSizeForView(5, &fullView))
	assert.Equal(t, defaultPageSize, listRunsPageSizeForView(0, &defaultView))
	assert.Equal(t, 7, listRunsPageSizeForView(0, &fullView))
	assert.Equal(t, 10, listRunsPageSizeForView(10, &defaultView))
	assert.Equal(t, -1, listRunsPageSizeForView(-1, &fullView))

	viper.Set(common.ListRunsFullViewMaxPageSize, 0)
	assert.Equal(t, defaultListRunsFullViewMaxPageSize, listRunsPageSizeForView(150, &fullView))
}

func TestGetRun_FullView_OrdersTasksAndChildTasksDeterministically(t *testing.T) {
	clients, manager, run := initWithOneTimeRunV2(t)
	defer clients.Close()

	server := createRunServer(manager)
	taskStore := clients.TaskStore()

	parentTask, err := taskStore.CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "parent",
		CreatedAtInSec: 30,
		StartedInSec:   30,
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "child-late",
		ParentTaskUUID: &parentTask.UUID,
		CreatedAtInSec: 25,
		StartedInSec:   25,
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "sibling-early",
		CreatedAtInSec: 10,
		StartedInSec:   10,
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "child-early",
		ParentTaskUUID: &parentTask.UUID,
		CreatedAtInSec: 15,
		StartedInSec:   15,
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)

	fullView := apiv2beta1.GetRunRequest_FULL
	response, err := server.GetRun(context.Background(), &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	assert.NoError(t, err)
	if assert.Len(t, response.GetTasks(), 4) {
		assert.Equal(t, "sibling-early", response.GetTasks()[0].GetName())
		assert.Equal(t, "child-early", response.GetTasks()[1].GetName())
		assert.Equal(t, "child-late", response.GetTasks()[2].GetName())
		assert.Equal(t, "parent", response.GetTasks()[3].GetName())
	}

	var apiParent *apiv2beta1.PipelineTask
	for _, task := range response.GetTasks() {
		if task.GetTaskId() == parentTask.UUID {
			apiParent = task
			break
		}
	}
	if assert.NotNil(t, apiParent) && assert.Len(t, apiParent.GetChildTasks(), 2) {
		assert.Equal(t, "child-early", apiParent.GetChildTasks()[0].GetName())
		assert.Equal(t, "child-late", apiParent.GetChildTasks()[1].GetName())
	}
}

func TestGetRun_FullView_AfterReportWorkflowHydratesTasks(t *testing.T) {
	clients, manager, run := initWithOneTimeRunV2(t)
	defer clients.Close()

	server := createRunServer(manager)
	_, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "reported-task",
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)

	workflow := util.NewWorkflow(testWorkflow.DeepCopy())
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowFailed
	workflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		"node-1": {
			Name:  "pod-1",
			Type:  v1alpha1.NodeTypePod,
			Phase: v1alpha1.NodeFailed,
		},
	}

	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	assert.NoError(t, err)

	fullView := apiv2beta1.GetRunRequest_FULL
	response, err := server.GetRun(context.Background(), &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), response.GetTaskCount())
	if assert.Len(t, response.GetTasks(), 1) {
		assert.Equal(t, run.UUID, response.GetTasks()[0].GetRunId())
		assert.Equal(t, "reported-task", response.GetTasks()[0].GetName())
	}
}

func TestListRuns_FullView_UsesConfiguredMaxPageSize(t *testing.T) {
	originalFullViewMaxPageSize := common.GetIntConfigWithDefault(
		common.ListRunsFullViewMaxPageSize,
		defaultListRunsFullViewMaxPageSize,
	)
	t.Cleanup(func() {
		viper.Set(common.ListRunsFullViewMaxPageSize, originalFullViewMaxPageSize)
	})
	viper.Set(common.ListRunsFullViewMaxPageSize, 2)

	clients, manager, _ := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)

	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	for index := 0; index < 3; index++ {
		run := &model.Run{
			DisplayName: fmt.Sprintf("test-run-extra-%d", index),
			Namespace:   "ns1",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
			},
		}
		extraRun, err := manager.CreateRun(ctx, run)
		assert.NoError(t, err)

		_, err = clients.TaskStore().CreateTask(&model.Task{
			Namespace: "ns1",
			RunUUID:   extraRun.UUID,
			Name:      fmt.Sprintf("task-extra-%d", index),
			State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		})
		assert.NoError(t, err)
	}

	fullView := apiv2beta1.ListRunsRequest_FULL
	response, err := server.ListRuns(ctx, &apiv2beta1.ListRunsRequest{
		Namespace: "ns1",
		View:      &fullView,
		PageSize:  10,
	})

	assert.NoError(t, err)
	assert.Len(t, response.Runs, 2)
	assert.NotEmpty(t, response.NextPageToken)

	for _, run := range response.Runs {
		assert.NotNil(t, run.Tasks)
		assert.Greater(t, run.TaskCount, int32(0))
	}
}
