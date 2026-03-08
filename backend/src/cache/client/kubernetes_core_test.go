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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesCorePodClient(t *testing.T) {
	fakeClientset := fakeclientset.NewSimpleClientset()
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}

	podClient := kubernetesCore.PodClient("test-namespace")
	assert.NotNil(t, podClient)
}

func TestKubernetesCorePodClientOperations(t *testing.T) {
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "busybox",
				},
			},
		},
	}

	fakeClientset := fakeclientset.NewSimpleClientset(testPod)
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}
	podClient := kubernetesCore.PodClient("test-namespace")

	// Get pod
	retrievedPod, err := podClient.Get(context.Background(), "test-pod", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "test-pod", retrievedPod.Name)
	assert.Equal(t, "test-namespace", retrievedPod.Namespace)

	// List pods
	podList, err := podClient.List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, podList.Items, 1)

	// Delete pod
	err = podClient.Delete(context.Background(), "test-pod", metav1.DeleteOptions{})
	assert.NoError(t, err)

	// Verify pod is deleted
	podList, err = podClient.List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, podList.Items, 0)
}

func TestKubernetesCorePodClientDifferentNamespaces(t *testing.T) {
	podInNamespaceA := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-a",
			Namespace: "namespace-a",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "busybox"}},
		},
	}
	podInNamespaceB := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-b",
			Namespace: "namespace-b",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "busybox"}},
		},
	}

	fakeClientset := fakeclientset.NewSimpleClientset(podInNamespaceA, podInNamespaceB)
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}

	// Namespace A should only see pod-a
	podsInA, err := kubernetesCore.PodClient("namespace-a").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, podsInA.Items, 1)
	assert.Equal(t, "pod-a", podsInA.Items[0].Name)

	// Namespace B should only see pod-b
	podsInB, err := kubernetesCore.PodClient("namespace-b").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, podsInB.Items, 1)
	assert.Equal(t, "pod-b", podsInB.Items[0].Name)

	// Empty namespace should see no pods
	podsInEmpty, err := kubernetesCore.PodClient("nonexistent").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, podsInEmpty.Items, 0)
}

func TestKubernetesCorePodClientCreateAndPatch(t *testing.T) {
	fakeClientset := fakeclientset.NewSimpleClientset()
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}
	podClient := kubernetesCore.PodClient("default")

	// Create a pod
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "busybox"}},
		},
	}
	createdPod, err := podClient.Create(context.Background(), newPod, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "new-pod", createdPod.Name)

	// Patch the pod
	patchData := []byte(`{"metadata":{"labels":{"cache-id":"abc123"}}}`)
	patchedPod, err := podClient.Patch(context.Background(), "new-pod", types.MergePatchType, patchData, metav1.PatchOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "abc123", patchedPod.Labels["cache-id"])
}

func TestKubernetesCoreImplementsInterface(t *testing.T) {
	fakeClientset := fakeclientset.NewSimpleClientset()
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}

	// Verify KubernetesCore satisfies KubernetesCoreInterface
	var _ KubernetesCoreInterface = kubernetesCore
}
