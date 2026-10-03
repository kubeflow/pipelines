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

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Unimplemented interface methods and Kubernetes clients deliberately remain nil:
// a rejected request must return before any runtime mutation or local cluster access.
type importedHistoryRunStore struct {
	storage.RunStoreInterface
	run     *model.Run
	deleted bool
}

func (s *importedHistoryRunStore) GetRun(id string, _ bool) (*model.Run, error) {
	if id != s.run.UUID || s.deleted {
		return nil, util.NewResourceNotFoundError("run", id)
	}
	return s.run, nil
}

func (s *importedHistoryRunStore) DeleteRun(id string) error {
	if id != s.run.UUID {
		return util.NewResourceNotFoundError("run", id)
	}
	s.deleted = true
	return nil
}

type importedHistoryTaskStore struct {
	storage.TaskStoreInterface
	task *model.Task
}

func (s *importedHistoryTaskStore) GetTask(string) (*model.Task, error) {
	return s.task, nil
}

func newImportedHistoryManager() (*ResourceManager, *importedHistoryRunStore) {
	store := &importedHistoryRunStore{run: &model.Run{
		UUID: "historical-run", ImportedFrom: "old-installation", K8SName: "shared-workflow-name",
		RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
	}}
	return &ResourceManager{runStore: store}, store
}

func TestImportedHistoryRejectsRuntimeMutations(t *testing.T) {
	for _, operation := range []string{"retry", "terminate", "create task", "update task", "move task", "report running", "report terminal", "report loaded"} {
		t.Run(operation, func(t *testing.T) {
			manager, store := newImportedHistoryManager()
			var err error
			switch operation {
			case "retry":
				err = manager.RetryRun(context.Background(), store.run.UUID)
			case "terminate":
				err = manager.TerminateRun(context.Background(), store.run.UUID)
			case "create task":
				_, err = manager.CreateTask(&model.Task{RunUUID: store.run.UUID})
			case "update task", "move task":
				manager.taskStore = &importedHistoryTaskStore{task: &model.Task{UUID: "historical-task", RunUUID: store.run.UUID}}
				task := &model.Task{UUID: "historical-task"}
				if operation == "move task" {
					task.RunUUID = "local-run"
				}
				_, err = manager.UpdateTask(task)
			default:
				phase := v1alpha1.WorkflowRunning
				if operation != "report running" {
					phase = v1alpha1.WorkflowSucceeded
				}
				workflow := util.NewWorkflow(&v1alpha1.Workflow{
					ObjectMeta: metav1.ObjectMeta{Name: store.run.K8SName, Namespace: "ns", Labels: map[string]string{util.LabelKeyWorkflowRunId: store.run.UUID}},
					Status:     v1alpha1.WorkflowStatus{Phase: phase},
				})
				if operation == "report loaded" {
					_, err = manager.ReportWorkflowResourceWithRun(context.Background(), workflow, store.run)
				} else {
					_, err = manager.ReportWorkflowResource(context.Background(), workflow)
				}
			}
			require.Error(t, err)
			assert.True(t, util.IsUserErrorCodeMatch(err, codes.FailedPrecondition), "%v", err)
			assert.Contains(t, err.Error(), "imported history")
			assert.Equal(t, model.RuntimeStateSucceeded, store.run.State)
			assert.False(t, store.deleted)
		})
	}
}

func TestImportedHistoryRejectsArtifactTaskWrites(t *testing.T) {
	for _, operation := range []string{"link", "bulk links", "artifact", "reused artifact", "bulk artifacts"} {
		t.Run(operation, func(t *testing.T) {
			manager, store := newImportedHistoryManager()
			link := &model.ArtifactTask{RunUUID: store.run.UUID, TaskID: "historical-task"}
			var err error
			switch operation {
			case "link":
				_, err = manager.CreateArtifactTask(link)
			case "bulk links":
				_, err = manager.CreateArtifactTasks([]*model.ArtifactTask{link})
			case "artifact":
				_, _, err = manager.CreateArtifactWithTask(&model.Artifact{}, link)
			case "reused artifact":
				_, _, err = manager.FindOrCreateArtifactWithTask(&model.Artifact{}, link)
			case "bulk artifacts":
				_, _, err = manager.CreateArtifactsWithTasks([]*model.Artifact{{}}, []*model.ArtifactTask{link})
			}
			require.Error(t, err)
			assert.True(t, util.IsUserErrorCodeMatch(err, codes.FailedPrecondition), "%v", err)
		})
	}
}

func TestImportedHistoryDeleteDoesNotAccessKubernetes(t *testing.T) {
	manager, store := newImportedHistoryManager()
	require.NoError(t, manager.DeleteRun(context.Background(), store.run.UUID))
	assert.True(t, store.deleted)
}

func TestImportedHistoryLogsUseOnlyArchive(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(map[bool]string{false: "unconfigured", true: "configured"}[configured], func(t *testing.T) {
			manager, store := newImportedHistoryManager()
			store.run.WorkflowRuntimeManifest = model.LargeText(testWorkflow.ToStringForStore())
			if configured {
				manager.logArchive = archive.NewLogArchive("/logs", "main.log")
				key, err := manager.logArchive.GetLogObjectKey(testWorkflow, "node-id")
				require.NoError(t, err)
				manager.objectStore = &readerOnlyObjectStore{files: map[string][]byte{key: []byte("old log\n")}}
			}
			var dst bytes.Buffer
			err := manager.ReadLog(context.Background(), store.run.UUID, "node-id", true, &dst)
			if configured {
				require.NoError(t, err)
				assert.Equal(t, "old log\n", dst.String())
			} else {
				require.Error(t, err)
				assert.True(t, util.IsUserErrorCodeMatch(err, codes.FailedPrecondition), "%v", err)
				assert.Empty(t, dst.String())
			}
		})
	}
}
