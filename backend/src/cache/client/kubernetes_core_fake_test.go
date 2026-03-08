// Copyright 2020 The Kubeflow Authors
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
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Verify it implements PodInterface
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
	result, err := fakePodClient.Patch(context.Background(), "test-pod", "merge-patch+json", []byte(`{}`), metav1.PatchOptions{})
	assert.Nil(t, result)
	assert.NoError(t, err)
}

func TestFakePodClientGet(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.Get(context.Background(), "test-pod", metav1.GetOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}

func TestFakePodClientList(t *testing.T) {
	fakePodClient := &FakePodClient{}
	podList, err := fakePodClient.List(context.Background(), metav1.ListOptions{})
	assert.Nil(t, podList)
	assert.NoError(t, err)
}

func TestFakePodClientCreate(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.Create(context.Background(), nil, metav1.CreateOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}

func TestFakePodClientUpdate(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.Update(context.Background(), nil, metav1.UpdateOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}

func TestFakePodClientUpdateStatus(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.UpdateStatus(context.Background(), nil, metav1.UpdateOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}

func TestFakePodClientDeleteCollection(t *testing.T) {
	fakePodClient := &FakePodClient{}
	err := fakePodClient.DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{})
	assert.NoError(t, err)
}

func TestFakePodClientBind(t *testing.T) {
	fakePodClient := &FakePodClient{}
	err := fakePodClient.Bind(context.Background(), nil, metav1.CreateOptions{})
	assert.NoError(t, err)
}

func TestFakePodClientEvict(t *testing.T) {
	fakePodClient := &FakePodClient{}
	err := fakePodClient.Evict(context.Background(), nil)
	assert.NoError(t, err)
}

func TestFakePodClientGetLogs(t *testing.T) {
	fakePodClient := &FakePodClient{}
	result := fakePodClient.GetLogs("test-pod", nil)
	assert.Nil(t, result)
}

func TestFakePodClientProxyGet(t *testing.T) {
	fakePodClient := &FakePodClient{}
	result := fakePodClient.ProxyGet("http", "test-pod", "8080", "/healthz", nil)
	assert.Nil(t, result)
}

func TestFakePodClientUpdateEphemeralContainers(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.UpdateEphemeralContainers(context.Background(), "test-pod", nil, metav1.UpdateOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}

func TestFakePodClientApply(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.Apply(context.Background(), nil, metav1.ApplyOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}

func TestFakePodClientApplyStatus(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.ApplyStatus(context.Background(), nil, metav1.ApplyOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}

func TestFakePodClientUpdateResize(t *testing.T) {
	fakePodClient := &FakePodClient{}
	pod, err := fakePodClient.UpdateResize(context.Background(), "test-pod", nil, metav1.UpdateOptions{})
	assert.Nil(t, pod)
	assert.NoError(t, err)
}
