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
	"k8s.io/apimachinery/pkg/types"
)

func TestFakeWorkflowClient_Create(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: "my-workflow",
		},
	})

	result, err := client.Create(ctx, workflow, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if result.ExecutionName() != "my-workflow" {
		t.Errorf("Create() name = %q, want %q", result.ExecutionName(), "my-workflow")
	}
}

func TestFakeWorkflowClient_CreateWithGenerateName(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: "workflow-",
		},
	})

	result, err := client.Create(ctx, workflow, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	expectedName := "workflow-0"
	if result.ExecutionName() != expectedName {
		t.Errorf("Create() name = %q, want %q", result.ExecutionName(), expectedName)
	}
}

func TestFakeWorkflowClient_Get(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "test-workflow"},
	})
	if _, err := client.Create(ctx, workflow, v1.CreateOptions{}); err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	result, err := client.Get(ctx, "test-workflow", v1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if result.ExecutionName() != "test-workflow" {
		t.Errorf("Get() name = %q, want %q", result.ExecutionName(), "test-workflow")
	}
}

func TestFakeWorkflowClient_GetNotFound(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	_, err := client.Get(ctx, "nonexistent", v1.GetOptions{})
	if err == nil {
		t.Error("Get() expected error for nonexistent workflow, got nil")
	}
}

func TestFakeWorkflowClient_Delete(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "to-delete"},
	})
	if _, err := client.Create(ctx, workflow, v1.CreateOptions{}); err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	err := client.Delete(ctx, "to-delete", v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}
}

func TestFakeWorkflowClient_DeleteNotFound(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	err := client.Delete(ctx, "nonexistent", v1.DeleteOptions{})
	if err == nil {
		t.Error("Delete() expected error for nonexistent workflow, got nil")
	}
}

func TestFakeWorkflowClient_Update(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "update-me"},
	})
	if _, err := client.Create(ctx, workflow, v1.CreateOptions{}); err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	_, err := client.Update(ctx, workflow, v1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
}

func TestFakeWorkflowClient_UpdateNotFound(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "nonexistent"},
	})

	_, err := client.Update(ctx, workflow, v1.UpdateOptions{})
	if err == nil {
		t.Error("Update() expected error for nonexistent workflow, got nil")
	}
}

func TestFakeWorkflowClient_PatchTerminate(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "patch-me"},
	})
	if _, err := client.Create(ctx, workflow, v1.CreateOptions{}); err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	patchData := []byte(`{"spec":{"activeDeadlineSeconds":0}}`)
	result, err := client.Patch(ctx, "patch-me", types.MergePatchType, patchData, v1.PatchOptions{})
	if err != nil {
		t.Fatalf("Patch() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Patch() returned nil result")
	}
}

func TestFakeWorkflowClient_PatchNotFound(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	patchData := []byte(`{"spec":{"activeDeadlineSeconds":0}}`)
	_, err := client.Patch(ctx, "nonexistent", types.MergePatchType, patchData, v1.PatchOptions{})
	if err == nil {
		t.Error("Patch() expected error for nonexistent workflow, got nil")
	}
}

func TestFakeWorkflowClient_PatchMetadata(t *testing.T) {
	client := NewWorkflowClientFake()
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "metadata-patch"},
	})
	if _, err := client.Create(ctx, workflow, v1.CreateOptions{}); err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	patchData := []byte(`{"metadata":{"labels":{"some-label":"value"}}}`)
	result, err := client.Patch(ctx, "metadata-patch", types.MergePatchType, patchData, v1.PatchOptions{})
	if err != nil {
		t.Fatalf("Patch() unexpected error: %v", err)
	}
	labels := result.ExecutionObjectMeta().GetLabels()
	if labels[util.LabelKeyWorkflowPersistedFinalState] != "true" {
		t.Errorf("Patch() did not set persisted final state label, got labels: %v", labels)
	}
}

func TestFakeBadWorkflowClient_Create(t *testing.T) {
	client := &FakeBadWorkflowClient{}
	ctx := context.Background()

	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "test"},
	})

	_, err := client.Create(ctx, workflow, v1.CreateOptions{})
	if err == nil {
		t.Error("FakeBadWorkflowClient.Create() expected error, got nil")
	}
}

func TestFakeBadWorkflowClient_Get(t *testing.T) {
	client := &FakeBadWorkflowClient{}
	ctx := context.Background()

	_, err := client.Get(ctx, "test", v1.GetOptions{})
	if err == nil {
		t.Error("FakeBadWorkflowClient.Get() expected error, got nil")
	}
}

func TestFakeBadWorkflowClient_Delete(t *testing.T) {
	client := &FakeBadWorkflowClient{}
	ctx := context.Background()

	err := client.Delete(ctx, "test", v1.DeleteOptions{})
	if err == nil {
		t.Error("FakeBadWorkflowClient.Delete() expected error, got nil")
	}
}
