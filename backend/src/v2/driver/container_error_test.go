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

package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestContainer_PostCreateErrorUpdatesExistingTask(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/componentInput.yaml")

	err := tc.Push("process-inputs")
	if err != nil {
		t.Fatalf("failed to push scope path: %v", err)
	}
	defer func() {
		_, ok := tc.Pop()
		if !ok {
			t.Fatal("failed to pop scope path")
		}
	}()

	taskSpec := tc.GetLast().GetTaskSpec()
	kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	if err != nil {
		t.Fatalf("failed to load kubernetes executor config: %v", err)
	}
	opts := tc.setupContainerOptions(tc.RootTask, taskSpec, kubernetesExecutorConfig)

	k8sClient := tc.ClientManager.K8sClient().(*k8sfake.Clientset)
	k8sClient.PrependReactor("get", "configmaps", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("boom")
	})

	execution, err := Container(context.Background(), opts, tc.ClientManager)
	if err == nil {
		t.Fatal("expected container execution to fail")
	}
	if execution != nil && execution.TaskID == "" {
		t.Fatal("expected execution task ID to be empty or populated")
	}

	fullView := apiv2beta1.GetRunRequest_FULL
	run, err := tc.ClientManager.KFPAPIClient().GetRun(context.Background(), &apiv2beta1.GetRunRequest{
		RunId: tc.Run.GetRunId(),
		View:  &fullView,
	})
	if err != nil {
		t.Fatalf("failed to refresh run: %v", err)
	}
	var children []*apiv2beta1.PipelineTask
	for _, task := range run.GetTasks() {
		if task.GetParentTaskId() == tc.RootTask.GetTaskId() {
			children = append(children, task)
		}
	}
	if len(children) != 1 {
		t.Fatalf("expected exactly one child task, got %d", len(children))
	}

	task := children[0]
	if execution != nil && execution.TaskID != "" && task.GetTaskId() != execution.TaskID {
		t.Fatalf("expected execution task ID %q, got %q", execution.TaskID, task.GetTaskId())
	}
	if task.GetState() != apiv2beta1.PipelineTask_FAILED {
		t.Fatalf("expected failed task state, got %v", task.GetState())
	}
	if task.GetEndTime() == nil {
		t.Fatal("expected failed task to have an end time")
	}
}
