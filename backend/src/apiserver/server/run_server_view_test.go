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
	"net"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			PipelineSpecManifest: model.LargeText(testIRPipeline),
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
			State:     model.TaskStatus(apiv2.PipelineTask_RUNNING),
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
	defaultView := apiv2.GetRunRequest_DEFAULT
	response, err := server.GetRun(ctx, &apiv2.GetRunRequest{
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
	response, err := server.GetRun(ctx, &apiv2.GetRunRequest{
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
	fullView := apiv2.GetRunRequest_FULL
	response, err := server.GetRun(ctx, &apiv2.GetRunRequest{
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
	defaultView := apiv2.ListRunsRequest_DEFAULT
	response, err := server.ListRuns(ctx, &apiv2.ListRunsRequest{
		Namespace: "ns1",
		View:      &defaultView,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.True(t, len(response.Runs) > 0)

	// Find our created run
	var foundRun *apiv2.Run
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
	response, err := server.ListRuns(ctx, &apiv2.ListRunsRequest{
		Namespace: "ns1",
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.True(t, len(response.Runs) > 0)

	// Find our created run
	var foundRun *apiv2.Run
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
	fullView := apiv2.ListRunsRequest_FULL
	response, err := server.ListRuns(ctx, &apiv2.ListRunsRequest{
		Namespace: "ns1",
		View:      &fullView,
	})

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.True(t, len(response.Runs) > 0)

	// Find our created run
	var foundRun *apiv2.Run
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
			PipelineSpecManifest: model.LargeText(testIRPipeline),
		},
	}
	createdRun, err := manager.CreateRun(ctx, run)
	assert.Nil(t, err)

	// Get run with DEFAULT view
	defaultView := apiv2.GetRunRequest_DEFAULT
	response, err := server.GetRun(ctx, &apiv2.GetRunRequest{
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
	defaultView := apiv2.GetRunRequest_DEFAULT
	defaultResponse, err := server.GetRun(ctx, &apiv2.GetRunRequest{
		RunId: createdRun.UUID,
		View:  &defaultView,
	})
	assert.Nil(t, err)
	assert.Equal(t, int32(3), defaultResponse.TaskCount)

	// Test with FULL view
	fullView := apiv2.GetRunRequest_FULL
	fullResponse, err := server.GetRun(ctx, &apiv2.GetRunRequest{
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

	fullView := apiv2.ListRunsRequest_FULL
	defaultView := apiv2.ListRunsRequest_DEFAULT

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
		State:          model.TaskStatus(apiv2.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "child-late",
		ParentTaskUUID: &parentTask.UUID,
		CreatedAtInSec: 25,
		StartedInSec:   25,
		State:          model.TaskStatus(apiv2.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "sibling-early",
		CreatedAtInSec: 10,
		StartedInSec:   10,
		State:          model.TaskStatus(apiv2.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "child-early",
		ParentTaskUUID: &parentTask.UUID,
		CreatedAtInSec: 15,
		StartedInSec:   15,
		State:          model.TaskStatus(apiv2.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)

	fullView := apiv2.GetRunRequest_FULL
	response, err := server.GetRun(context.Background(), &apiv2.GetRunRequest{
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

	var apiParent *apiv2.PipelineTask
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
		State:     model.TaskStatus(apiv2.PipelineTask_RUNNING),
	})
	assert.NoError(t, err)

	execution, err := clients.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
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

	fullView := apiv2.GetRunRequest_FULL
	response, err := server.GetRun(context.Background(), &apiv2.GetRunRequest{
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
				PipelineSpecManifest: model.LargeText(testIRPipeline),
			},
		}
		extraRun, err := manager.CreateRun(ctx, run)
		assert.NoError(t, err)

		_, err = clients.TaskStore().CreateTask(&model.Task{
			Namespace: "ns1",
			RunUUID:   extraRun.UUID,
			Name:      fmt.Sprintf("task-extra-%d", index),
			State:     model.TaskStatus(apiv2.PipelineTask_RUNNING),
		})
		assert.NoError(t, err)
	}

	fullView := apiv2.ListRunsRequest_FULL
	response, err := server.ListRuns(ctx, &apiv2.ListRunsRequest{
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

func TestExitHandlerCompletionGuarantee_RunServer_LiveSynchronizationAndAggregation(t *testing.T) {
	clients, manager, run := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// 1. Sibling Processing Order 1: child1 fails, child2 completes/succeeds
	dag1, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "dag-order-1",
		ScopePath: "root.dag-order-1",
		Type:      model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	pods1, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "c1-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dag1.UUID,
		Name:           "child-1",
		ScopePath:      "root.dag-order-1.child-1",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           pods1,
	})
	require.NoError(t, err)

	pods2, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "c2-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dag1.UUID,
		Name:           "child-2",
		ScopePath:      "root.dag-order-1.child-2",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           pods2,
	})
	require.NoError(t, err)

	// 2. Sibling Processing Order 2: childA completes/succeeds, childB fails
	dag2, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "dag-order-2",
		ScopePath: "root.dag-order-2",
		Type:      model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	podsA, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "ca-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dag2.UUID,
		Name:           "child-a",
		ScopePath:      "root.dag-order-2.child-a",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           podsA,
	})
	require.NoError(t, err)

	podsB, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "cb-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dag2.UUID,
		Name:           "child-b",
		ScopePath:      "root.dag-order-2.child-b",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           podsB,
	})
	require.NoError(t, err)

	// 3. Stranded sibling: sibling-normal finished, sibling-stranded crashed in Argo
	dagStranded, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "dag-stranded",
		ScopePath: "root.dag-stranded",
		Type:      model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dagStranded.UUID,
		Name:           "sibling-normal",
		ScopePath:      "root.dag-stranded.sibling-normal",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_FAILED),
		FinishedInSec:  10,
	})
	require.NoError(t, err)

	podsStranded, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "stranded-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dagStranded.UUID,
		Name:           "sibling-stranded",
		ScopePath:      "root.dag-stranded.sibling-stranded",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           podsStranded,
	})
	require.NoError(t, err)

	// 4. Nested exit handlers / nested DAGs: outer DAG -> inner DAG -> worker
	outerDag, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "outer-dag",
		ScopePath: "root.outer-dag",
		Type:      model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	innerDag, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &outerDag.UUID,
		Name:           "inner-dag",
		ScopePath:      "root.outer-dag.inner-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	podsWorker, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "nested-worker-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &innerDag.UUID,
		Name:           "nested-worker",
		ScopePath:      "root.outer-dag.inner-dag.nested-worker",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           podsWorker,
	})
	require.NoError(t, err)

	// Set up live workflow with actual Argo execution state
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowFailed
	workflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		// Order 1: c1 failed, c2 succeeded
		"c1-pod": {ID: "c1-pod", Name: "c1-pod", DisplayName: "child-1", Phase: v1alpha1.NodeFailed, Message: "error in child 1"},
		"c2-pod": {ID: "c2-pod", Name: "c2-pod", DisplayName: "child-2", Phase: v1alpha1.NodeSucceeded},
		// Order 2: ca succeeded, cb failed
		"ca-pod": {ID: "ca-pod", Name: "ca-pod", DisplayName: "child-a", Phase: v1alpha1.NodeSucceeded},
		"cb-pod": {ID: "cb-pod", Name: "cb-pod", DisplayName: "child-b", Phase: v1alpha1.NodeFailed, Message: "error in child b"},
		// Stranded sibling: stranded pod crashed with exit code 137
		"stranded-pod": {ID: "stranded-pod", Name: "stranded-pod", DisplayName: "sibling-stranded", Phase: v1alpha1.NodeFailed, Message: "OOMKilled exit code 137"},
		// Nested worker crashed
		"nested-worker-pod": {ID: "nested-worker-pod", Name: "nested-worker-pod", DisplayName: "nested-worker", Phase: v1alpha1.NodeFailed, Message: "crashed"},
	}

	// Update live workflow in fake cluster so live workflowClient.Get retrieves it
	_, err = clients.ExecClient().Execution(run.Namespace).Update(context.Background(), workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Now simulate the exit handler starting: it calls RunServer.GetRun with authoritative requirement and FULL view
	authCtx := common.WithAuthoritativeRequirement(ctx)
	fullView := apiv2beta1.GetRunRequest_FULL
	response, err := server.GetRun(authCtx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	tasksByName := make(map[string]*apiv2beta1.PipelineTask)
	for _, task := range response.GetTasks() {
		tasksByName[task.GetName()] = task
	}

	// Verification 1: Order 1 (child1 failed, child2 succeeded) -> dag-order-1 MUST be FAILED
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["child-1"].GetState())
	assert.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, tasksByName["child-2"].GetState())
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["dag-order-1"].GetState(), "dag-order-1 must be aggregated to FAILED")

	// Verification 2: Order 2 (childA succeeded, childB failed) -> dag-order-2 MUST be FAILED
	assert.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, tasksByName["child-a"].GetState())
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["child-b"].GetState())
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["dag-order-2"].GetState(), "dag-order-2 must be aggregated to FAILED")

	// Verification 3: Stranded sibling -> sibling-stranded reconciled to FAILED, dag-stranded aggregated to FAILED
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["sibling-stranded"].GetState(), "stranded sibling must be reconciled to FAILED")
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["dag-stranded"].GetState(), "dag-stranded must be aggregated to FAILED")

	// Verification 4: Nested exit handlers -> inner-dag FAILED, outer-dag FAILED
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["nested-worker"].GetState())
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["inner-dag"].GetState(), "inner-dag must be aggregated to FAILED for inner exit handler")
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, tasksByName["outer-dag"].GetState(), "outer-dag must be aggregated to FAILED for outer exit handler")
}

// TestExitHandlerCompletionGuarantee_FailsClosedWhenSynchronizationFails proves that when live
// workflow synchronization fails for an active, non-terminal run, the exit-handler path (GetRun with FULL view)
// fails closed with an error and cannot return stale, unverified database task states.
// In contrast, ordinary reads (DEFAULT view) succeed normally, preserving standard API behavior.
func TestExitHandlerCompletionGuarantee_FailsClosedWhenSynchronizationFails(t *testing.T) {
	clients, manager, run := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Create running tasks in the DB
	dagTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "dag-failing-sync",
		ScopePath: "root.dag-failing-sync",
		Type:      model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	pods, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "worker-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dagTask.UUID,
		Name:           "worker-task",
		ScopePath:      "root.dag-failing-sync.worker-task",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           pods,
	})
	require.NoError(t, err)

	// Ensure the workflow does NOT exist in the fake cluster (live cluster lookup fails with NotFound)
	_ = clients.ExecClient().Execution(run.Namespace).Delete(context.Background(), run.K8SName, metav1.DeleteOptions{})

	// 1. Exit-handler path calls GetRun with authoritative requirement and FULL view
	// Since run is non-terminal in DB and live workflow synchronization fails (NotFound in k8s),
	// it MUST fail closed and return an error. It must NOT return stale RUNNING tasks.
	authCtx := common.WithAuthoritativeRequirement(ctx)
	fullView := apiv2beta1.GetRunRequest_FULL
	respFull, err := server.GetRun(authCtx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.Error(t, err, "GetRun(FULL) with authoritative requirement must fail closed when live synchronization fails")
	assert.Nil(t, respFull)
	assert.Contains(t, err.Error(), "Failed to synchronize run")

	// 2. Ordinary read calls GetRun with FULL view without authoritative requirement
	// Ordinary hydrated reads remain read-only from database storage and succeed normally
	respOrdinaryFull, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err, "Ordinary GetRun(FULL) must succeed without error from database")
	require.NotNil(t, respOrdinaryFull)
	assert.Equal(t, run.UUID, respOrdinaryFull.RunId)

	// 3. Ordinary read calls GetRun with DEFAULT view (no task hydration)
	// Normal API behavior is preserved and succeeds without error
	defaultView := apiv2beta1.GetRunRequest_DEFAULT
	respDefault, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &defaultView,
	})
	require.NoError(t, err, "GetRun(DEFAULT) must succeed for ordinary reads")
	require.NotNil(t, respDefault)
	assert.Equal(t, run.UUID, respDefault.RunId)
	assert.Nil(t, respDefault.Tasks, "DEFAULT view omits tasks")
}

// TestExitHandler_PersistenceOrderingRace_CannotSucceedPrematurely verifies that:
//  1. If an exit handler runs while Argo's deferred workflow persistence has not yet recorded the
//     terminal main-DAG status, the final-status read cannot succeed prematurely.
//  2. Once Argo's authoritative terminal state becomes available in Kubernetes, the final status
//     read through server.GetRun and resolver.ResolveInputs succeeds authoritatively.
func TestExitHandler_PersistenceOrderingRace_CannotSucceedPrematurely(t *testing.T) {
	clients, manager, run := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// Create root task, main DAG task, and worker task
	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "root",
		ScopePath: "root",
		Type:      model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	dagTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	podsWorker, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{Name: "worker-pod", Type: apiv2beta1.PipelineTask_EXECUTOR}})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dagTask.UUID,
		Name:           "worker",
		ScopePath:      "root.main-dag.worker",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           podsWorker,
	})
	require.NoError(t, err)

	// Step 1: Initial Stale / Non-terminal Workflow observation in Kubernetes.
	// Argo controller has started the exit hook pod, but has NOT yet persisted the terminal status to Workflow CR.
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowRunning
	workflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		"worker-pod": {ID: "worker-pod", Name: "worker-pod", DisplayName: "worker", Phase: v1alpha1.NodeRunning},
	}
	_, err = clients.ExecClient().Execution(run.Namespace).Update(context.Background(), workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Exit handler starts and calls GetRun(FULL) to resolve its PipelineTaskFinalStatus
	fullView := apiv2beta1.GetRunRequest_FULL
	initialResp, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)
	require.NotNil(t, initialResp)

	// Setup options for resolving PipelineTaskFinalStatus of main-dag
	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:            initialResp,
		RunName:        run.UUID,
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	// Verify that resolving inputs prematurely FAILS CLOSED
	_, _, err = resolver.ResolveInputs(ctx, driverOpts)
	require.Error(t, err, "Final-status resolution must not succeed prematurely while producer task is RUNNING")
	assert.Contains(t, err.Error(), "producer task state RUNNING is not terminal")

	// Step 2: Authoritative terminal state becomes available later in Kubernetes
	// Argo controller completes deferred persistence:
	workflow.Status.Phase = v1alpha1.WorkflowFailed
	workflow.Status.Nodes["worker-pod"] = v1alpha1.NodeStatus{
		ID:          "worker-pod",
		Name:        "worker-pod",
		DisplayName: "worker",
		Phase:       v1alpha1.NodeFailed,
		Message:     "Worker OOMKilled exit code 137",
	}
	_, err = clients.ExecClient().Execution(run.Namespace).Update(context.Background(), workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Exit handler calls GetRun(FULL) again with terminal state now available
	authoritativeResp, err := server.GetRun(common.WithAuthoritativeRequirement(ctx), &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)
	require.NotNil(t, authoritativeResp)

	// Update driver options with the authoritatively hydrated run
	driverOpts.Run = authoritativeResp

	// Verify that resolving inputs now succeeds with authoritative final status
	inputs, _, err := resolver.ResolveInputs(ctx, driverOpts)
	require.NoError(t, err, "Final-status resolution must succeed once authoritative terminal state is available")
	require.NotNil(t, inputs)
	require.Len(t, inputs.Parameters, 1)

	param := inputs.Parameters[0]
	assert.Equal(t, "status", param.Key)
	assert.Equal(t, "FAILED", param.ParameterIO.Value.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "main-dag", param.ParameterIO.Value.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
}

// TestExitHandler_AutomaticCompletionHandoff_ResolvesWhenTerminalStateBecomesAvailable proves that:
// When an exit handler pod starts while the Workflow CR in Kubernetes is initially non-terminal (RUNNING),
// the exit handler execution path (resolver.ResolveInputs) automatically waits/retries with its defined deadline,
// establishes authoritative final status when Argo persists the terminal node status in Kubernetes,
// and returns the resolved status without requiring manual intervention or a manually issued second request.
func TestExitHandler_AutomaticCompletionHandoff_ResolvesWhenTerminalStateBecomesAvailable(t *testing.T) {
	clients, manager, run := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "root",
		ScopePath: "root",
		Type:      model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	dagTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	podsWorker := []interface{}{
		map[string]interface{}{"name": "worker-driver", "type": apiv2beta1.PipelineTask_DRIVER.String()},
		map[string]interface{}{"name": "worker-pod", "type": apiv2beta1.PipelineTask_EXECUTOR.String()},
	}
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dagTask.UUID,
		Name:           "worker",
		ScopePath:      "root.main-dag.worker",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods:           podsWorker,
	})
	require.NoError(t, err)

	// Step 1: Initial Workflow observation in Kubernetes is nonterminal (RUNNING).
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowRunning
	workflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		"worker-pod": {ID: "worker-pod", Name: "worker-pod", DisplayName: "worker", Phase: v1alpha1.NodeRunning},
	}
	_, err = clients.ExecClient().Execution(run.Namespace).Update(context.Background(), workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	fullView := apiv2beta1.GetRunRequest_FULL
	initialResp, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)
	require.NotNil(t, initialResp)

	// Exit handler driver options configured with live RunRefresher pointing to server.GetRun
	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:                     initialResp,
		RunName:                 run.UUID,
		IterationIndex:          -1,
		FinalStatusPollInterval: 20 * time.Millisecond,
		FinalStatusTimeout:      2 * time.Second,
		RunRefresher: func(refreshCtx context.Context) (*apiv2beta1.Run, error) {
			return server.GetRun(refreshCtx, &apiv2beta1.GetRunRequest{
				RunId: run.UUID,
				View:  &fullView,
			})
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	// Step 2: In a concurrent goroutine simulating Argo's deferred persistence to etcd,
	// the terminal state becomes available in Kubernetes after a short latency window (50ms).
	go func() {
		time.Sleep(50 * time.Millisecond)
		workflow.Status.Phase = v1alpha1.WorkflowFailed
		workflow.Status.Nodes["worker-pod"] = v1alpha1.NodeStatus{
			ID:          "worker-pod",
			Name:        "worker-pod",
			DisplayName: "worker",
			Phase:       v1alpha1.NodeFailed,
			Message:     "Worker OOMKilled exit code 137",
		}
		_, _ = clients.ExecClient().Execution(run.Namespace).Update(context.Background(), workflow, metav1.UpdateOptions{})
	}()

	// The exit handler executes ResolveInputs. It automatically waits and retries until
	// the authoritative terminal status is synchronized, without manual intervention.
	inputs, _, err := resolver.ResolveInputs(ctx, driverOpts)
	require.NoError(t, err, "ResolveInputs must automatically complete without error when terminal state becomes available")
	require.NotNil(t, inputs)
	require.Len(t, inputs.Parameters, 1)

	param := inputs.Parameters[0]
	assert.Equal(t, "status", param.Key)
	assert.Equal(t, "FAILED", param.ParameterIO.Value.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "main-dag", param.ParameterIO.Value.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
	assert.Equal(t, "Worker OOMKilled exit code 137", param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())
}

// TestExitHandler_CompletionHandoff_TimesOutWhenProducerNeverTerminates proves that:
// When a producer task remains non-terminal and does not reach a terminal state within the defined deadline,
// the exit handler completion handoff times out and returns a clear, bounded timeout error rather than hanging indefinitely.
func TestExitHandler_CompletionHandoff_TimesOutWhenProducerNeverTerminates(t *testing.T) {
	clients, manager, run := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "root",
		ScopePath: "root",
		Type:      model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	dagTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &dagTask.UUID,
		Name:           "worker",
		ScopePath:      "root.main-dag.worker",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Pods: []interface{}{
			map[string]interface{}{"name": "worker-pod", "type": apiv2beta1.PipelineTask_EXECUTOR.String()},
		},
	})
	require.NoError(t, err)

	// Live workflow is running
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowRunning
	workflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		"worker-pod": {ID: "worker-pod", Name: "worker-pod", DisplayName: "worker", Phase: v1alpha1.NodeRunning},
	}
	_, err = clients.ExecClient().Execution(run.Namespace).Update(context.Background(), workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	fullView := apiv2beta1.GetRunRequest_FULL
	initialResp, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)

	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:                     initialResp,
		RunName:                 run.UUID,
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      60 * time.Millisecond,
		RunRefresher: func(refreshCtx context.Context) (*apiv2beta1.Run, error) {
			return server.GetRun(refreshCtx, &apiv2beta1.GetRunRequest{
				RunId: run.UUID,
				View:  &fullView,
			})
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	_, _, err = resolver.ResolveInputs(ctx, driverOpts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out waiting for producer task")
	assert.Contains(t, err.Error(), "RUNNING")
}

// TestExitHandler_CompletionHandoff_FailsImmediatelyOnContextCancellation proves that:
// When the request context is canceled during the completion handoff wait,
// the operation returns immediately with a context cancellation error.
func TestExitHandler_CompletionHandoff_FailsImmediatelyOnContextCancellation(t *testing.T) {
	clients, manager, run := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx, cancel := context.WithCancel(context.Background())
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "root",
		ScopePath: "root",
		Type:      model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	fullView := apiv2beta1.GetRunRequest_FULL
	initialResp, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)

	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:                     initialResp,
		RunName:                 run.UUID,
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      5 * time.Second,
		RunRefresher: func(refreshCtx context.Context) (*apiv2beta1.Run, error) {
			return server.GetRun(refreshCtx, &apiv2beta1.GetRunRequest{
				RunId: run.UUID,
				View:  &fullView,
			})
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	// Cancel the context prior to execution
	cancel()

	_, _, err = resolver.ResolveInputs(ctx, driverOpts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// TestExitHandler_CompletionHandoff_FailsClosedWhenRefresherFails proves that:
// When the API server synchronization fails during the handoff retry,
// the exit handler fails closed immediately and does not return stale success.
func TestExitHandler_CompletionHandoff_FailsClosedWhenRefresherFails(t *testing.T) {
	clients, manager, run := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "root",
		ScopePath: "root",
		Type:      model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
	})
	require.NoError(t, err)

	fullView := apiv2beta1.GetRunRequest_FULL
	initialResp, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)

	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:                     initialResp,
		RunName:                 run.UUID,
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      1 * time.Second,
		RunRefresher: func(refreshCtx context.Context) (*apiv2beta1.Run, error) {
			return nil, fmt.Errorf("kubernetes apiserver connection reset")
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	_, _, err = resolver.ResolveInputs(ctx, driverOpts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kubernetes apiserver connection reset")
}

// TestExitHandler_StalePersistedParentSucceeded_ReconcilesToFailedWhenChildFailed proves that:
// When the database says a producer parent DAG is SUCCEEDED, but a child task is FAILED (and the live Argo executor node is Failed),
// the exit handler resolving final status once must resolve FAILED, never SUCCEEDED, and receive correct error metadata.
func TestExitHandler_StalePersistedParentSucceeded_ReconcilesToFailedWhenChildFailed(t *testing.T) {
	clients, manager, _ := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName: "pipeline-run-stale-parent",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(testIRPipeline),
		},
	})
	require.NoError(t, err)

	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:   "ns1",
		RunUUID:     run.UUID,
		Name:        "root",
		ScopePath:   "root",
		Type:        model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:       model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		Fingerprint: "fp-root",
	})
	require.NoError(t, err)

	// In the database: parent DAG task is falsely/stalely marked SUCCEEDED!
	parentDAGTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      "ns1",
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), // Stale fast path in DB
		Fingerprint:    "fp-main-dag",
		FinishedInSec:  100,
	})
	require.NoError(t, err)

	// Child task is FAILED in DB
	childPods, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{
		Name: "worker-pod",
		Type: apiv2beta1.PipelineTask_EXECUTOR,
	}})
	require.NoError(t, err)

	childErrMsg := "Worker execution failed: container exited with code 1"
	childLM := model.LargeText(childErrMsg)
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          run.UUID,
		ParentTaskUUID:   &parentDAGTask.UUID,
		Name:             "worker",
		ScopePath:        "root.main-dag.worker",
		Type:             model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:            model.TaskStatus(apiv2beta1.PipelineTask_FAILED),
		Fingerprint:      "fp-worker",
		Pods:             childPods,
		FinishedInSec:    100,
		StatusMetadata:   model.JSONData{"message": childErrMsg},
		LifecycleMessage: &childLM,
	})
	require.NoError(t, err)

	// Update live workflow in Kubernetes
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(ctx, run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowFailed
	workflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		"worker-pod": {
			ID:          "worker-pod",
			Name:        "worker-pod",
			DisplayName: "worker",
			Phase:       v1alpha1.NodeFailed,
			Message:     childErrMsg,
		},
	}
	_, err = clients.ExecClient().Execution(run.Namespace).Update(ctx, workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	// The exit handler fetches the run authoritatively via GetRun(view=FULL)
	fullView := apiv2beta1.GetRunRequest_FULL
	authoritativeResp, err := server.GetRun(common.WithAuthoritativeRequirement(ctx), &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)
	require.NotNil(t, authoritativeResp)

	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:            authoritativeResp,
		RunName:        run.UUID,
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	// Exit handler resolves inputs once
	inputs, _, err := resolver.ResolveInputs(ctx, driverOpts)
	require.NoError(t, err)
	require.NotNil(t, inputs)
	require.Len(t, inputs.Parameters, 1)

	param := inputs.Parameters[0]
	assert.Equal(t, "status", param.Key)
	// Must resolve FAILED, NEVER SUCCEEDED!
	assert.Equal(t, "FAILED", param.ParameterIO.Value.GetStructValue().GetFields()["state"].GetStringValue(), "Exit handler must resolve FAILED, never SUCCEEDED when child failed")
	assert.Equal(t, "main-dag", param.ParameterIO.Value.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
	assert.Equal(t, childErrMsg, param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())

	// Verify database state: parent DAG task must be reconciled to FAILED in DB as well
	reconciledParentInDB, err := clients.TaskStore().GetTask(parentDAGTask.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), reconciledParentInDB.State, "Parent DAG in database must be reconciled to FAILED")
	assert.NotNil(t, reconciledParentInDB.StatusMetadata)
	assert.Equal(t, childErrMsg, reconciledParentInDB.StatusMetadata["message"])
}

// TestExitHandler_StalePersistedParentSucceeded_ReconcilesToFailedWhenAllTasksTerminal proves that:
// When all persisted tasks in the database are already terminal (no tasks in RUNNING state),
// but the database has an inconsistent state where a parent DAG is SUCCEEDED and a child task is FAILED,
// live workflow synchronization does not skip DAG reconciliation, and the exit handler resolves FAILED.
func TestExitHandler_StalePersistedParentSucceeded_ReconcilesToFailedWhenAllTasksTerminal(t *testing.T) {
	clients, manager, _ := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	// All tasks are terminal; run is in terminal state as well
	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName: "pipeline-run-terminal",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(testIRPipeline),
		},
	})
	require.NoError(t, err)

	run.State = model.RuntimeStateFailed
	run.Conditions = string(model.RuntimeStateFailed)
	err = clients.RunStore().UpdateRun(run)
	require.NoError(t, err)

	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:     "ns1",
		RunUUID:       run.UUID,
		Name:          "root",
		ScopePath:     "root",
		Type:          model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:         model.TaskStatus(apiv2beta1.PipelineTask_FAILED), // terminal
		Fingerprint:   "fp-root",
		FinishedInSec: 100,
	})
	require.NoError(t, err)

	// Parent DAG task is terminal SUCCEEDED in DB (inconsistent with child)
	parentDAGTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      "ns1",
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), // terminal SUCCEEDED
		Fingerprint:    "fp-main-dag",
		FinishedInSec:  100,
	})
	require.NoError(t, err)

	// Child task is terminal FAILED in DB
	childErrMsg := "Child task failed: test failure"
	childLM := model.LargeText(childErrMsg)
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          run.UUID,
		ParentTaskUUID:   &parentDAGTask.UUID,
		Name:             "worker",
		ScopePath:        "root.main-dag.worker",
		Type:             model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:            model.TaskStatus(apiv2beta1.PipelineTask_FAILED), // terminal FAILED
		Fingerprint:      "fp-worker",
		FinishedInSec:    100,
		StatusMetadata:   model.JSONData{"message": childErrMsg},
		LifecycleMessage: &childLM,
	})
	require.NoError(t, err)

	// Live Argo workflow in Kubernetes
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(ctx, run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowFailed
	_, err = clients.ExecClient().Execution(run.Namespace).Update(ctx, workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Fetch run via GetRun(view=FULL) with authoritative requirement
	fullView := apiv2beta1.GetRunRequest_FULL
	authoritativeResp, err := server.GetRun(common.WithAuthoritativeRequirement(ctx), &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)
	require.NotNil(t, authoritativeResp)

	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:            authoritativeResp,
		RunName:        run.UUID,
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	// Exit handler resolves inputs once
	inputs, _, err := resolver.ResolveInputs(ctx, driverOpts)
	require.NoError(t, err)
	require.NotNil(t, inputs)
	require.Len(t, inputs.Parameters, 1)

	param := inputs.Parameters[0]
	assert.Equal(t, "status", param.Key)
	assert.Equal(t, "FAILED", param.ParameterIO.Value.GetStructValue().GetFields()["state"].GetStringValue(), "Must reconcile to FAILED even when all persisted tasks were terminal")
	assert.Equal(t, float64(codes.Unknown), param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
	assert.Equal(t, childErrMsg, param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())

	// Verify database state reconciled
	reconciledParentInDB, err := clients.TaskStore().GetTask(parentDAGTask.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), reconciledParentInDB.State)
}

// TestExitHandler_StalePersistedChildSucceeded_ReconcilesToFailedWhenLiveArgoNodeFailed proves that:
// When both parent DAG and child task are marked SUCCEEDED in the database, but the authoritative Argo executor node
// has Failed, GetRunWithHydration must reconcile both the child and parent DAG to FAILED, and the exit handler resolves FAILED.
func TestExitHandler_StalePersistedChildSucceeded_ReconcilesToFailedWhenLiveArgoNodeFailed(t *testing.T) {
	clients, manager, _ := initWithRunAndTasks(t)
	defer clients.Close()
	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName: "pipeline-run-argo-fail",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(testIRPipeline),
		},
	})
	require.NoError(t, err)

	// Set initial database run state to terminal SUCCEEDED
	run.State = model.RuntimeStateSucceeded
	run.Conditions = string(model.RuntimeStateSucceeded)
	err = clients.RunStore().UpdateRun(run)
	require.NoError(t, err)

	// Initial database root task is terminal SUCCEEDED
	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:     "ns1",
		RunUUID:       run.UUID,
		Name:          "root",
		ScopePath:     "root",
		Type:          model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:         model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), // terminal SUCCEEDED
		Fingerprint:   "fp-root",
		FinishedInSec: 100,
	})
	require.NoError(t, err)

	// Initial database parent DAG task is terminal SUCCEEDED (no DB inconsistency with children)
	parentDAGTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      "ns1",
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), // terminal SUCCEEDED
		Fingerprint:    "fp-main-dag",
		FinishedInSec:  100,
	})
	require.NoError(t, err)

	childPods, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{{
		Name: "worker-pod",
		Type: apiv2beta1.PipelineTask_EXECUTOR,
	}})
	require.NoError(t, err)

	// Initial database child task is terminal SUCCEEDED (all tasks in DB are terminal SUCCEEDED)
	childTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      "ns1",
		RunUUID:        run.UUID,
		ParentTaskUUID: &parentDAGTask.UUID,
		Name:           "worker",
		ScopePath:      "root.main-dag.worker",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), // terminal SUCCEEDED
		Fingerprint:    "fp-worker",
		Pods:           childPods,
		FinishedInSec:  100,
	})
	require.NoError(t, err)

	// Live Argo workflow: executor node actually Failed (e.g. OOMKilled)
	argoFailMsg := "OOMKilled: container exceeded memory limit"
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(ctx, run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.Status.Phase = v1alpha1.WorkflowFailed
	workflow.Status.Nodes = map[string]v1alpha1.NodeStatus{
		"worker-pod": {
			ID:          "worker-pod",
			Name:        "worker-pod",
			DisplayName: "worker",
			Phase:       v1alpha1.NodeFailed,
			Message:     argoFailMsg,
		},
	}
	_, err = clients.ExecClient().Execution(run.Namespace).Update(ctx, workflow, metav1.UpdateOptions{})
	require.NoError(t, err)

	fullView := apiv2beta1.GetRunRequest_FULL
	authoritativeResp, err := server.GetRun(common.WithAuthoritativeRequirement(ctx), &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err)
	require.NotNil(t, authoritativeResp)

	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:            authoritativeResp,
		RunName:        run.UUID,
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
	}

	inputs, _, err := resolver.ResolveInputs(ctx, driverOpts)
	require.NoError(t, err)
	require.NotNil(t, inputs)

	param := inputs.Parameters[0]
	assert.Equal(t, "status", param.Key)
	assert.Equal(t, "FAILED", param.ParameterIO.Value.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, argoFailMsg, param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())

	// Verify child task reconciled in DB
	reconciledChildInDB, err := clients.TaskStore().GetTask(childTask.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), reconciledChildInDB.State)

	// Verify parent DAG reconciled in DB
	reconciledParentInDB, err := clients.TaskStore().GetTask(parentDAGTask.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), reconciledParentInDB.State)

	// Verify run state reconciled in DB
	reconciledRunInDB, err := clients.RunStore().GetRun(run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateFailed, reconciledRunInDB.State)
}

// TestExitHandler_StalePersistedAllSucceeded_WorkflowNotFound_FailsClosedWithoutAuthoritativeRecord proves that:
// 1. When all tasks (run, root, parent DAG, child) are persisted as SUCCEEDED in the DB and the Workflow is NotFound in Kubernetes:
//   - Ordinary historical GetRun requests succeed and return the persisted database records (preserving historical reads after GC).
//   - Exit-handler final-status resolution fails closed with a clear error and never returns an unverified SUCCEEDED status.
//     2. When an independently authoritative, persisted completion record (WorkflowRuntimeManifest with PersistedFinalState) exists,
//     the final-status resolution succeeds authoritatively even if the live Workflow was garbage collected.
func TestExitHandler_StalePersistedAllSucceeded_WorkflowNotFound_FailsClosedWithoutAuthoritativeRecord(t *testing.T) {
	clients, manager, _ := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()
	if common.IsMultiUserMode() {
		md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
		ctx = metadata.NewIncomingContext(context.Background(), md)
	}

	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName: "pipeline-run-all-succeeded",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(testIRPipeline),
		},
	})
	require.NoError(t, err)

	// Persist run as SUCCEEDED in DB with NO verified persisted completion manifest
	run.State = model.RuntimeStateSucceeded
	run.Conditions = string(model.RuntimeStateSucceeded)
	run.WorkflowRuntimeManifest = ""
	err = clients.RunStore().UpdateRun(run)
	require.NoError(t, err)

	// Persist root task as SUCCEEDED in DB
	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "root",
		ScopePath: "root",
		Type:      model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
	})
	require.NoError(t, err)

	// Persist parent DAG task as SUCCEEDED in DB
	parentDAGTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
	})
	require.NoError(t, err)

	// Persist child task as SUCCEEDED in DB
	podsChild, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{
		{Name: "worker-driver", Type: apiv2beta1.PipelineTask_DRIVER},
		{Name: "worker-pod", Type: apiv2beta1.PipelineTask_EXECUTOR},
	})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &parentDAGTask.UUID,
		Name:           "worker",
		ScopePath:      "root.main-dag.worker",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
		Pods:           podsChild,
	})
	require.NoError(t, err)

	// Delete live Workflow from Kubernetes so live lookup returns NotFound
	_ = clients.ExecClient().Execution(run.Namespace).Delete(context.Background(), run.K8SName, metav1.DeleteOptions{})

	fullView := apiv2beta1.GetRunRequest_FULL

	// Step 1: Ordinary historical read (without authoritative requirement).
	// Must preserve normal database fallback after workflow GC for terminal runs.
	historicalResp, err := server.GetRun(ctx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err, "Ordinary historical GetRun must succeed using database fallback")
	require.NotNil(t, historicalResp)
	assert.Equal(t, apiv2beta1.RuntimeState_SUCCEEDED, historicalResp.State)
	assert.Len(t, historicalResp.Tasks, 3)

	// Step 2: Exit-handler final-status resolution request with authoritative requirement.
	// Since the live workflow is NotFound and no verified persisted completion record exists,
	// it MUST fail closed and NEVER return an unverified SUCCEEDED status.
	authCtx := common.WithAuthoritativeRequirement(ctx)
	_, err = server.GetRun(authCtx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.Error(t, err, "Authoritative GetRun must fail closed when workflow is NotFound and no verified completion record exists")
	assert.Contains(t, err.Error(), "cannot authoritatively verify final status")

	// Verify that resolver.ResolveInputs also fails closed when RunRefresher encounters the missing workflow
	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:            historicalResp,
		RunName:        run.UUID,
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
		RunRefresher: func(refreshCtx context.Context) (*apiv2beta1.Run, error) {
			return server.GetRun(refreshCtx, &apiv2beta1.GetRunRequest{
				RunId: run.UUID,
				View:  &fullView,
			})
		},
	}

	_, _, err = resolver.ResolveInputs(authCtx, driverOpts)
	require.Error(t, err, "Exit-handler ResolveInputs must fail closed without authoritative evidence")
	assert.Contains(t, err.Error(), "cannot authoritatively verify final status")

	// Step 3: When an independently authoritative, persisted completion record exists in WorkflowRuntimeManifest,
	// exit-handler final-status resolution must SUCCEED authoritatively even though the live Workflow is NotFound.
	terminalWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: run.Namespace,
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               run.UUID,
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase: v1alpha1.WorkflowSucceeded,
			Nodes: map[string]v1alpha1.NodeStatus{
				"worker-pod": {
					ID:          "worker-pod",
					Name:        "worker-pod",
					DisplayName: "worker",
					Phase:       v1alpha1.NodeSucceeded,
				},
			},
		},
	})
	run.WorkflowRuntimeManifest = model.LargeText(terminalWorkflow.ToStringForStore())
	err = clients.RunStore().UpdateRun(run)
	require.NoError(t, err)

	authoritativeResp, err := server.GetRun(authCtx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err, "Authoritative GetRun must succeed when verified persisted completion record exists")
	require.NotNil(t, authoritativeResp)
	assert.Equal(t, apiv2beta1.RuntimeState_SUCCEEDED, authoritativeResp.State)

	driverOpts.Run = authoritativeResp
	inputs, _, err := resolver.ResolveInputs(authCtx, driverOpts)
	require.NoError(t, err, "Exit-handler ResolveInputs must succeed with authoritative evidence")
	require.NotNil(t, inputs)
	param := inputs.Parameters[0]
	assert.Equal(t, "status", param.Key)
	assert.Equal(t, "SUCCEEDED", param.ParameterIO.Value.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, float64(codes.OK), param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
}

// TestExitHandler_AuthoritativeRequirement_SurvivesGRPCTransport verifies that
// WithAuthoritativeRequirement(ctx) genuinely survives the driver's gRPC boundary over the wire:
// 1. Outgoing gRPC metadata is properly attached by the client.
// 2. Incoming gRPC metadata is received by the server handler across a real gRPC connection.
// 3. Authoritative verification fails closed over gRPC when live workflow is missing.
// 4. Authoritative verification succeeds over gRPC when an authoritative persisted manifest is present.
func TestExitHandler_AuthoritativeRequirement_SurvivesGRPCTransport(t *testing.T) {
	clients, manager, _ := initWithRunAndTasks(t)
	defer clients.Close()

	server := createRunServer(manager)
	ctx := context.Background()

	run, err := manager.CreateRun(ctx, &model.Run{
		DisplayName: "grpc-authoritative-run",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(testIRPipeline),
		},
	})
	require.NoError(t, err)

	run.State = model.RuntimeStateSucceeded
	run.Conditions = string(model.RuntimeStateSucceeded)
	run.WorkflowRuntimeManifest = ""
	err = clients.RunStore().UpdateRun(run)
	require.NoError(t, err)

	rootTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace: run.Namespace,
		RunUUID:   run.UUID,
		Name:      "root",
		ScopePath: "root",
		Type:      model.TaskType(apiv2beta1.PipelineTask_ROOT),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
	})
	require.NoError(t, err)

	parentDAGTask, err := clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &rootTask.UUID,
		Name:           "main-dag",
		ScopePath:      "root.main-dag",
		Type:           model.TaskType(apiv2beta1.PipelineTask_DAG),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
	})
	require.NoError(t, err)

	podsChild, _ := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskPod{
		{Name: "worker-driver", Type: apiv2beta1.PipelineTask_DRIVER},
		{Name: "worker-pod", Type: apiv2beta1.PipelineTask_EXECUTOR},
	})
	_, err = clients.TaskStore().CreateTask(&model.Task{
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		ParentTaskUUID: &parentDAGTask.UUID,
		Name:           "worker",
		ScopePath:      "root.main-dag.worker",
		Type:           model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:          model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
		Pods:           podsChild,
	})
	require.NoError(t, err)

	// Delete live Workflow so it is NotFound in Kubernetes
	_ = clients.ExecClient().Execution(run.Namespace).Delete(context.Background(), run.K8SName, metav1.DeleteOptions{})

	// Spin up in-memory gRPC server and dial with real gRPC client
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	apiv2beta1.RegisterRunServiceServer(grpcServer, server)
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	defer grpcServer.Stop()

	conn, err := grpc.NewClient(
		"passthrough:///test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	require.NoError(t, err)
	defer conn.Close()

	grpcClient := apiv2beta1.NewRunServiceClient(conn)
	fullView := apiv2beta1.GetRunRequest_FULL

	// 1. Real gRPC call without authoritative metadata: historical read succeeds
	histResp, err := grpcClient.GetRun(context.Background(), &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err, "Historical GetRun over real gRPC must succeed")
	assert.Equal(t, apiv2beta1.RuntimeState_SUCCEEDED, histResp.State)
	assert.Len(t, histResp.Tasks, 3)

	// 2. Real gRPC call with WithAuthoritativeRequirement: must fail closed across gRPC
	authCtx := common.WithAuthoritativeRequirement(context.Background())
	_, err = grpcClient.GetRun(authCtx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.Error(t, err, "Authoritative GetRun over real gRPC must fail closed without authoritative record")
	assert.Contains(t, err.Error(), "cannot authoritatively verify final status")

	// 3. Now persist an independently authoritative completion record
	terminalWorkflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      run.K8SName,
			Namespace: run.Namespace,
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               run.UUID,
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase: v1alpha1.WorkflowSucceeded,
			Nodes: map[string]v1alpha1.NodeStatus{
				"worker-pod": {
					ID:          "worker-pod",
					Name:        "worker-pod",
					DisplayName: "worker",
					Phase:       v1alpha1.NodeSucceeded,
				},
			},
		},
	})
	run.WorkflowRuntimeManifest = model.LargeText(terminalWorkflow.ToStringForStore())
	err = clients.RunStore().UpdateRun(run)
	require.NoError(t, err)

	// 4. Real gRPC call with WithAuthoritativeRequirement succeeds when authoritative record exists
	authResp, err := grpcClient.GetRun(authCtx, &apiv2beta1.GetRunRequest{
		RunId: run.UUID,
		View:  &fullView,
	})
	require.NoError(t, err, "Authoritative GetRun over real gRPC must succeed with verified record")
	assert.Equal(t, apiv2beta1.RuntimeState_SUCCEEDED, authResp.State)

	// 5. Verify through driver resolver using real gRPC RunRefresher
	driverOpts := drivercommon.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: rootTask.UUID,
			Name:   "root",
		},
		Run:            authResp,
		RunName:        run.UUID,
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"status": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
							TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
								ProducerTask: "main-dag",
							},
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {
						ParameterType: pipelinespec.ParameterType_STRUCT,
					},
				},
			},
		},
		RunRefresher: func(refreshCtx context.Context) (*apiv2beta1.Run, error) {
			return grpcClient.GetRun(refreshCtx, &apiv2beta1.GetRunRequest{
				RunId: run.UUID,
				View:  &fullView,
			})
		},
	}

	inputs, _, err := resolver.ResolveInputs(authCtx, driverOpts)
	require.NoError(t, err, "Exit-handler ResolveInputs must succeed over real gRPC with authoritative evidence")
	require.NotNil(t, inputs)
	param := inputs.Parameters[0]
	assert.Equal(t, "status", param.Key)
	assert.Equal(t, "SUCCEEDED", param.ParameterIO.Value.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, float64(codes.OK), param.ParameterIO.Value.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
}
