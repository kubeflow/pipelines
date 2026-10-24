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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeLauncherAllMutationsRejectMissingCanceledOrFinalizedIdentity(t *testing.T) {
	for _, phase := range []string{"missing-identity", "canceling", "finalized", "canceled-finalized", "run-finalized", "publication-finalizing"} {
		t.Run(phase, func(t *testing.T) {
			clients, runs, artifacts, task, parent, runtime := managedReportFixture(t)
			existingArtifact, err := clients.ArtifactStore().CreateArtifact(&model.Artifact{Name: "existing", Type: model.ArtifactType(apiv2.Artifact_Model)})
			require.NoError(t, err)
			expected := model.TaskStatus(apiv2.PipelineTask_RUNNING)
			if phase == "run-finalized" {
				_, err := clients.DB().Exec("UPDATE run_details SET State = 'SUCCEEDED' WHERE UUID = ?", task.RunUUID)
				require.NoError(t, err)
			}
			if phase == "publication-finalizing" {
				current, err := runtime.Get(task.UUID)
				require.NoError(t, err)
				current.State = model.RuntimeTaskFinalizing
				require.NoError(t, runtime.Update(current, "attempt-a"))
			}
			if phase == "canceling" || phase == "canceled-finalized" {
				_, err := clients.DB().Exec("UPDATE run_details SET State = 'CANCELING' WHERE UUID = ?", task.RunUUID)
				require.NoError(t, err)
			}
			if phase == "finalized" || phase == "canceled-finalized" {
				expected = model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
				if phase == "canceled-finalized" {
					expected = model.TaskStatus(apiv2.PipelineTask_SKIPPED)
				}
				require.NoError(t, runtime.Finalize(task.UUID, "attempt-a", &model.Task{UUID: task.UUID, State: expected}))
			}
			request := &apiv2.CreateArtifactRequest{
				RunId: task.RunUUID, TaskId: task.UUID, ProducerKey: "late",
				Artifact: &apiv2.Artifact{Namespace: "ignored", Type: apiv2.Artifact_Model, Name: "late"},
			}
			link := &apiv2.ArtifactTask{
				RunId: task.RunUUID, TaskId: task.UUID, ArtifactId: existingArtifact.UUID,
				Type: apiv2.IOType_OUTPUT, Key: "late", Producer: &apiv2.IOProducer{TaskName: task.Name},
			}
			mutations := map[string]func(context.Context) error{
				"create-task": func(ctx context.Context) error {
					_, err := runs.CreateTask(ctx, &apiv2.CreateTaskRequest{RunId: task.RunUUID, Task: &apiv2.PipelineTask{Name: "late"}})
					return err
				},
				"task": func(ctx context.Context) error {
					_, err := runs.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: task.RunUUID, TaskId: task.UUID, Task: reportedTask("late")})
					return err
				},
				"bulk": func(ctx context.Context) error {
					_, err := runs.UpdateTasksBulk(ctx, &apiv2.UpdateTasksBulkRequest{RunId: task.RunUUID, Tasks: map[string]*apiv2.PipelineTask{task.UUID: reportedTask("late"), parent.UUID: reportedTask("late-parent")}})
					return err
				},
				"artifact": func(ctx context.Context) error {
					_, err := artifacts.CreateArtifact(ctx, request)
					return err
				},
				"bulk-artifacts": func(ctx context.Context) error {
					_, err := artifacts.CreateArtifactsBulk(ctx, &apiv2.CreateArtifactsBulkRequest{Artifacts: []*apiv2.CreateArtifactRequest{request}})
					return err
				},
				"link": func(ctx context.Context) error {
					_, err := artifacts.CreateArtifactTask(ctx, &apiv2.CreateArtifactTaskRequest{ArtifactTask: link})
					return err
				},
				"bulk-links": func(ctx context.Context) error {
					_, err := artifacts.CreateArtifactTasksBulk(ctx, &apiv2.CreateArtifactTasksBulkRequest{ArtifactTasks: []*apiv2.ArtifactTask{link}})
					return err
				},
			}
			contexts := []context.Context{context.Background()}
			if phase != "missing-identity" {
				contexts = append(contexts, reportContext(task.UUID, "attempt-a"))
			}
			for name, mutation := range mutations {
				for _, ctx := range contexts {
					require.Error(t, mutation(ctx), "%s report must be rejected in phase %s", name, phase)
				}
			}
			stored, err := clients.TaskStore().GetTask(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, expected, stored.State)
			assert.Empty(t, stored.OutputParameters)
			assert.Empty(t, stored.StatusMetadata)
			storedParent, err := clients.TaskStore().GetTask(parent.UUID)
			require.NoError(t, err)
			assert.Empty(t, storedParent.OutputParameters)
			var count int
			require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
			assert.Equal(t, 1, count)
			require.NoError(t, clients.DB().QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&count))
			assert.Zero(t, count)
		})
	}
}
