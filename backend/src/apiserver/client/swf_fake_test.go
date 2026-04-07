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

	"github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFakeSwfClient_ScheduledWorkflow(t *testing.T) {
	client := NewFakeSwfClient()
	scheduledWorkflow := client.ScheduledWorkflow("default")
	if scheduledWorkflow == nil {
		t.Error("ScheduledWorkflow() returned nil for valid namespace")
	}
}

func TestFakeSwfClient_CreateAndGet(t *testing.T) {
	client := NewFakeSwfClient()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	swf := &v1beta1.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: "my-scheduled-wf",
		},
	}

	created, err := swfClient.Create(ctx, swf)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if created.Name != "my-scheduled-wf" {
		t.Errorf("Create() Name = %q, want %q", created.Name, "my-scheduled-wf")
	}
	if created.Namespace != "ns1" {
		t.Errorf("Create() Namespace = %q, want %q", created.Namespace, "ns1")
	}

	fetched, err := swfClient.Get(ctx, "my-scheduled-wf", v1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if fetched.Name != "my-scheduled-wf" {
		t.Errorf("Get() Name = %q, want %q", fetched.Name, "my-scheduled-wf")
	}
}

func TestFakeSwfClient_GetNotFound(t *testing.T) {
	client := NewFakeSwfClient()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	_, err := swfClient.Get(ctx, "nonexistent", v1.GetOptions{})
	if err == nil {
		t.Error("Get() expected error for nonexistent scheduled workflow, got nil")
	}
}

func TestFakeSwfClient_Delete(t *testing.T) {
	client := NewFakeSwfClient()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	swf := &v1beta1.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "to-delete"},
	}
	if _, err := swfClient.Create(ctx, swf); err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	err := swfClient.Delete(ctx, "to-delete", &v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	_, err = swfClient.Get(ctx, "to-delete", v1.GetOptions{})
	if err == nil {
		t.Error("Get() after Delete() expected error, got nil")
	}
}

func TestFakeSwfClient_DeleteNotFound(t *testing.T) {
	client := NewFakeSwfClient()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	err := swfClient.Delete(ctx, "nonexistent", &v1.DeleteOptions{})
	if err == nil {
		t.Error("Delete() expected error for nonexistent scheduled workflow, got nil")
	}
}

func TestFakeSwfClientWithBadWorkflow_ScheduledWorkflow(t *testing.T) {
	client := NewFakeSwfClientWithBadWorkflow()
	scheduledWorkflow := client.ScheduledWorkflow("default")
	if scheduledWorkflow == nil {
		t.Error("ScheduledWorkflow() returned nil for valid namespace")
	}
}

func TestFakeSwfClient_Update(t *testing.T) {
	client := NewFakeSwfClient()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	swf := &v1beta1.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "update-me"},
	}
	created, err := swfClient.Create(ctx, swf)
	if err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	fetchedBeforeUpdate, err := swfClient.Get(ctx, "update-me", v1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() before Update() unexpected error: %v", err)
	}
	if fetchedBeforeUpdate.Spec.MaxConcurrency != nil {
		t.Fatal("Create() unexpectedly persisted MaxConcurrency before Update")
	}

	updatedInput := created.DeepCopy()
	updatedInput.Spec.MaxConcurrency = intPtr(5)
	updated, err := swfClient.Update(ctx, updatedInput)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if updated.Name != "update-me" {
		t.Errorf("Update() Name = %q, want %q", updated.Name, "update-me")
	}

	fetched, err := swfClient.Get(ctx, "update-me", v1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() after Update() unexpected error: %v", err)
	}
	if fetched.Spec.MaxConcurrency == nil || *fetched.Spec.MaxConcurrency != 5 {
		t.Errorf("Get() after Update() MaxConcurrency not persisted")
	}
}

func TestFakeSwfClient_Patch(t *testing.T) {
	client := NewFakeSwfClient()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	swf := &v1beta1.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "patch-me"},
	}
	if _, err := swfClient.Create(ctx, swf); err != nil {
		t.Fatalf("setup: Create() unexpected error: %v", err)
	}

	result, err := swfClient.Patch(ctx, "patch-me", "application/merge-patch+json", []byte(`{}`))
	if err != nil {
		t.Fatalf("Patch() unexpected error: %v", err)
	}
	if result != nil {
		t.Error("Patch() expected nil result from stub implementation")
	}
}

func TestFakeScheduledWorkflowClient_Stubs(t *testing.T) {
	swfClient := NewFakeSwfClient().ScheduledWorkflow("default")
	ctx := context.Background()

	t.Run("DeleteCollection", func(t *testing.T) {
		if err := swfClient.DeleteCollection(ctx, &v1.DeleteOptions{}, v1.ListOptions{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("List", func(t *testing.T) {
		list, err := swfClient.List(ctx, v1.ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if list != nil {
			t.Error("expected nil from unimplemented stub")
		}
	})
	t.Run("Watch", func(t *testing.T) {
		watcher, err := swfClient.Watch(ctx, v1.ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if watcher != nil {
			t.Error("expected nil from unimplemented stub")
		}
	})
}

func TestFakeSwfClientWithBadWorkflow_Create(t *testing.T) {
	client := NewFakeSwfClientWithBadWorkflow()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	swf := &v1beta1.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "bad-wf"},
	}

	_, err := swfClient.Create(ctx, swf)
	if err == nil {
		t.Error("Create() expected error for bad workflow client, got nil")
	}
}

func TestFakeSwfClientWithBadWorkflow_Get(t *testing.T) {
	client := NewFakeSwfClientWithBadWorkflow()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	_, err := swfClient.Get(ctx, "anything", v1.GetOptions{})
	if err == nil {
		t.Error("Get() expected error for bad workflow client, got nil")
	}
}

func TestFakeSwfClientWithBadWorkflow_Patch(t *testing.T) {
	client := NewFakeSwfClientWithBadWorkflow()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	_, err := swfClient.Patch(ctx, "anything", "application/merge-patch+json", []byte(`{}`))
	if err == nil {
		t.Error("Patch() expected error for bad workflow client, got nil")
	}
}

func TestFakeSwfClientWithBadWorkflow_Delete(t *testing.T) {
	client := NewFakeSwfClientWithBadWorkflow()
	ctx := context.Background()
	swfClient := client.ScheduledWorkflow("default")

	err := swfClient.Delete(ctx, "anything", &v1.DeleteOptions{})
	if err == nil {
		t.Error("Delete() expected error for bad workflow client, got nil")
	}
}

func intPtr(v int64) *int64 { return &v }
