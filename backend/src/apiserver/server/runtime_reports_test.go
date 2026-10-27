// Copyright 2026 The Kubeflow Authors
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

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func managedReportFixture(t *testing.T) (*resource.FakeClientManager, *RunServer, *ArtifactServer, *model.Task, *model.Task, storage.RuntimeStoreInterface) {
	t.Helper()
	initEnvVars()
	clients, err := resource.NewFakeClientManager(util.NewRealTime(), util.NewUUIDGenerator())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	clients.DB().SetMaxOpenConns(1)
	experiment, err := clients.ExperimentStore().CreateExperiment(&model.Experiment{Name: "reports"})
	require.NoError(t, err)
	executor := "docker"
	_, err = clients.RunStore().CreateRun(&model.Run{
		UUID: "managed-report-run", ExperimentId: experiment.UUID, RunDetails: model.RunDetails{State: model.RuntimeStateRunning, ExecutorType: &executor},
	})
	require.NoError(t, err)
	parent, err := clients.TaskStore().CreateTask(&model.Task{RunUUID: "managed-report-run", Name: "root", State: model.TaskStatus(apiv2.PipelineTask_RUNNING)})
	require.NoError(t, err)
	task, err := clients.TaskStore().CreateTask(&model.Task{RunUUID: parent.RunUUID, Name: "leaf", ParentTaskUUID: &parent.UUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING)})
	require.NoError(t, err)
	runtime := clients.TaskStore().(*storage.TaskStore).RuntimeStore()
	attempt := "attempt-a"
	require.NoError(t, runtime.Create(&model.RuntimeTask{TaskUUID: task.UUID, RunUUID: task.RunUUID, AttemptUUID: &attempt, State: model.RuntimeTaskRunning}))
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	return clients, createRunServer(manager), createArtifactServer(manager), task, parent, runtime
}

func reportContext(taskID, attemptID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		apiclient.KFPTaskIDHeader, taskID, apiclient.KFPTaskAttemptIDHeader, attemptID,
	))
}

func reportedTask(value string) *apiv2.PipelineTask {
	return &apiv2.PipelineTask{
		State:          apiv2.PipelineTask_RUNNING,
		StatusMetadata: &apiv2.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{"attempt": structpb.NewStringValue(value)}},
		Pods:           []*apiv2.PipelineTask_TaskPod{{Name: value}},
		Outputs: &apiv2.PipelineTask_InputOutputs{Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{{
			ParameterKey: "result", Type: apiv2.IOType_OUTPUT, Value: structpb.NewStringValue(value),
		}}},
	}
}

func TestRuntimeLauncherRejectsReplacedAttemptReports(t *testing.T) {
	for _, mutation := range []string{"task", "bulk", "artifact", "bulk-artifacts", "link", "bulk-links"} {
		t.Run(mutation, func(t *testing.T) {
			clients, runs, artifacts, task, parent, runtime := managedReportFixture(t)
			attempt := "attempt-b"
			require.NoError(t, runtime.BeginRetry(task.UUID, "attempt-a", &model.RuntimeTask{TaskUUID: task.UUID, RunUUID: task.RunUUID, AttemptUUID: &attempt, State: model.RuntimeTaskRunning}))
			_, err := runs.UpdateTask(reportContext(task.UUID, attempt), &apiv2.UpdateTaskRequest{
				RunId: task.RunUUID, TaskId: task.UUID, Task: reportedTask("current"),
			})
			require.NoError(t, err)
			before, err := clients.TaskStore().GetTask(task.UUID)
			require.NoError(t, err)
			artifact, err := clients.ArtifactStore().CreateArtifact(&model.Artifact{Name: "existing", Type: model.ArtifactType(apiv2.Artifact_Model)})
			require.NoError(t, err)
			link := &apiv2.ArtifactTask{ArtifactId: artifact.UUID, TaskId: task.UUID, RunId: task.RunUUID, Type: apiv2.IOType_OUTPUT, Key: "result", Producer: &apiv2.IOProducer{TaskName: task.Name}}
			ctx := reportContext(task.UUID, "attempt-a")
			switch mutation {
			case "task":
				_, err = runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: task.RunUUID, TaskId: task.UUID, Task: reportedTask("stale")})
			case "bulk":
				_, err = runs.UpdateTasksBulk(ctx, &apiv2.UpdateTasksBulkRequest{RunId: task.RunUUID, Tasks: map[string]*apiv2.PipelineTask{
					task.UUID: reportedTask("stale"), parent.UUID: reportedTask("stale-parent"),
				}})
			case "artifact":
				_, err = artifacts.CreateArtifact(ctx, &apiv2.CreateArtifactRequest{RunId: task.RunUUID, TaskId: task.UUID, ProducerKey: "result", Artifact: &apiv2.Artifact{Namespace: "ns1", Type: apiv2.Artifact_Model, Name: "stale"}})
			case "bulk-artifacts":
				_, err = artifacts.CreateArtifactsBulk(ctx, &apiv2.CreateArtifactsBulkRequest{Artifacts: []*apiv2.CreateArtifactRequest{{
					RunId: task.RunUUID, TaskId: task.UUID, ProducerKey: "result", Artifact: &apiv2.Artifact{Namespace: "ns1", Type: apiv2.Artifact_Model, Name: "stale"},
				}}})
			case "link":
				_, err = artifacts.CreateArtifactTask(ctx, &apiv2.CreateArtifactTaskRequest{ArtifactTask: link})
			case "bulk-links":
				_, err = artifacts.CreateArtifactTasksBulk(ctx, &apiv2.CreateArtifactTasksBulkRequest{ArtifactTasks: []*apiv2.ArtifactTask{link}})
			}

			require.Error(t, err, "old launcher must not mutate a newer attempt")
			after, err := clients.TaskStore().GetTask(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			var count int
			require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
			assert.Equal(t, 1, count, "rejected report must not create orphaned artifact metadata")
			require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&count))
			assert.Zero(t, count)
		})
	}
}

func TestRuntimeLauncherCurrentReportsAndIdempotentArtifactReplay(t *testing.T) {
	clients, runs, artifacts, task, parent, runtime := managedReportFixture(t)
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	current.State = model.RuntimeTaskSubmitting
	require.NoError(t, runtime.Update(current, "attempt-a"))
	ctx := reportContext(task.UUID, "attempt-a")
	terminal := reportedTask("current")
	terminal.State = apiv2.PipelineTask_SUCCEEDED
	updated, err := runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: task.RunUUID, TaskId: task.UUID, Task: terminal})
	require.NoError(t, err)
	assert.Equal(t, apiv2.PipelineTask_RUNNING, updated.State, "only framework finalization may expose terminal results")
	require.NotEmpty(t, updated.GetOutputs().GetParameters())
	_, err = runs.UpdateTasksBulk(ctx, &apiv2.UpdateTasksBulkRequest{RunId: task.RunUUID, Tasks: map[string]*apiv2.PipelineTask{
		task.UUID: reportedTask("current"), parent.UUID: reportedTask("parent-output"),
	}})
	require.NoError(t, err)
	storedParent, err := clients.TaskStore().GetTask(parent.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), storedParent.State)
	assert.Empty(t, storedParent.StatusMetadata, "child propagation must not replace parent diagnostics")
	assert.Empty(t, storedParent.Pods)
	assert.NotEmpty(t, storedParent.OutputParameters)

	uri := "file:///artifacts/output"
	number := 0.75
	request := &apiv2.CreateArtifactsBulkRequest{Artifacts: []*apiv2.CreateArtifactRequest{
		{RunId: task.RunUUID, TaskId: task.UUID, ProducerKey: "result",
			Artifact: &apiv2.Artifact{Namespace: "ignored", Type: apiv2.Artifact_Model, Name: "result", Uri: &uri, CreatedAt: &timestamppb.Timestamp{Seconds: 1}}},
		{RunId: task.RunUUID, TaskId: task.UUID, ProducerKey: "accuracy",
			Artifact: &apiv2.Artifact{Namespace: "ignored", Type: apiv2.Artifact_Metric, Name: "accuracy", NumberValue: &number, CreatedAt: &timestamppb.Timestamp{Seconds: 1}}},
	}}
	first, err := artifacts.CreateArtifactsBulk(ctx, request)
	require.NoError(t, err)
	replayedRequest := proto.Clone(request).(*apiv2.CreateArtifactsBulkRequest)
	for _, artifact := range replayedRequest.Artifacts {
		artifact.Artifact.CreatedAt = &timestamppb.Timestamp{Seconds: 2}
	}
	replay, err := artifacts.CreateArtifactsBulk(ctx, replayedRequest)
	require.NoError(t, err)
	require.Len(t, first.Artifacts, 2)
	for i := range first.Artifacts {
		assert.Equal(t, first.Artifacts[i].ArtifactId, replay.Artifacts[i].ArtifactId)
	}
	link := &apiv2.ArtifactTask{ArtifactId: first.Artifacts[0].ArtifactId, TaskId: task.UUID, RunId: task.RunUUID, Type: apiv2.IOType_OUTPUT, Key: "result", Producer: &apiv2.IOProducer{TaskName: task.Name}}
	firstLink, err := artifacts.CreateArtifactTask(ctx, &apiv2.CreateArtifactTaskRequest{ArtifactTask: link})
	require.NoError(t, err)
	replayedLink, err := artifacts.CreateArtifactTask(ctx, &apiv2.CreateArtifactTaskRequest{ArtifactTask: link})
	require.NoError(t, err)
	assert.Equal(t, firstLink.Id, replayedLink.Id)
	parentLink := proto.Clone(link).(*apiv2.ArtifactTask)
	parentLink.TaskId, parentLink.Key = parent.UUID, "parent-result"
	parentLinks, err := artifacts.CreateArtifactTasksBulk(ctx, &apiv2.CreateArtifactTasksBulkRequest{ArtifactTasks: []*apiv2.ArtifactTask{parentLink}})
	require.NoError(t, err)
	replayedParentLinks, err := artifacts.CreateArtifactTasksBulk(ctx, &apiv2.CreateArtifactTasksBulkRequest{ArtifactTasks: []*apiv2.ArtifactTask{parentLink}})
	require.NoError(t, err)
	assert.Equal(t, parentLinks.ArtifactTasks[0].Id, replayedParentLinks.ArtifactTasks[0].Id)
	var count int
	require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
	assert.Equal(t, 2, count)
	require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&count))
	assert.Equal(t, 3, count)
}

func TestRuntimeLauncherRejectsMissingIdentityAfterSuccessfulFinalization(t *testing.T) {
	clients, runs, artifacts, task, _, runtime := managedReportFixture(t)
	require.NoError(t, runtime.Finalize(task.UUID, "attempt-a", &model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)}))
	for _, ctx := range []context.Context{context.Background(), reportContext(task.UUID, "attempt-a")} {
		_, err := runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: task.RunUUID, TaskId: task.UUID, Task: reportedTask("late")})
		require.Error(t, err)
		_, err = artifacts.CreateArtifact(ctx, &apiv2.CreateArtifactRequest{
			RunId: task.RunUUID, TaskId: task.UUID, ProducerKey: "late",
			Artifact: &apiv2.Artifact{Namespace: "ignored", Type: apiv2.Artifact_Model, Name: "late"},
		})
		require.Error(t, err)
	}
	stored, err := clients.TaskStore().GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), stored.State)
	assert.Empty(t, stored.OutputParameters)
	var count int
	require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
	assert.Zero(t, count)
}

func TestRuntimeLauncherBatchScopeAndRollback(t *testing.T) {
	clients, runs, artifacts, task, parent, _ := managedReportFixture(t)
	sibling, err := clients.TaskStore().CreateTask(&model.Task{RunUUID: task.RunUUID, Name: "sibling", ParentTaskUUID: &parent.UUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING)})
	require.NoError(t, err)
	ctx := reportContext(task.UUID, "attempt-a")
	_, err = clients.RunStore().CreateRun(&model.Run{UUID: "other-report-run", RunDetails: model.RunDetails{State: model.RuntimeStateRunning}})
	require.NoError(t, err)
	other, err := clients.TaskStore().CreateTask(&model.Task{RunUUID: "other-report-run", Name: "other", State: model.TaskStatus(apiv2.PipelineTask_RUNNING)})
	require.NoError(t, err)
	_, err = runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: other.RunUUID, TaskId: other.UUID, Task: reportedTask("wrong-run")})
	require.Error(t, err, "a current attempt cannot report into another run")
	_, err = runs.UpdateTasksBulk(ctx, &apiv2.UpdateTasksBulkRequest{RunId: task.RunUUID, Tasks: map[string]*apiv2.PipelineTask{
		task.UUID: reportedTask("must-rollback"), sibling.UUID: reportedTask("forbidden"),
	}})
	require.Error(t, err, "a current attempt is not permission to mutate siblings")
	_, err = runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: task.RunUUID, TaskId: task.UUID, Task: &apiv2.PipelineTask{ParentTaskId: &sibling.UUID}})
	require.Error(t, err, "reports cannot rewrite lineage to gain additional ancestor permissions")
	_, err = artifacts.CreateArtifactsBulk(ctx, &apiv2.CreateArtifactsBulkRequest{Artifacts: []*apiv2.CreateArtifactRequest{
		{RunId: task.RunUUID, TaskId: task.UUID, ProducerKey: "valid", Artifact: &apiv2.Artifact{Namespace: "ignored", Type: apiv2.Artifact_Model, Name: "valid"}},
		{RunId: task.RunUUID, TaskId: sibling.UUID, ProducerKey: "invalid", Artifact: &apiv2.Artifact{Namespace: "ignored", Type: apiv2.Artifact_Model, Name: "invalid"}},
	}})
	require.Error(t, err)
	for _, id := range []string{task.UUID, sibling.UUID} {
		stored, err := clients.TaskStore().GetTask(id)
		require.NoError(t, err)
		assert.Empty(t, stored.OutputParameters)
	}
	var count int
	require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
	assert.Zero(t, count, "all metadata in an invalid bulk report must roll back")
}

func TestRuntimeLauncherManagedCreateAndMalformedMetadataRejected(t *testing.T) {
	clients, runs, _, task, _, _ := managedReportFixture(t)
	for _, ctx := range []context.Context{context.Background(), reportContext(task.UUID, "attempt-a")} {
		_, err := runs.CreateTask(ctx, &apiv2.CreateTaskRequest{RunId: task.RunUUID, Task: &apiv2.PipelineTask{Name: "injected", Type: apiv2.PipelineTask_RUNTIME}})
		require.Error(t, err, "managed tasks can only be materialized by the coordinator")
	}
	for _, headers := range []metadata.MD{
		metadata.Pairs(apiclient.KFPTaskIDHeader, task.UUID),
		metadata.Pairs(apiclient.KFPTaskIDHeader, task.UUID, apiclient.KFPTaskAttemptIDHeader, ""),
		metadata.Pairs(apiclient.KFPTaskIDHeader, task.UUID, apiclient.KFPTaskAttemptIDHeader, "attempt-a", apiclient.KFPTaskAttemptIDHeader, "attempt-b"),
	} {
		ctx := metadata.NewIncomingContext(context.Background(), headers)
		_, err := runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: task.RunUUID, TaskId: task.UUID, Task: reportedTask("invalid")})
		require.Error(t, err)
	}
	stored, err := clients.TaskStore().GetTask(task.UUID)
	require.NoError(t, err)
	assert.Empty(t, stored.OutputParameters)
}
func TestRuntimeLauncherCannotResurrectFinalizedCanceledTask(t *testing.T) {
	clients, runs, _, task, _, runtime := managedReportFixture(t)
	_, err := clients.DB().Exec("UPDATE run_details SET State = 'CANCELING' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	require.NoError(t, runtime.Finalize(task.UUID, "attempt-a", &model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)}))
	for _, ctx := range []context.Context{reportContext(task.UUID, "attempt-a"), context.Background()} {
		_, err := runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: task.RunUUID, TaskId: task.UUID, Task: reportedTask("late")})
		require.Error(t, err, "managed run marker must reject reports after coordination row deletion, including missing identity")
	}
	stored, err := clients.TaskStore().GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), stored.State)
	assert.Empty(t, stored.OutputParameters)
}
