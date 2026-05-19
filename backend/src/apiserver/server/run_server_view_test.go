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

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
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
			State:     model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING),
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
