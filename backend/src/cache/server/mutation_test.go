// Copyright 2020 Google LLC
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
				ArgoWorkflowTemplate: `{"name": "Does not matter","container":{"command":["echo", "Hello"],"image":"python:3.7"}}`,
			},
			Labels: map[string]string{
				ArgoCompleteLabelKey:    "true",
				KFPCacheEnabledLabelKey: KFPCacheEnabledLabelValue,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:    "main",
					Image:   "test_image",
					Command: []string{"python"},
				},
			},
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
		ExecutionCacheKey: "f5fe913be7a4516ebfe1b5de29bcb35edd12ecc776b2f33f10ca19709ea3b2f0",
		ExecutionOutput:   "testOutput",
		ExecutionTemplate: `{"container":{"command":["echo", "Hello"],"image":"python:3.7"}}`,
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

func TestMutatePodIfCachedWithTeamplateCleanup(t *testing.T) {
	executionCache := &model.ExecutionCache{
		ExecutionCacheKey: "f5fe913be7a4516ebfe1b5de29bcb35edd12ecc776b2f33f10ca19709ea3b2f0",
		ExecutionOutput:   "testOutput",
		ExecutionTemplate: `Cache key was calculated from this: {"container":{"command":["echo", "Hello"],"image":"python:3.7"}}`,
		MaxCacheStaleness: -1,
	}
	fakeClientManager.CacheStore().CreateExecutionCache(executionCache)

	pod := *fakePod.DeepCopy()
	pod.ObjectMeta.Annotations[ArgoWorkflowTemplate] = `{
		"name": "Does not matter",
		"metadata": "anything",
		"container": {
			"image": "python:3.7",
			"command": ["echo", "Hello"]
		},
		"outputs": "anything",
		"foo": "bar"
	}`
	request := GetFakeRequestFromPod(&pod)

	patchOperation, err := MutatePodIfCached(request, fakeClientManager)
	assert.Nil(t, err)
	require.NotNil(t, patchOperation)
	require.Equal(t, 3, len(patchOperation))
	require.Equal(t, patchOperation[0].Op, OperationTypeReplace)
	require.Equal(t, patchOperation[1].Op, OperationTypeAdd)
	require.Equal(t, patchOperation[2].Op, OperationTypeAdd)
}
