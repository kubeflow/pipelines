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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestFakePodClient_Stubs verifies that every unimplemented stub method on
// FakePodClient returns nil without error. These are interface-satisfaction
// stubs — a single table-driven test keeps coverage without per-method bloat.
func TestFakePodClient_Stubs(t *testing.T) {
	client := FakePodClient{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Create", func() error { _, e := client.Create(ctx, &corev1.Pod{}, v1.CreateOptions{}); return e }},
		{"Get", func() error { _, e := client.Get(ctx, "p", v1.GetOptions{}); return e }},
		{"Update", func() error { _, e := client.Update(ctx, &corev1.Pod{}, v1.UpdateOptions{}); return e }},
		{"UpdateStatus", func() error { _, e := client.UpdateStatus(ctx, &corev1.Pod{}, v1.UpdateOptions{}); return e }},
		{"UpdateEphemeralContainers", func() error {
			_, e := client.UpdateEphemeralContainers(ctx, "p", &corev1.Pod{}, v1.UpdateOptions{})
			return e
		}},
		{"UpdateResize", func() error {
			_, e := client.UpdateResize(ctx, "p", &corev1.Pod{}, v1.UpdateOptions{})
			return e
		}},
		{"Patch", func() error {
			_, e := client.Patch(ctx, "p", types.MergePatchType, []byte(`{}`), v1.PatchOptions{})
			return e
		}},
		{"Apply", func() error { _, e := client.Apply(ctx, nil, v1.ApplyOptions{}); return e }},
		{"ApplyStatus", func() error { _, e := client.ApplyStatus(ctx, nil, v1.ApplyOptions{}); return e }},
		{"List", func() error { _, e := client.List(ctx, v1.ListOptions{}); return e }},
		{"Watch", func() error { _, e := client.Watch(ctx, v1.ListOptions{}); return e }},
		{"Delete", func() error { return client.Delete(ctx, "p", v1.DeleteOptions{}) }},
		{"DeleteCollection", func() error { return client.DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{}) }},
		{"Bind", func() error { return client.Bind(ctx, &corev1.Binding{}, v1.CreateOptions{}) }},
		{"Evict", func() error { return client.Evict(ctx, &v1beta1.Eviction{}) }},
		{"GetLogs", func() error {
			client.GetLogs("p", &corev1.PodLogOptions{})
			return nil
		}},
		{"ProxyGet", func() error {
			client.ProxyGet("http", "p", "8080", "/", nil)
			return nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestFakeBadPodClient_Delete(t *testing.T) {
	client := FakeBadPodClient{}
	err := client.Delete(context.Background(), "test-pod", v1.DeleteOptions{})
	if err == nil {
		t.Error("FakeBadPodClient.Delete() expected error, got nil")
	}
	if err.Error() != "failed to delete pod" {
		t.Errorf("FakeBadPodClient.Delete() error = %q, want %q", err.Error(), "failed to delete pod")
	}
}
