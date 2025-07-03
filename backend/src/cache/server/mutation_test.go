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

package server

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	fakePod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ArgoWorkflowNodeName: "test_node",
			},
			Labels: map[string]string{
				ArgoCompleteLabelKey:    "true",
				KFPCacheEnabledLabelKey: KFPCacheEnabledLabelValue,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "main",
					Image:   "test_image",
					Command: []string{"python"},
					Env: []corev1.EnvVar{{
						Name:  ArgoWorkflowTemplateEnvKey,
						Value: `{"name": "Does not matter","container":{"command":["echo", "Hello"],"image":"python:3.9"}}`,
					}},
				},
			},
			NodeSelector: map[string]string{"disktype": "ssd"},
		},
	}
	fakeAdmissionRequest = v1beta1.AdmissionRequest{
		UID: "test-12345",
		Kind: metav1.GroupVersionKind{
			Group:   "group",
			Version: "v1",
			Kind:    "k8s",
		},
		Resource: metav1.GroupVersionResource{
			Version:  "v1",
			Resource: "pods",
		},
		SubResource: "subresource",
		Name:        "test",
		Namespace:   "default",
		Operation:   "test",
		Object: runtime.RawExtension{
			Raw: EncodePod(fakePod),
		},
	}
)

func EncodePod(pod *corev1.Pod) []byte {
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(*pod)

	return reqBodyBytes.Bytes()
}

func GetFakeRequestFromPod(pod *corev1.Pod) *v1beta1.AdmissionRequest {
	fakeRequest := fakeAdmissionRequest
	fakeRequest.Object.Raw = EncodePod(pod)
	return &fakeRequest
}

func TestMain(m *testing.M) {
	os.Setenv("CACHE_NODE_RESTRICTIONS", "true")
	defer os.Unsetenv("CACHE_NODE_RESTRICTIONS")
	code := m.Run()
	os.Exit(code)
}

func TestMutatePodIfCachedWithErrorPodResource(t *testing.T) {
	mockAdmissionRequest := &v1beta1.AdmissionRequest{
		Resource: metav1.GroupVersionResource{
			Version: "wrong", Resource: "wrong",
		},
	}
	patchOperations, err := MutatePodIfCached(mockAdmissionRequest, fakeClientManager)
	assert.Nil(t, patchOperations)
	assert.Nil(t, err)
}

func TestMutatePodIfCachedWithDecodeError(t *testing.T) {
	invalidAdmissionRequest := fakeAdmissionRequest
	invalidAdmissionRequest.Object.Raw = []byte{5, 5}
	patchOperation, err := MutatePodIfCached(&invalidAdmissionRequest, fakeClientManager)
	assert.Nil(t, patchOperation)
	assert.Contains(t, err.Error(), "could not deserialize pod object")
}

func TestMutatePodIfCachedWithCacheDisabledPod(t *testing.T) {
	cacheDisabledPod := *fakePod.DeepCopy()
	cacheDisabledPod.ObjectMeta.Labels[KFPCacheEnabledLabelKey] = "false"
	patchOperation, err := MutatePodIfCached(GetFakeRequestFromPod(&cacheDisabledPod), fakeClientManager)
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestMutatePodIfCachedWithTFXPod(t *testing.T) {
	tfxPod := *fakePod.DeepCopy()
	mainContainerCommand := append(tfxPod.Spec.Containers[0].Command, "/tfx-src/"+TFXPodSuffix)
	tfxPod.Spec.Containers[0].Command = mainContainerCommand
	patchOperation, err := MutatePodIfCached(GetFakeRequestFromPod(&tfxPod), fakeClientManager)
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestMutatePodIfCachedWithTFXPod2(t *testing.T) {
	tfxPod := *fakePod.DeepCopy()
	tfxPod.Labels["pipelines.kubeflow.org/pipeline-sdk-type"] = "tfx"
	patchOperation, err := MutatePodIfCached(GetFakeRequestFromPod(&tfxPod), fakeClientManager)
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
	// test variation 2
	tfxPod = *fakePod.DeepCopy()
	tfxPod.Labels["pipelines.kubeflow.org/pipeline-sdk-type"] = "tfx-template"
	patchOperation, err = MutatePodIfCached(GetFakeRequestFromPod(&tfxPod), fakeClientManager)
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestMutatePodIfCachedWithKfpV2Pod(t *testing.T) {
	v2Pod := *fakePod.DeepCopy()
	v2Pod.Annotations["pipelines.kubeflow.org/v2_component"] = "true"
	patchOperation, err := MutatePodIfCached(GetFakeRequestFromPod(&v2Pod), fakeClientManager)
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestMutatePodIfCached(t *testing.T) {
	patchOperation, err := MutatePodIfCached(&fakeAdmissionRequest, fakeClientManager)
	assert.Nil(t, err)
	require.NotNil(t, patchOperation)
	require.Equal(t, 2, len(patchOperation))
	require.Equal(t, patchOperation[0].Op, OperationTypeAdd)
	require.Equal(t, patchOperation[1].Op, OperationTypeAdd)
}

func TestMutatePodIfCachedWithCacheEntryExist(t *testing.T) {
	executionCache := &model.ExecutionCache{
		ExecutionCacheKey: "1933d178a14bc415466cfd1b3ca2100af975e8c59e1ff9d502fcf18eb5cbd7f7",
		ExecutionOutput:   "testOutput",
		ExecutionTemplate: `{"container":{"command":["echo", "Hello"],"image":"python:3.9"}}`,
		MaxCacheStaleness: -1,
	}
	fakeClientManager.CacheStore().CreateExecutionCache(executionCache)

	patchOperation, err := MutatePodIfCached(&fakeAdmissionRequest, fakeClientManager)
	assert.Nil(t, err)

	require.NotNil(t, patchOperation)
	require.Equal(t, 3, len(patchOperation))
	require.Equal(t, patchOperation[0].Op, OperationTypeReplace)
	require.Equal(t, patchOperation[1].Op, OperationTypeAdd)
	require.Equal(t, patchOperation[2].Op, OperationTypeAdd)
}

func TestDefaultImage(t *testing.T) {
	executionCache := &model.ExecutionCache{
		ExecutionCacheKey: "1933d178a14bc415466cfd1b3ca2100af975e8c59e1ff9d502fcf18eb5cbd7f7",
		ExecutionOutput:   "testOutput",
		ExecutionTemplate: `{"container":{"command":["echo", "Hello"],"image":"python:3.9"}}`,
		MaxCacheStaleness: -1,
	}
	fakeClientManager.CacheStore().CreateExecutionCache(executionCache)

	patchOperation, err := MutatePodIfCached(&fakeAdmissionRequest, fakeClientManager)
	assert.Nil(t, err)
	container := patchOperation[0].Value.([]corev1.Container)[0]
	require.Equal(t, "ghcr.io/containerd/busybox", container.Image)
}

func TestSetImage(t *testing.T) {
	testImage := "testimage"
	os.Setenv("CACHE_IMAGE", testImage)
	defer os.Unsetenv("CACHE_IMAGE")

	executionCache := &model.ExecutionCache{
		ExecutionCacheKey: "f5fe913be7a4516ebfe1b5de29bcb35edd12ecc776b2f33f10ca19709ea3b2f0",
		ExecutionOutput:   "testOutput",
		ExecutionTemplate: `{"container":{"command":["echo", "Hello"],"image":"python:3.9"}}`,
		MaxCacheStaleness: -1,
	}
	fakeClientManager.CacheStore().CreateExecutionCache(executionCache)

	patchOperation, err := MutatePodIfCached(&fakeAdmissionRequest, fakeClientManager)
	assert.Nil(t, err)
	container := patchOperation[0].Value.([]corev1.Container)[0]
	assert.Equal(t, testImage, container.Image)
}

func TestCacheNodeRestriction(t *testing.T) {
	os.Setenv("CACHE_NODE_RESTRICTIONS", "false")

	executionCache := &model.ExecutionCache{
		ExecutionCacheKey: "f5fe913be7a4516ebfe1b5de29bcb35edd12ecc776b2f33f10ca19709ea3b2f0",
		ExecutionOutput:   "testOutput",
		ExecutionTemplate: `{"container":{"command":["echo", "Hello"],"image":"python:3.9"},"nodeSelector":{"disktype":"ssd"}}`,
		MaxCacheStaleness: -1,
	}
	fakeClientManager.CacheStore().CreateExecutionCache(executionCache)
	patchOperation, err := MutatePodIfCached(&fakeAdmissionRequest, fakeClientManager)
	assert.Nil(t, err)
	assert.Equal(t, OperationTypeRemove, patchOperation[1].Op)
	assert.Nil(t, patchOperation[1].Value)
	os.Setenv("CACHE_NODE_RESTRICTIONS", "true")
}

func TestMutatePodIfCachedWithTeamplateCleanup(t *testing.T) {
	executionCache := &model.ExecutionCache{
		ExecutionCacheKey: "c81988503d55a5817d79bd972017d95c37f72b024e522b4d79787d9f599c0725",
		ExecutionOutput:   "testOutput",
		ExecutionTemplate: `Cache key was calculated from this: {"container":{"command":["echo", "Hello"],"image":"python:3.9"},"outputs":"anything"}`,
		MaxCacheStaleness: -1,
	}
	fakeClientManager.CacheStore().CreateExecutionCache(executionCache)

	pod := *fakePod.DeepCopy()
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{
		Name: ArgoWorkflowTemplateEnvKey,
		Value: `{
			"name": "Does not matter",
			"metadata": "anything",
			"container": {
				"image": "python:3.9",
				"command": ["echo", "Hello"]
			},
			"outputs": "anything",
			"foo": "bar"
		}`,
	}}
	request := GetFakeRequestFromPod(&pod)

	patchOperation, err := MutatePodIfCached(request, fakeClientManager)
	assert.Nil(t, err)
	require.NotNil(t, patchOperation)
	require.Equal(t, 3, len(patchOperation))
	require.Equal(t, patchOperation[0].Op, OperationTypeReplace)
	require.Equal(t, patchOperation[1].Op, OperationTypeAdd)
	require.Equal(t, patchOperation[2].Op, OperationTypeAdd)
}
