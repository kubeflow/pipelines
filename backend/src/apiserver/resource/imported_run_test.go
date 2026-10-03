// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"bytes"
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImportedRunRejectsRuntimeWrites(t *testing.T) {
	for _, operation := range []string{
		"create run", "create task", "update tasks", "update tasks with run",
		"update tasks missing run ID", "report workflow", "report workflow with run", "report metric",
	} {
		t.Run(operation, func(t *testing.T) {
			store, manager, run := initWithOneTimeFailedRun(t)
			defer store.Close()
			storedTask, err := manager.CreateTask(&model.Task{
				RunID: run.UUID, Namespace: run.Namespace, PodName: "historical-pod",
				State: model.RuntimeStateSucceeded,
			})
			require.NoError(t, err)
			workflow, err := manager.getWorkflowClient(run.Namespace).Get(context.Background(), run.K8SName, v1.GetOptions{})
			require.NoError(t, err)
			_, err = store.DB().Exec(`UPDATE run_details SET ImportedFrom = ? WHERE UUID = ?`, "source-cluster", run.UUID)
			require.NoError(t, err)
			before, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			beforeTask, err := manager.GetTask(storedTask.UUID)
			require.NoError(t, err)
			incoming := &model.Task{
				UUID: storedTask.UUID, RunID: run.UUID, Namespace: run.Namespace,
				PodName: storedTask.PodName, Fingerprint: "must-not-restore-cache-key",
				State: model.RuntimeStateRunning,
			}
			manager.execClient = nil
			manager.k8sCoreClient = nil

			switch operation {
			case "create run":
				_, err = manager.CreateRun(context.Background(), &model.Run{UUID: run.UUID})
			case "create task":
				_, err = manager.CreateTask(incoming)
			case "update tasks":
				_, err = manager.CreateOrUpdateTasks([]*model.Task{incoming}, run.UUID, run.Namespace)
			case "update tasks with run":
				_, err = manager.CreateOrUpdateTasksForRun([]*model.Task{incoming}, before, run.Namespace)
			case "update tasks missing run ID":
				incoming.RunID = ""
				_, err = manager.CreateOrUpdateTasksForRun([]*model.Task{incoming}, before, run.Namespace)
			case "report workflow":
				_, err = manager.ReportWorkflowResource(context.Background(), workflow)
			case "report workflow with run":
				_, err = manager.ReportWorkflowResourceWithRun(context.Background(), workflow, before)
			case "report metric":
				err = manager.ReportMetric(&model.RunMetric{RunUUID: run.UUID, Name: "changed", NodeID: "node", NumberValue: 1})
			}
			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, err.(*util.UserError).ExternalStatusCode())
			assert.Contains(t, err.Error(), "imported historical run")
			after, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			assert.Equal(t, before, after, "runtime updates must not change imported history or metrics")
			afterTask, err := manager.GetTask(storedTask.UUID)
			require.NoError(t, err)
			assert.Equal(t, beforeTask, afterTask, "runtime updates must not repopulate cache fingerprints")
			var taskCount int
			require.NoError(t, store.DB().QueryRow(`SELECT COUNT(*) FROM tasks WHERE RunUUID = ?`, run.UUID).Scan(&taskCount))
			assert.Equal(t, 1, taskCount, "runtime updates must not create another task")
		})
	}
}

func TestImportedRunRejectsExecutionChanges(t *testing.T) {
	for _, operation := range []string{"retry", "terminate"} {
		t.Run(operation, func(t *testing.T) {
			store, manager, run := initWithOneTimeFailedRun(t)
			defer store.Close()
			_, err := store.DB().Exec(`UPDATE run_details SET ImportedFrom = ? WHERE UUID = ?`, "source-cluster", run.UUID)
			require.NoError(t, err)
			before, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			// Any attempt to contact Kubernetes must fail this test.
			manager.execClient = nil
			manager.k8sCoreClient = nil

			if operation == "retry" {
				err = manager.RetryRun(context.Background(), run.UUID)
			} else {
				err = manager.TerminateRun(context.Background(), run.UUID)
			}
			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, err.(*util.UserError).ExternalStatusCode())
			assert.Contains(t, err.Error(), "imported historical run")
			after, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			assert.Equal(t, before, after, "rejected operations must not mutate imported history")
		})
	}
}

func TestDeleteImportedRunPreservesLocalWorkflow(t *testing.T) {
	store, manager, run := initWithOneTimeFailedRun(t)
	defer store.Close()
	_, err := store.DB().Exec(`UPDATE run_details SET ImportedFrom = ? WHERE UUID = ?`, "source-cluster", run.UUID)
	require.NoError(t, err)
	workflowClient := manager.getWorkflowClient(run.Namespace)
	before, err := workflowClient.Get(context.Background(), run.K8SName, v1.GetOptions{})
	require.NoError(t, err)

	require.NoError(t, manager.DeleteRun(context.Background(), run.UUID))
	_, err = manager.GetRun(run.UUID)
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	after, err := workflowClient.Get(context.Background(), run.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, before, after, "history deletion must preserve even a matching local workflow")
}

func TestReadImportedRunLogsSkipsLocalPods(t *testing.T) {
	for _, archived := range []bool{false, true} {
		name := "without archive"
		if archived {
			name = "with archive"
		}
		t.Run(name, func(t *testing.T) {
			store, manager, run := initWithOneTimeFailedRun(t)
			defer store.Close()
			_, err := store.DB().Exec(`UPDATE run_details SET ImportedFrom = ?, WorkflowRuntimeManifest = ? WHERE UUID = ?`, "source-cluster", testWorkflow.ToStringForStore(), run.UUID)
			require.NoError(t, err)
			manager.k8sCoreClient = nil
			manager.logArchive = nil
			if archived {
				manager.logArchive = archive.NewLogArchive("/logs", "main.log")
				execSpec, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(testWorkflow.ToStringForStore()))
				require.NoError(t, err)
				logPath, err := manager.logArchive.GetLogObjectKey(execSpec, "node-id")
				require.NoError(t, err)
				manager.objectStore = &readerOnlyObjectStore{files: map[string][]byte{logPath: []byte("source cluster log\n")}}
			}

			var dst bytes.Buffer
			err = manager.ReadLog(context.Background(), run.UUID, "node-id", true, &dst)
			if archived {
				require.NoError(t, err)
				assert.Equal(t, "source cluster log\n", dst.String())
			} else {
				require.Error(t, err)
				assert.Equal(t, codes.FailedPrecondition, err.(*util.UserError).ExternalStatusCode())
				assert.Contains(t, err.Error(), "configure access to its archived logs")
				assert.Empty(t, dst.String())
			}
		})
	}
}
