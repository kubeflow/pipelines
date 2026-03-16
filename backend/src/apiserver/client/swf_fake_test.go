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
