// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	runid1 = "123e4567-e89b-12d3-a456-426655440001"
)

func createArtifactServer(resourceManager *resource.ResourceManager) *ArtifactServer {
	return &ArtifactServer{resourceManager: resourceManager}
}

// ctxWithUser returns a context with a fake user identity header so that
// authorization in multi-user mode passes in tests.
func ctxWithUser() context.Context {
	header := common.GetKubeflowUserIDHeader()
	prefix := common.GetKubeflowUserIDPrefix()
	// Typical header value is like: "accounts.google.com:alice@example.com"
	val := prefix + "test-user@example.com"
	md := metadata.New(map[string]string{header: val})
	return metadata.NewIncomingContext(context.Background(), md)
}

func strPTR(s string) *string {
	return &s
}

func TestArtifactServer_CreateArtifact_MultiUserCreateAndGet_Succeeds(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create run
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "test-run",
		DisplayName:  "test-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)

	// Create task for the run
	task, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "test-task",
		State:     1,
	})
	assert.NoError(t, err)

	req := &apiv2beta1.CreateArtifactRequest{
		RunId:       runid1,
		TaskId:      task.UUID,
		ProducerKey: "producer-key",
		Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
		Artifact: &apiv2beta1.Artifact{
			Namespace:   "ns1",
			Type:        apiv2beta1.Artifact_Model,
			Uri:         strPTR("gs://b/f"),
			Name:        "a1",
			Description: "desc1",
		}}
	created, err := s.CreateArtifact(ctxWithUser(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.GetArtifactId())
	assert.Equal(t, "ns1", created.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Model, created.GetType())
	assert.Equal(t, "gs://b/f", created.GetUri())
	assert.Equal(t, "a1", created.GetName())
	assert.Equal(t, "desc1", created.GetDescription())

	// Creating an artifact should create an artifact task
	// Fetch the artifact task
	artifactTasks, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
		TaskIds:  []string{task.UUID},
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), artifactTasks.GetTotalSize())
	assert.Equal(t, 1, len(artifactTasks.GetArtifactTasks()))

	at := artifactTasks.GetArtifactTasks()[0]
	assert.Equal(t, created.GetArtifactId(), at.GetArtifactId())
	assert.Equal(t, task.UUID, at.GetTaskId())
	assert.Equal(t, apiv2beta1.IOType_OUTPUT, at.GetType())
	assert.NotNil(t, at.GetProducer())
	assert.Equal(t, task.Name, at.GetProducer().GetTaskName())
	assert.Equal(t, "producer-key", at.GetKey())

}

func TestArtifactServer_CreateArtifact_WithIterationIndex(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create run
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "iteration-run",
		DisplayName:  "iteration-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)

	// Create task for the run
	task, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "iteration-task",
		State:     1,
	})
	assert.NoError(t, err)

	// Create artifact with iteration_index
	iterationIndex := int64(5)
	req := &apiv2beta1.CreateArtifactRequest{
		RunId:          runid1,
		TaskId:         task.UUID,
		ProducerKey:    "output-artifact",
		IterationIndex: &iterationIndex,
		Type:           apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
		Artifact: &apiv2beta1.Artifact{
			Namespace:   "ns1",
			Type:        apiv2beta1.Artifact_Dataset,
			Uri:         strPTR("gs://bucket/iteration-5/data"),
			Name:        "iteration-dataset",
			Description: "Dataset from iteration 5",
		}}
	created, err := s.CreateArtifact(ctxWithUser(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.GetArtifactId())
	assert.Equal(t, "ns1", created.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Dataset, created.GetType())
	assert.Equal(t, "iteration-dataset", created.GetName())

	// Verify the artifact task was created with iteration in producer
	artifactTasks, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
		TaskIds:  []string{task.UUID},
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), artifactTasks.GetTotalSize())
	assert.Equal(t, 1, len(artifactTasks.GetArtifactTasks()))

	at := artifactTasks.GetArtifactTasks()[0]
	assert.Equal(t, created.GetArtifactId(), at.GetArtifactId())
	assert.Equal(t, task.UUID, at.GetTaskId())
	assert.Equal(t, apiv2beta1.IOType_OUTPUT, at.GetType())
	assert.Equal(t, "output-artifact", at.GetKey())
	assert.NotNil(t, at.GetProducer())
	assert.Equal(t, task.Name, at.GetProducer().GetTaskName())
	// Verify iteration was set
	assert.NotNil(t, at.GetProducer().Iteration)
	assert.Equal(t, int64(5), *at.GetProducer().Iteration)
}

func TestArtifactServer_ListArtifacts_HappyPath(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create required run and task for artifact creation
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "list-run",
		DisplayName:  "list-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails:   model.RunDetails{CreatedAtInSec: 1, ScheduledAtInSec: 1, State: model.RuntimeStateRunning},
	})
	assert.NoError(t, err)
	listTask, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "t-list",
		State:     1,
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifact(ctxWithUser(),
		&apiv2beta1.CreateArtifactRequest{
			RunId:       runid1,
			TaskId:      listTask.UUID,
			ProducerKey: "producer-key",
			Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
			Artifact: &apiv2beta1.Artifact{
				Namespace:   "ns1",
				Type:        apiv2beta1.Artifact_Model,
				Uri:         strPTR("gs://b/f"),
				Name:        "a1",
				Description: "desc-list",
			},
		},
	)
	require.NoError(t, err)
	listResp, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{
		Namespace: "ns1",
		PageSize:  10,
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, int(listResp.GetTotalSize()), 1)
	assert.GreaterOrEqual(t, len(listResp.GetArtifacts()), 1)
}

func TestArtifactServer_GetArtifact_Errors(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Missing ID
	_, err := s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{ArtifactId: ""})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	// Non-existent
	_, err = s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{ArtifactId: "does-not-exist"})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_Authorization_MultiUser(t *testing.T) {
	// Turn on MU mode by setting viper flag
	// Note: IsMultiUserMode() reads from viper, so configure it here
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	// In MU, Create should preserve namespace; and List with empty namespace should fail
	s := createArtifactServer(resourceManager)

	// By default FakeResourceManager authorizes everything in MU, unless namespace is empty
	// ListArtifacts with empty namespace should fail in MU
	_, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{Namespace: ""})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_SingleUserNamespaceEmpty(t *testing.T) {
	// Ensure single-user mode
	viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Even if request carries a namespace, in single-user mode it should be cleared/empty in stored artifact
	// Create run and task required by CreateArtifact
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         "single-run",
		K8SName:      "single-run",
		DisplayName:  "single-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails:   model.RunDetails{CreatedAtInSec: 1, ScheduledAtInSec: 1, State: model.RuntimeStateRunning},
	})
	assert.NoError(t, err)
	singleTask, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   "single-run",
		Name:      "t-single",
		State:     1,
	})
	assert.NoError(t, err)
	created, err := s.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		RunId:       "single-run",
		TaskId:      singleTask.UUID,
		ProducerKey: "producer-key",
		Type:        apiv2beta1.IOType_TASK_OUTPUT_INPUT,
		Artifact: &apiv2beta1.Artifact{
			Namespace:   "ns1",
			Type:        apiv2beta1.Artifact_Artifact,
			Uri:         strPTR("u"),
			Name:        "a",
			Description: "single-desc",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "", created.GetNamespace())

	// Get artifact and verify it matches
	fetched, err := s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{
		ArtifactId: created.GetArtifactId(),
	})
	assert.NoError(t, err)
	assert.Equal(t, "", fetched.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Artifact, fetched.GetType())
	assert.Equal(t, "u", fetched.GetUri())
	assert.Equal(t, "a", fetched.GetName())
	assert.Equal(t, "single-desc", fetched.GetDescription())
}

const (
	serverRunID1 = "run-1"
	serverRunID2 = "run-2"
)

// seedArtifactTasks sets up two runs, two tasks, two artifacts and three links.
// Returns server, clientManager, entities.
func seedArtifactTasks(t *testing.T) (*ArtifactServer, *resource.FakeClientManager, *model.Task, *model.Task, *model.Artifact, *model.Artifact) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Runs
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         serverRunID1,
		ExperimentId: "",
		K8SName:      "r1",
		DisplayName:  "r1",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)
	_, err = clientManager.RunStore().CreateRun(&model.Run{
		UUID:         serverRunID2,
		ExperimentId: "",
		K8SName:      "r2",
		DisplayName:  "r2",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			State:            model.RuntimeStateRunning,
		},
	})
	require.NoError(t, err)

	// Tasks
	t1, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   serverRunID1,
		Name:      "t1",
		State:     1,
	})
	assert.NoError(t, err)
	t2, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   serverRunID2,
		Name:      "t2",
		State:     1,
	})
	assert.NoError(t, err)

	// Artifacts
	art1, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace:   "ns1",
		Type:        model.ArtifactType(apiv2beta1.Artifact_Artifact),
		URI:         strPTR("u"),
		Name:        "a1",
		Description: "d1",
	})
	assert.NoError(t, err)
	art2, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace:   "ns1",
		Type:        model.ArtifactType(apiv2beta1.Artifact_Artifact),
		URI:         strPTR("u2"),
		Name:        "a2",
		Description: "d2",
	})
	assert.NoError(t, err)

	// Links
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId: art1.UUID,
			TaskId:     t1.UUID,
			RunId:      serverRunID1,
			Type:       apiv2beta1.IOType_COMPONENT_INPUT,
			Producer: &apiv2beta1.IOProducer{
				TaskName: "t1",
			},
			Key: "in1",
		},
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId: art2.UUID,
			TaskId:     t1.UUID,
			RunId:      serverRunID1,
			Type:       apiv2beta1.IOType_OUTPUT,
			Producer: &apiv2beta1.IOProducer{
				TaskName: "t1",
			},
			Key: "out1",
		},
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId: art2.UUID,
			TaskId:     t2.UUID,
			RunId:      serverRunID2,
			Type:       apiv2beta1.IOType_COMPONENT_INPUT,
			Producer: &apiv2beta1.IOProducer{
				TaskName: "t2",
			},
			Key: "in2",
		},
	})
	assert.NoError(t, err)

	return s, clientManager, t1, t2, art1, art2
}

func TestArtifactServer_ListArtifactTasks_FilterByTaskIds(t *testing.T) {
	s, _, t1, _, _, _ := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.GetTotalSize())
	assert.Equal(t, 2, len(resp.GetArtifactTasks()))
	// Ensure key values are present
	keys := []string{resp.GetArtifactTasks()[0].GetKey(), resp.GetArtifactTasks()[1].GetKey()}
	assert.Contains(t, keys, "in1")
	assert.Contains(t, keys, "out1")
	assert.Empty(t, resp.GetNextPageToken())
}

func TestArtifactServer_ListArtifactTasks_FilterByArtifactIds(t *testing.T) {
	s, _, _, _, _, art2 := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{ArtifactIds: []string{art2.UUID}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.GetTotalSize())
	assert.Equal(t, 2, len(resp.GetArtifactTasks()))
	// Ensure key values are as expected for art2 links
	keys := []string{resp.GetArtifactTasks()[0].GetKey(), resp.GetArtifactTasks()[1].GetKey()}
	assert.Contains(t, keys, "out1")
	assert.Contains(t, keys, "in2")
}

func TestArtifactServer_ListArtifactTasks_FilterByRunIds(t *testing.T) {
	s, _, _, t2, _, art2 := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{RunIds: []string{serverRunID2}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.GetTotalSize())
	assert.Equal(t, 1, len(resp.GetArtifactTasks()))
	at := resp.GetArtifactTasks()[0]
	assert.Equal(t, art2.UUID, at.GetArtifactId())
	assert.Equal(t, t2.UUID, at.GetTaskId())
	assert.Equal(t, "in2", at.GetKey())
}

func TestArtifactServer_ListArtifactTasks_ErrorWhenNoFilters(t *testing.T) {
	s, _, _, _, _, _ := seedArtifactTasks(t)
	_, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{PageSize: 2})
	assert.Error(t, err)
}

func TestArtifactServer_ListArtifactTasks_Pagination_TaskIds(t *testing.T) {
	s, _, t1, _, _, _ := seedArtifactTasks(t)
	page1, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageSize: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), page1.GetTotalSize())
	assert.Equal(t, 1, len(page1.GetArtifactTasks()))
	assert.NotEmpty(t, page1.GetNextPageToken())

	page2, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageToken: page1.GetNextPageToken(), PageSize: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), page2.GetTotalSize())
	assert.Equal(t, 1, len(page2.GetArtifactTasks()))
	assert.Empty(t, page2.GetNextPageToken())

	id1 := page1.GetArtifactTasks()[0].GetId()
	id2 := page2.GetArtifactTasks()[0].GetId()
	assert.NotEqual(t, id1, id2)
}

func TestArtifactServer_CreateArtifactTasksBulk_Success(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create a run
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         serverRunID1,
		K8SName:      "bulk-run",
		DisplayName:  "bulk-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)

	// Create tasks
	t1, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   serverRunID1,
		Name:      "task1",
		State:     1,
	})
	assert.NoError(t, err)

	t2, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   serverRunID1,
		Name:      "task2",
		State:     1,
	})
	assert.NoError(t, err)

	// Create artifacts
	art1, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Artifact),
		URI:       strPTR("uri1"),
		Name:      "artifact1",
	})
	assert.NoError(t, err)

	art2, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Artifact),
		URI:       strPTR("uri2"),
		Name:      "artifact2",
	})
	assert.NoError(t, err)

	// Create bulk artifact tasks
	req := &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{
			{
				ArtifactId: art1.UUID,
				TaskId:     t1.UUID,
				RunId:      serverRunID1,
				Type:       apiv2beta1.IOType_COMPONENT_INPUT,
				Producer: &apiv2beta1.IOProducer{
					TaskName: "task1",
				},
				Key: "input1",
			},
			{
				ArtifactId: art2.UUID,
				TaskId:     t1.UUID,
				RunId:      serverRunID1,
				Type:       apiv2beta1.IOType_OUTPUT,
				Producer: &apiv2beta1.IOProducer{
					TaskName: "task1",
				},
				Key: "output1",
			},
			{
				ArtifactId: art2.UUID,
				TaskId:     t2.UUID,
				RunId:      serverRunID1,
				Type:       apiv2beta1.IOType_COMPONENT_INPUT,
				Producer: &apiv2beta1.IOProducer{
					TaskName: "task2",
				},
				Key: "input2",
			},
		},
	}

	resp, err := s.CreateArtifactTasksBulk(ctxWithUser(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, len(resp.GetArtifactTasks()))

	// Verify all artifact tasks were created
	for i, at := range resp.GetArtifactTasks() {
		assert.NotEmpty(t, at.GetId())
		assert.Equal(t, req.ArtifactTasks[i].GetArtifactId(), at.GetArtifactId())
		assert.Equal(t, req.ArtifactTasks[i].GetTaskId(), at.GetTaskId())
		assert.Equal(t, req.ArtifactTasks[i].GetRunId(), at.GetRunId())
		assert.Equal(t, req.ArtifactTasks[i].GetType(), at.GetType())
		assert.Equal(t, req.ArtifactTasks[i].GetKey(), at.GetKey())
		assert.Equal(t, req.ArtifactTasks[i].GetProducer().GetTaskName(), at.GetProducer().GetTaskName())
	}

	// Verify they can be listed
	listResp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
		TaskIds:  []string{t1.UUID},
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), listResp.GetTotalSize())
}

func TestArtifactServer_CreateArtifactTasksBulk_EmptyRequest(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Empty request should fail with validation error
	_, err := s.CreateArtifactTasksBulk(ctxWithUser(), &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{},
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_CreateArtifactTasksBulk_ValidationError(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Request with invalid artifact task (missing required fields)
	req := &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{
			{
				ArtifactId: "art1",
				// Missing TaskId, RunId, Type
			},
		},
	}

	_, err := s.CreateArtifactTasksBulk(ctxWithUser(), req)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_CreateArtifactsBulk_Success(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create run
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "bulk-run",
		DisplayName:  "bulk-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)

	// Create three tasks for the run
	task1, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "task1",
		State:     1,
	})
	assert.NoError(t, err)

	task2, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "task2",
		State:     1,
	})
	assert.NoError(t, err)

	task3, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "task3",
		State:     1,
	})
	assert.NoError(t, err)

	// Create multiple artifacts in bulk
	req := &apiv2beta1.CreateArtifactsBulkRequest{
		Artifacts: []*apiv2beta1.CreateArtifactRequest{
			{
				RunId:       runid1,
				TaskId:      task1.UUID,
				ProducerKey: "output1",
				Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace:   "ns1",
					Type:        apiv2beta1.Artifact_Model,
					Uri:         strPTR("gs://bucket/model1"),
					Name:        "model1",
					Description: "First model",
				},
			},
			{
				RunId:       runid1,
				TaskId:      task2.UUID,
				ProducerKey: "output2",
				Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace:   "ns1",
					Type:        apiv2beta1.Artifact_Dataset,
					Uri:         strPTR("gs://bucket/dataset1"),
					Name:        "dataset1",
					Description: "First dataset",
				},
			},
			{
				RunId:       runid1,
				TaskId:      task3.UUID,
				ProducerKey: "output3",
				Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace:   "ns1",
					Type:        apiv2beta1.Artifact_Metric,
					Uri:         strPTR("gs://bucket/metrics1"),
					Name:        "metrics1",
					Description: "First metrics",
					NumberValue: func() *float64 { v := 0.95; return &v }(),
				},
			},
		},
	}

	resp, err := s.CreateArtifactsBulk(ctxWithUser(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, len(resp.GetArtifacts()))

	// Verify all artifacts were created correctly
	artifact1 := resp.GetArtifacts()[0]
	assert.NotEmpty(t, artifact1.GetArtifactId())
	assert.Equal(t, "ns1", artifact1.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Model, artifact1.GetType())
	assert.Equal(t, "gs://bucket/model1", artifact1.GetUri())
	assert.Equal(t, "model1", artifact1.GetName())
	assert.Equal(t, "First model", artifact1.GetDescription())

	artifact2 := resp.GetArtifacts()[1]
	assert.NotEmpty(t, artifact2.GetArtifactId())
	assert.Equal(t, "ns1", artifact2.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Dataset, artifact2.GetType())
	assert.Equal(t, "gs://bucket/dataset1", artifact2.GetUri())
	assert.Equal(t, "dataset1", artifact2.GetName())

	artifact3 := resp.GetArtifacts()[2]
	assert.NotEmpty(t, artifact3.GetArtifactId())
	assert.Equal(t, "ns1", artifact3.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Metric, artifact3.GetType())
	assert.Equal(t, "gs://bucket/metrics1", artifact3.GetUri())
	assert.Equal(t, "metrics1", artifact3.GetName())

	// Verify artifact-task relationships were created for each
	for i, artifact := range resp.GetArtifacts() {
		artifactTasks, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
			ArtifactIds: []string{artifact.GetArtifactId()},
			PageSize:    10,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(1), artifactTasks.GetTotalSize())
		assert.Equal(t, 1, len(artifactTasks.GetArtifactTasks()))

		at := artifactTasks.GetArtifactTasks()[0]
		assert.Equal(t, artifact.GetArtifactId(), at.GetArtifactId())
		assert.Equal(t, apiv2beta1.IOType_OUTPUT, at.GetType())
		assert.Equal(t, req.Artifacts[i].ProducerKey, at.GetKey())
	}

	// Verify artifacts can be listed
	listResp, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{
		Namespace: "ns1",
		PageSize:  10,
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, int(listResp.GetTotalSize()), 3)
}

func TestArtifactServer_CreateArtifactsBulk_WithIterationIndex(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create run
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "iteration-bulk-run",
		DisplayName:  "iteration-bulk-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)

	// Create task for the run
	task, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "iteration-task",
		State:     1,
	})
	assert.NoError(t, err)

	// Create artifacts with iteration indices
	iter0 := int64(0)
	iter1 := int64(1)
	iter2 := int64(2)

	req := &apiv2beta1.CreateArtifactsBulkRequest{
		Artifacts: []*apiv2beta1.CreateArtifactRequest{
			{
				RunId:          runid1,
				TaskId:         task.UUID,
				ProducerKey:    "iteration-output",
				IterationIndex: &iter0,
				Type:           apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace:   "ns1",
					Type:        apiv2beta1.Artifact_Dataset,
					Uri:         strPTR("gs://bucket/iter-0"),
					Name:        "dataset-iter-0",
					Description: "Dataset from iteration 0",
				},
			},
			{
				RunId:          runid1,
				TaskId:         task.UUID,
				ProducerKey:    "iteration-output",
				IterationIndex: &iter1,
				Type:           apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace:   "ns1",
					Type:        apiv2beta1.Artifact_Dataset,
					Uri:         strPTR("gs://bucket/iter-1"),
					Name:        "dataset-iter-1",
					Description: "Dataset from iteration 1",
				},
			},
			{
				RunId:          runid1,
				TaskId:         task.UUID,
				ProducerKey:    "iteration-output",
				IterationIndex: &iter2,
				Type:           apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace:   "ns1",
					Type:        apiv2beta1.Artifact_Dataset,
					Uri:         strPTR("gs://bucket/iter-2"),
					Name:        "dataset-iter-2",
					Description: "Dataset from iteration 2",
				},
			},
		},
	}

	resp, err := s.CreateArtifactsBulk(ctxWithUser(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, len(resp.GetArtifacts()))

	// Verify all artifacts were created with correct iteration indices
	for i, artifact := range resp.GetArtifacts() {
		artifactTasks, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
			ArtifactIds: []string{artifact.GetArtifactId()},
			PageSize:    10,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(1), artifactTasks.GetTotalSize())

		at := artifactTasks.GetArtifactTasks()[0]
		assert.NotNil(t, at.GetProducer())
		assert.NotNil(t, at.GetProducer().Iteration)
		assert.Equal(t, int64(i), *at.GetProducer().Iteration)
	}
}

func TestArtifactServer_CreateArtifactsBulk_EmptyRequest(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Nil request should fail
	_, err := s.CreateArtifactsBulk(ctxWithUser(), nil)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "must contain at least one artifact")

	// Empty artifacts list should fail
	_, err = s.CreateArtifactsBulk(ctxWithUser(), &apiv2beta1.CreateArtifactsBulkRequest{
		Artifacts: []*apiv2beta1.CreateArtifactRequest{},
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "must contain at least one artifact")
}

func TestArtifactServer_CreateArtifactsBulk_ValidationErrors(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create run
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "validation-run",
		DisplayName:  "validation-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)

	task, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   runid1,
		Name:      "validation-task",
		State:     1,
	})
	assert.NoError(t, err)

	// Test with missing artifact
	_, err = s.CreateArtifactsBulk(ctxWithUser(), &apiv2beta1.CreateArtifactsBulkRequest{
		Artifacts: []*apiv2beta1.CreateArtifactRequest{
			{
				RunId:       runid1,
				TaskId:      task.UUID,
				ProducerKey: "output",
				Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact:    nil, // Missing artifact!
			},
		},
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Artifact is required")

	// Test with missing namespace
	_, err = s.CreateArtifactsBulk(ctxWithUser(), &apiv2beta1.CreateArtifactsBulkRequest{
		Artifacts: []*apiv2beta1.CreateArtifactRequest{
			{
				RunId:       runid1,
				TaskId:      task.UUID,
				ProducerKey: "output",
				Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace: "", // Missing namespace!
					Type:      apiv2beta1.Artifact_Model,
					Name:      "test",
				},
			},
		},
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "namespace is required")

	// Test with wrong run ID
	otherTask, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1",
		RunUUID:   "other-run-id",
		Name:      "other-task",
		State:     1,
	})
	assert.NoError(t, err)

	_, err = s.CreateArtifactsBulk(ctxWithUser(), &apiv2beta1.CreateArtifactsBulkRequest{
		Artifacts: []*apiv2beta1.CreateArtifactRequest{
			{
				RunId:       runid1,
				TaskId:      otherTask.UUID, // Task belongs to different run!
				ProducerKey: "output",
				Type:        apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
				Artifact: &apiv2beta1.Artifact{
					Namespace: "ns1",
					Type:      apiv2beta1.Artifact_Model,
					Name:      "test",
				},
			},
		},
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "does not belong to this Run ID")
}
