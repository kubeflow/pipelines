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

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func TestNewFakeKuberneteCoresClient(t *testing.T) {
	fakeClient := NewFakeKuberneteCoresClient()
	assert.NotNil(t, fakeClient)
	assert.NotNil(t, fakeClient.podClientFake)
}

func TestFakeKuberneteCoreClientImplementsInterface(t *testing.T) {
	var _ KubernetesCoreInterface = NewFakeKuberneteCoresClient()
}

func TestFakeKuberneteCoreClientPodClientReturnsClient(t *testing.T) {
	fakeClient := NewFakeKuberneteCoresClient()
	podClient := fakeClient.PodClient("test-namespace")
	assert.NotNil(t, podClient)

	var _ v1.PodInterface = podClient
}

func TestFakeKuberneteCoreClientPodClientEmptyNamespacePanics(t *testing.T) {
	fakeClient := NewFakeKuberneteCoresClient()
	assert.Panics(t, func() {
		fakeClient.PodClient("")
	})
}

func TestNewFakeKubernetesCoreClientWithBadPodClient(t *testing.T) {
	fakeClient := NewFakeKubernetesCoreClientWithBadPodClient()
	assert.NotNil(t, fakeClient)
	assert.NotNil(t, fakeClient.podClientFake)
}

func TestFakeKubernetesCoreClientWithBadPodClientImplementsInterface(t *testing.T) {
	var _ KubernetesCoreInterface = NewFakeKubernetesCoreClientWithBadPodClient()
}

func TestFakeKubernetesCoreClientWithBadPodClientReturnsClient(t *testing.T) {
	fakeClient := NewFakeKubernetesCoreClientWithBadPodClient()
	podClient := fakeClient.PodClient("test-namespace")
	assert.NotNil(t, podClient)
}

func TestFakeBadPodClientDeleteFails(t *testing.T) {
	fakeClient := NewFakeKubernetesCoreClientWithBadPodClient()
	podClient := fakeClient.PodClient("test-namespace")
	err := podClient.Delete(context.Background(), "some-pod", metav1.DeleteOptions{})
	assert.Error(t, err)
	assert.Equal(t, "failed to delete pod", err.Error())
}

func TestFakePodClientDeleteSucceeds(t *testing.T) {
	fakeClient := NewFakeKuberneteCoresClient()
	podClient := fakeClient.PodClient("test-namespace")
	err := podClient.Delete(context.Background(), "some-pod", metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func TestFakePodClientEvictV1(t *testing.T) {
	fakePodClient := &FakePodClient{}
	err := fakePodClient.EvictV1(context.Background(), &policyv1.Eviction{})
	assert.NoError(t, err)
}

func TestFakePodClientEvictV1beta1(t *testing.T) {
	fakePodClient := &FakePodClient{}
	err := fakePodClient.EvictV1beta1(context.Background(), &policyv1beta1.Eviction{})
	assert.NoError(t, err)
}

func TestFakePodClientWatch(t *testing.T) {
	fakePodClient := &FakePodClient{}
	watcher, err := fakePodClient.Watch(context.Background(), metav1.ListOptions{})
	assert.Nil(t, watcher)
	assert.NoError(t, err)
}

func TestFakePodClientPatch(t *testing.T) {
	fakePodClient := &FakePodClient{}
	result, err := fakePodClient.Patch(context.Background(), "test-pod", types.MergePatchType, []byte(`{}`), metav1.PatchOptions{})
	assert.Nil(t, result)
	assert.NoError(t, err)
}

// TestFakePodClientStubMethods verifies all unimplemented stub methods return nil values.
func TestFakePodClientStubMethods(t *testing.T) {
	fakePodClient := &FakePodClient{}
	ctx := context.Background()
	minimalPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}}

	t.Run("Get", func(t *testing.T) {
		pod, err := fakePodClient.Get(ctx, "test-pod", metav1.GetOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})

	t.Run("List", func(t *testing.T) {
		podList, err := fakePodClient.List(ctx, metav1.ListOptions{})
		assert.Nil(t, podList)
		assert.NoError(t, err)
	})

	t.Run("Create", func(t *testing.T) {
		pod, err := fakePodClient.Create(ctx, minimalPod, metav1.CreateOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})

	t.Run("Update", func(t *testing.T) {
		pod, err := fakePodClient.Update(ctx, minimalPod, metav1.UpdateOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		pod, err := fakePodClient.UpdateStatus(ctx, minimalPod, metav1.UpdateOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})

	t.Run("DeleteCollection", func(t *testing.T) {
		err := fakePodClient.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
		assert.NoError(t, err)
	})

	t.Run("Bind", func(t *testing.T) {
		binding := &corev1.Binding{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
			Target:     corev1.ObjectReference{Name: "test-node"},
		}
		err := fakePodClient.Bind(ctx, binding, metav1.CreateOptions{})
		assert.NoError(t, err)
	})

	t.Run("Evict", func(t *testing.T) {
		err := fakePodClient.Evict(ctx, nil)
		assert.NoError(t, err)
	})

	t.Run("GetLogs", func(t *testing.T) {
		result := fakePodClient.GetLogs("test-pod", nil)
		assert.Nil(t, result)
	})

	t.Run("ProxyGet", func(t *testing.T) {
		result := fakePodClient.ProxyGet("http", "test-pod", "8080", "/healthz", nil)
		assert.Nil(t, result)
	})

	t.Run("UpdateEphemeralContainers", func(t *testing.T) {
		pod, err := fakePodClient.UpdateEphemeralContainers(ctx, "test-pod", minimalPod, metav1.UpdateOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})

	t.Run("Apply", func(t *testing.T) {
		pod, err := fakePodClient.Apply(ctx, nil, metav1.ApplyOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})

	t.Run("ApplyStatus", func(t *testing.T) {
		pod, err := fakePodClient.ApplyStatus(ctx, nil, metav1.ApplyOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})

	t.Run("UpdateResize", func(t *testing.T) {
		pod, err := fakePodClient.UpdateResize(ctx, "test-pod", minimalPod, metav1.UpdateOptions{})
		assert.Nil(t, pod)
		assert.NoError(t, err)
	})
}
