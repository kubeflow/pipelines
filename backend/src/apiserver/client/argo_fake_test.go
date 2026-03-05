// Copyright 2026 The Kubeflow Authors
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

package client

import (
	"context"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFakeExecClient_Execution(t *testing.T) {
	client := NewFakeExecClient()
	execution := client.Execution("default")
	if execution == nil {
		t.Error("Execution() returned nil for valid namespace")
	}
}

func TestFakeExecClient_ExecutionPanicsOnEmptyNamespace(t *testing.T) {
	client := NewFakeExecClient()

	defer func() {
		if recover() == nil {
			t.Error("Execution() did not panic for empty namespace")
		}
	}()
	client.Execution("")
}

func TestFakeExecClient_Compare(t *testing.T) {
	client := NewFakeExecClient()
	if client.Compare("old", "new") {
		t.Error("Compare() = true, want false")
	}
}

func TestFakeExecClient_GetWorkflowCount(t *testing.T) {
	client := NewFakeExecClient()
	if count := client.GetWorkflowCount(); count != 0 {
		t.Errorf("GetWorkflowCount() = %d, want 0", count)
	}

	ctx := context.Background()
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "test-wf"},
	})
	client.Execution("default").Create(ctx, workflow, v1.CreateOptions{})

	if count := client.GetWorkflowCount(); count != 1 {
		t.Errorf("GetWorkflowCount() after create = %d, want 1", count)
	}
}

func TestFakeExecClient_GetWorkflowKeys(t *testing.T) {
	client := NewFakeExecClient()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "key-workflow"},
	})
	client.Execution("default").Create(ctx, workflow, v1.CreateOptions{})

	keys := client.GetWorkflowKeys()
	if !keys["key-workflow"] {
		t.Errorf("GetWorkflowKeys() missing expected key, got: %v", keys)
	}
}

func TestFakeExecClient_IsTerminated(t *testing.T) {
	client := NewFakeExecClient()
	ctx := context.Background()

	terminatedDeadline := int64(0)
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "terminated-wf"},
		Spec: v1alpha1.WorkflowSpec{
			ActiveDeadlineSeconds: &terminatedDeadline,
		},
	})
	client.Execution("default").Create(ctx, workflow, v1.CreateOptions{})

	isTerminated, err := client.IsTerminated("terminated-wf")
	if err != nil {
		t.Fatalf("IsTerminated() unexpected error: %v", err)
	}
	if !isTerminated {
		t.Error("IsTerminated() = false, want true")
	}
}

func TestFakeExecClient_IsTerminatedNotFound(t *testing.T) {
	client := NewFakeExecClient()

	_, err := client.IsTerminated("nonexistent")
	if err == nil {
		t.Error("IsTerminated() expected error for nonexistent workflow, got nil")
	}
}

func TestFakeExecClient_IsTerminatedNoDeadline(t *testing.T) {
	client := NewFakeExecClient()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "no-deadline"},
	})
	client.Execution("default").Create(ctx, workflow, v1.CreateOptions{})

	_, err := client.IsTerminated("no-deadline")
	if err == nil {
		t.Error("IsTerminated() expected error for workflow without ActiveDeadlineSeconds, got nil")
	}
}

func TestFakeExecClientWithBadWorkflow_Execution(t *testing.T) {
	client := NewFakeExecClientWithBadWorkflow()
	execution := client.Execution("default")
	if execution == nil {
		t.Error("Execution() returned nil for valid namespace")
	}
}

func TestFakeExecClientWithBadWorkflow_Compare(t *testing.T) {
	client := NewFakeExecClientWithBadWorkflow()
	if client.Compare("old", "new") {
		t.Error("Compare() = true, want false")
	}
}
