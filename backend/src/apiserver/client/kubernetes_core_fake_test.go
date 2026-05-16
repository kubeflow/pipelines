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

	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
)

func TestFakeKuberneteCoreClient_PodClient(t *testing.T) {
	client := NewFakeKuberneteCoresClient()
	podClient := client.PodClient("default")
	if podClient == nil {
		t.Error("PodClient() returned nil for valid namespace")
	}
}

func TestFakeKuberneteCoreClient_PodClientPanicsOnEmptyNamespace(t *testing.T) {
	client := NewFakeKuberneteCoresClient()

	defer func() {
		if recover() == nil {
			t.Error("PodClient() did not panic for empty namespace")
		}
	}()
	client.PodClient("")
}

func TestFakeKuberneteCoreClient_GetClientSet(t *testing.T) {
	client := NewFakeKuberneteCoresClient()
	clientSet := client.GetClientSet()
	if clientSet != nil {
		t.Error("GetClientSet() expected nil for fake implementation, got non-nil")
	}
}

func TestFakeKubernetesCoreClientWithBadPodClient_PodClient(t *testing.T) {
	client := NewFakeKubernetesCoreClientWithBadPodClient()
	podClient := client.PodClient("default")
	if podClient == nil {
		t.Error("PodClient() returned nil for valid namespace")
	}
}

func TestFakeKubernetesCoreClientWithBadPodClient_GetClientSet(t *testing.T) {
	client := NewFakeKubernetesCoreClientWithBadPodClient()
	clientSet := client.GetClientSet()
	if clientSet != nil {
		t.Error("GetClientSet() expected nil for fake implementation, got non-nil")
	}
}

func TestFakePodClient_EvictV1(t *testing.T) {
	client := &FakePodClient{}
	err := client.EvictV1(context.Background(), &policyv1.Eviction{})
	if err != nil {
		t.Fatalf("EvictV1() unexpected error: %v", err)
	}
}

func TestFakePodClient_EvictV1beta1(t *testing.T) {
	client := &FakePodClient{}
	err := client.EvictV1beta1(context.Background(), &policyv1beta1.Eviction{})
	if err != nil {
		t.Fatalf("EvictV1beta1() unexpected error: %v", err)
	}
}
